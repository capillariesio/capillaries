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

	var lineIdx int64 // CSV file line idx, includes headers

	instr, err := createInserterAndStartWorkers(logger, envConfig, pCtx, &node.TableCreator, DataIdxSeqModeDataFirst, logger.ZapMachine.String)
	if err != nil {
		return bs, err
	}
	instr.startDrainer()
	defer instr.closeInserter(logger, pCtx)

	for {
		line, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			instr.cancelDrainer(fmt.Errorf("cannot read csv file [%s]: [%s]", filePath, err.Error()))
			return bs, instr.waitForDrainer(logger, pCtx)
		}
		if node.FileReader.Csv.ColumnIndexingMode == sc.FileColumnIndexingName && int64(node.FileReader.Csv.SrcFileHdrLineIdx) == lineIdx {
			if err := node.FileReader.ResolveCsvColumnIndexesFromNames(line); err != nil {
				instr.cancelDrainer(fmt.Errorf("cannot parse column headers of csv file [%s]: [%s]", filePath, err.Error()))
				return bs, instr.waitForDrainer(logger, pCtx)
			}
		} else if lineIdx >= int64(node.FileReader.Csv.SrcFileFirstDataLineIdx) {

			// FileReader: read columns
			colVars := eval.VarValuesMap{}
			if err := node.FileReader.ReadCsvLineToValuesMap(&line, colVars); err != nil {
				instr.cancelDrainer(fmt.Errorf("cannot read values from csv file [%s], line %d: [%s]", filePath, lineIdx, err.Error()))
				return bs, instr.waitForDrainer(logger, pCtx)
			}

			// TableCreator: evaluate table column expressions
			tableRecord, err := node.TableCreator.CalculateTableRecordFromSrcVars(false, colVars)
			if err != nil {
				instr.cancelDrainer(fmt.Errorf("cannot populate table record from csv file [%s], line %d: [%s]", filePath, lineIdx, err.Error()))
				return bs, instr.waitForDrainer(logger, pCtx)
			}

			// Check table creator having
			inResult, err := node.TableCreator.CheckTableRecordHavingCondition(tableRecord)
			if err != nil {
				instr.cancelDrainer(fmt.Errorf("cannot check having condition [%s], csv file [%s] line %d, table record [%v]: [%s]", node.TableCreator.RawHaving, filePath, lineIdx, tableRecord, err.Error()))
				return bs, instr.waitForDrainer(logger, pCtx)
			}

			if inResult {
				indexKeyMap, err := instr.buildIndexKeys(tableRecord)
				if err != nil {
					instr.cancelDrainer(fmt.Errorf("cannot build index keys for table %s, csv file [%s] line %d: [%s]", node.TableCreator.Name, filePath, lineIdx, err.Error()))
					return bs, instr.waitForDrainer(logger, pCtx)
				}

				instr.add(tableRecord, indexKeyMap)
				bs.RowsWritten++
			}
			bs.RowsRead++
		}
		lineIdx++
	}

	instr.doneSending()
	if err := instr.waitForDrainer(logger, pCtx); err != nil {
		return bs, err
	}

	bs.Elapsed = time.Since(totalStartTime)
	reportWriteTableComplete(logger, pCtx, bs.RowsRead, bs.RowsWritten, bs.Elapsed, len(node.TableCreator.Indexes), instr.NumWorkers)

	return bs, nil
}
