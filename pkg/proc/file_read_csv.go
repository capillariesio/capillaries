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

func readCsv(envConfig *env.EnvConfig, logger *l.Logger, pCtx *ctx.MessageProcessingContext, totalStartTime time.Time, filePath string, fileReader io.Reader) (BatchStats, error) {
	bs := BatchStats{RowsRead: 0, RowsWritten: 0}
	node := pCtx.CurrentScriptNode

	r := csv.NewReader(fileReader)
	r.Comma = rune(node.FileReader.Csv.Separator[0])

	// To avoid bare \" error: https://stackoverflow.com/questions/31326659/golang-csv-error-bare-in-non-quoted-field
	r.LazyQuotes = true

	var lineIdx int64 = 0
	tableRecordBatchCount := 0

	instr := newTableInserter(envConfig, logger, pCtx, &node.TableCreator, DefaultInserterBatchSize)
	//instr.verifyTablesExist()
	if err := instr.startWorkers(logger, pCtx); err != nil {
		return bs, err
	}
	defer instr.waitForWorkersAndCloseErrorsOut(logger, pCtx)

	batchStartTime := time.Now()

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
				if err = instr.add(tableRecord); err == nil {
					tableRecordBatchCount++
					if tableRecordBatchCount == DefaultInserterBatchSize {
						if err := instr.waitForWorkers(logger, pCtx); err != nil {
							return bs, fmt.Errorf("cannot save record to batch of size %d to %s: [%s]", tableRecordBatchCount, node.TableCreator.Name, err.Error())
						}
						reportWriteTable(logger, pCtx, tableRecordBatchCount, time.Since(batchStartTime), len(node.TableCreator.Indexes), instr.NumWorkers)
						batchStartTime = time.Now()
						tableRecordBatchCount = 0
						if err := instr.startWorkers(logger, pCtx); err != nil {
							return bs, err
						}

					}
					bs.RowsWritten++
				} else {
					return bs, fmt.Errorf("cannot add record to batch of size %d to %s: [%s]", tableRecordBatchCount, node.TableCreator.Name, err.Error())
				}
			}
			bs.RowsRead++
		}
		lineIdx++
	}

	// Write leftovers regardless of tableRecordBatchCount == 0
	if err := instr.waitForWorkers(logger, pCtx); err != nil {
		return bs, fmt.Errorf("cannot save leftover record batch of size %d to %s: [%s]", tableRecordBatchCount, node.TableCreator.Name, err.Error())
	}
	reportWriteTableLeftovers(logger, pCtx, tableRecordBatchCount, time.Since(batchStartTime), len(node.TableCreator.Indexes), instr.NumWorkers)

	bs.Elapsed = time.Since(totalStartTime)
	reportWriteTableComplete(logger, pCtx, bs.RowsRead, bs.RowsWritten, bs.Elapsed, len(node.TableCreator.Indexes), instr.NumWorkers)

	return bs, nil
}
