package proc

import (
	"encoding/csv"
	"fmt"
	"io"
	"time"

	"github.com/capillariesio/capillaries/pkg/ctx"
	"github.com/capillariesio/capillaries/pkg/env"
	"github.com/capillariesio/capillaries/pkg/eval"
	"github.com/capillariesio/capillaries/pkg/l"
	"github.com/capillariesio/capillaries/pkg/sc"
)

func readCsv(envConfig *env.EnvConfig, logger *l.CapiLogger, pCtx *ctx.MessageProcessingContext, totalStartTime time.Time, filePath string, fileReader io.Reader) (BatchStats, error) {
	node := pCtx.CurrentScriptNode
	bs := BatchStats{RowsRead: 0, RowsWritten: 0, Src: filePath, Dst: node.TableCreator.Name}

	r := csv.NewReader(fileReader)
	r.Comma = rune(node.FileReader.Csv.Separator[0])

	// To avoid bare \" error: https://stackoverflow.com/questions/31326659/golang-csv-error-bare-in-non-quoted-field
	r.LazyQuotes = true

	var lineIdx int64
	// tableRecordBatchCount := 0

	instr, err := createInserterAndStartWorkers(logger, envConfig, pCtx, &node.TableCreator, ReadFileTableInserterBatchSize, DataIdxSeqModeDataFirst, logger.ZapMachine.String)
	if err != nil {
		return bs, err
	}
	defer instr.letWorkersDrainRecordWrittenStatusesAndCloseInserter(logger, pCtx)

	// batchStartTime := time.Now()

	for {
		line, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return bs, fmt.Errorf("cannot read file [%s]: [%s]", filePath, err.Error())
		}
		if node.FileReader.Csv.ColumnIndexingMode == sc.FileColumnIndexingName && int64(node.FileReader.Csv.SrcFileHdrLineIdx) == lineIdx {
			if err := node.FileReader.ResolveCsvColumnIndexesFromNames(line); err != nil {
				return bs, fmt.Errorf("cannot parse column headers of [%s]: [%s]", filePath, err.Error())
			}
		} else if lineIdx >= int64(node.FileReader.Csv.SrcFileFirstDataLineIdx) {

			// FileReader: read columns
			colVars := eval.VarValuesMap{}
			if err := node.FileReader.ReadCsvLineToValuesMap(&line, colVars); err != nil {
				return bs, fmt.Errorf("cannot read values from [%s], line %d: [%s]", filePath, lineIdx, err.Error())
			}

			// TableCreator: evaluate table column expressions
			tableRecord, err := node.TableCreator.CalculateTableRecordFromSrcVars(false, colVars)
			if err != nil {
				return bs, fmt.Errorf("cannot populate table record from [%s], line %d: [%s]", filePath, lineIdx, err.Error())
			}

			// Check table creator having
			inResult, err := node.TableCreator.CheckTableRecordHavingCondition(tableRecord)
			if err != nil {
				return bs, fmt.Errorf("cannot check having condition [%s], table record [%v]: [%s]", node.TableCreator.RawHaving, tableRecord, err.Error())
			}

			// Write batch if needed
			if inResult {
				indexKeyMap, err := instr.buildIndexKeys(tableRecord)
				if err != nil {
					return bs, fmt.Errorf("cannot build index keys for %s: [%s]", node.TableCreator.Name, err.Error())
				}

				if len(instr.RecordWrittenStatuses) == cap(instr.RecordWrittenStatuses) {
					if err := instr.letWorkersDrainRecordWrittenStatuses(logger, pCtx); err != nil {
						return bs, err
					}
				}
				if err := instr.add(logger, pCtx, tableRecord, indexKeyMap); err != nil {
					return bs, fmt.Errorf("cannot add record to inserter %s: [%s]", node.TableCreator.Name, err.Error())
				}

				bs.RowsWritten++
			}
			bs.RowsRead++
		}
		lineIdx++
	}

	// Write leftovers if anything was sent at all
	if instr.RecordsSent > 0 {
		if err := instr.letWorkersDrainRecordWrittenStatuses(logger, pCtx); err != nil {
			return bs, err
		}
	}

	// reportWriteTableLeftovers(logger, pCtx, tableRecordBatchCount, time.Since(batchStartTime), len(node.TableCreator.Indexes), instr.NumWorkers)

	bs.Elapsed = time.Since(totalStartTime)
	reportWriteTableComplete(logger, pCtx, bs.RowsRead, bs.RowsWritten, bs.Elapsed, len(node.TableCreator.Indexes), instr.NumWorkers)

	return bs, nil
}
