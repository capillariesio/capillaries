package proc

import (
	"container/heap"
	"fmt"
	"time"

	"github.com/capillariesio/capillaries/pkg/ctx"
	"github.com/capillariesio/capillaries/pkg/eval"
	"github.com/capillariesio/capillaries/pkg/l"
	"github.com/capillariesio/capillaries/pkg/sc"
)

type FileRecordHeapItem struct {
	FileRecord *[]interface{}
	Key        string
}

type FileRecordHeap []*FileRecordHeapItem

func (h FileRecordHeap) Len() int           { return len(h) }
func (h FileRecordHeap) Less(i, j int) bool { return h[i].Key > h[j].Key } // Reverse order: https://stackoverflow.com/questions/49065781/limit-size-of-the-priority-queue-for-gos-heap-interface-implementation
func (h FileRecordHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }
func (h *FileRecordHeap) Push(x interface{}) {
	item := x.(*FileRecordHeapItem)
	*h = append(*h, item)
}
func (h *FileRecordHeap) Pop() interface{} {
	old := *h
	n := len(old)
	item := old[n-1]
	old[n-1] = nil // avoid memory leak
	*h = old[0 : n-1]
	return item
}

func RunCreateFile(logger *l.Logger,
	pCtx *ctx.MessageProcessingContext,
	readerNodeRunId int16,
	startToken int64,
	endToken int64) (BatchStats, error) {

	logger.PushF("proc.RunCreateFile")
	defer logger.PopF()

	totalStartTime := time.Now()
	bs := BatchStats{RowsRead: 0, RowsWritten: 0}

	if readerNodeRunId == 0 {
		return bs, fmt.Errorf("this node has a dependency node to read data from that was never started in this keyspace (readerNodeRunId == 0)")
	}

	node := pCtx.CurrentScriptNode

	if !node.HasFileCreator() {
		return bs, fmt.Errorf("node does not have file creator")
	}

	// Fields to read from source table
	srcFieldRefs := sc.FieldRefs{}
	// No src fields in having!
	srcFieldRefs.AppendWithFilter(node.FileCreator.UsedInTargetExpressionsFields, sc.ReaderAlias)

	srcBatchSize := node.TableReader.RowsetSize
	tableRecordBatchCount := 0
	curStartToken := startToken

	rs := NewRowsetFromFieldRefs(
		sc.FieldRefs{sc.RowidFieldRef(node.TableReader.TableName)},
		sc.FieldRefs{sc.RowidTokenFieldRef()},
		srcFieldRefs)

	instr := newFileInserter(pCtx, &node.FileCreator)
	if err := instr.createFileAndStartWorker(logger, pCtx.BatchInfo.RunId, pCtx.BatchInfo.BatchIdx); err != nil {
		return bs, fmt.Errorf("cannot start file inserter worker: %s", err.Error())
	}
	defer instr.waitForWorkerAndClose(logger, pCtx)

	var topHeap FileRecordHeap
	if node.FileCreator.HasTop() {
		topHeap := FileRecordHeap{}
		heap.Init(&topHeap)
	}

	for {
		lastRetrievedToken, err := selectBatchFromTableByToken(logger,
			pCtx,
			rs,
			node.TableReader.TableName,
			readerNodeRunId,
			srcBatchSize,
			curStartToken,
			endToken)
		if err != nil {
			return bs, err
		}
		curStartToken = lastRetrievedToken + 1

		if rs.RowCount == 0 {
			break
		}

		for rowIdx := 0; rowIdx < rs.RowCount; rowIdx++ {
			vars := eval.VarValuesMap{}
			if err := rs.ExportToVars(rowIdx, &vars); err != nil {
				return bs, err
			}

			fileRecord, err := node.FileCreator.CalculateFileRecordFromSrcVars(vars)
			if err != nil {
				return bs, fmt.Errorf("cannot populate file record from [%v]: [%s]", vars, err.Error())
			}

			inResult, err := node.FileCreator.CheckFileRecordHavingCondition(fileRecord)
			if err != nil {
				return bs, fmt.Errorf("cannot check having condition [%s], file record [%v]: [%s]", node.FileCreator.RawHaving, fileRecord, err.Error())
			}

			if !inResult {
				continue
			}

			if node.FileCreator.HasTop() {
				keyVars := map[string]interface{}{}
				for i := 0; i < len(node.FileCreator.Columns); i++ {
					keyVars[node.FileCreator.Columns[i].Name] = fileRecord[i]
				}
				key, err := sc.BuildKey(keyVars, &node.FileCreator.Top.OrderIdxDef)
				if err != nil {
					return bs, fmt.Errorf("cannot build top key for [%v]: [%s]", vars, err.Error())
				}
				heap.Push(&topHeap, &FileRecordHeapItem{FileRecord: &fileRecord, Key: key})
				if len(topHeap) > node.FileCreator.Top.Limit {
					heap.Pop(&topHeap)
				}
			} else {
				instr.add(fileRecord)
				bs.RowsWritten++
			}
		}

		bs.RowsRead += rs.RowCount
		if rs.RowCount < srcBatchSize {
			break
		}

		if err := instr.checkWorkerOutput(); err != nil {
			return bs, fmt.Errorf("cannot save record batch of size %d to %s: [%s]", tableRecordBatchCount, node.TableCreator.Name, err.Error())
		}

	} // for each source table batch

	if node.FileCreator.HasTop() {
		properlyOrderedTopList := make([]*FileRecordHeapItem, topHeap.Len())
		for i := topHeap.Len() - 1; i >= 0; i-- {
			properlyOrderedTopList[i] = heap.Pop(&topHeap).(*FileRecordHeapItem)
		}
		for i := 0; i < len(properlyOrderedTopList); i++ {
			instr.add(*properlyOrderedTopList[i].FileRecord)
			bs.RowsWritten++
		}
	}

	// Successful so far, write leftovers
	if err := instr.waitForWorker(logger, pCtx); err != nil {
		return bs, fmt.Errorf("cannot save record batch of size %d to %s: [%s]", tableRecordBatchCount, node.TableCreator.Name, err.Error())
	}

	bs.Elapsed = time.Since(totalStartTime)
	logger.InfoCtx(pCtx, "WriteFileComplete: read %d, wrote %d items in %.3fs (%.0f items/s)", bs.RowsRead, bs.RowsWritten, bs.Elapsed.Seconds(), float64(bs.RowsWritten)/bs.Elapsed.Seconds())

	return bs, nil
}
