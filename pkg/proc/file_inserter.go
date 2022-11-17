package proc

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/capillariesio/capillaries/pkg/ctx"
	"github.com/capillariesio/capillaries/pkg/l"
	"github.com/capillariesio/capillaries/pkg/sc"
	"github.com/shopspring/decimal"
)

type FileInserter struct {
	PCtx          *ctx.MessageProcessingContext
	FileCreator   *sc.FileCreatorDef
	CurrentBatch  *WriteFileBatch
	BatchCapacity int
	BatchesIn     chan *WriteFileBatch
	ErrorsOut     chan error
	BatchesSent   int
	BatchesInOpen bool
}

const DefaultFileInserterBatchCapacity int = 1000

type WriteFileBatch struct {
	Rows     [][]interface{}
	RowCount int
}

func newWriteFileBatch(batchCapacity int) *WriteFileBatch {
	return &WriteFileBatch{
		Rows:     make([][]interface{}, batchCapacity),
		RowCount: 0,
	}
}

func newFileInserter(pCtx *ctx.MessageProcessingContext, fileCreator *sc.FileCreatorDef) *FileInserter {
	instr := FileInserter{
		PCtx:          pCtx,
		FileCreator:   fileCreator,
		BatchCapacity: DefaultFileInserterBatchCapacity,
		BatchesIn:     make(chan *WriteFileBatch, sc.MaxFileCreatorTopLimit/DefaultFileInserterBatchCapacity),
		ErrorsOut:     make(chan error, 1),
		BatchesSent:   0,
		BatchesInOpen: true,
	}

	return &instr
}

func (instr *FileInserter) createFileAndStartWorker(logger *l.Logger, runId int16, batchIdx int16) error {
	logger.PushF("proc.createFileAndStartWorker")
	defer logger.PopF()

	// Templated file name
	actualFileUrl := strings.ReplaceAll(strings.ReplaceAll(instr.FileCreator.UrlTemplate, sc.ReservedParamRunId, fmt.Sprintf("%05d", runId)), sc.ReservedParamBatchIdx, fmt.Sprintf("%05d", batchIdx))
	f, err := os.Create(actualFileUrl)
	if err != nil {
		return err
	}

	// Header
	b := strings.Builder{}
	for i := 0; i < len(instr.FileCreator.Columns); i++ {
		if strings.Contains(instr.FileCreator.Columns[i].Header, ",") {
			b.WriteString("\"")
		}
		b.WriteString(instr.FileCreator.Columns[i].Header)
		if strings.Contains(instr.FileCreator.Columns[i].Header, ",") {
			b.WriteString("\"")
		}
		if i == len(instr.FileCreator.Columns)-1 {
			b.WriteString("\r\n")
		} else {
			b.WriteString(instr.FileCreator.Separator)
		}
	}
	if _, err := f.WriteString(b.String()); err != nil {
		return fmt.Errorf("cannot write file [%s] header line: [%s]", actualFileUrl, err.Error())
	}

	f.Close()

	newLogger, err := l.NewLoggerFromLogger(logger)
	if err != nil {
		return err
	}
	go instr.fileInserterWorker(newLogger, actualFileUrl, instr.FileCreator.Separator)

	return nil
}

func (instr *FileInserter) checkWorkerOutput() error {
	errors := make([]string, 0)
	for {
		select {
		case err := <-instr.ErrorsOut:
			instr.BatchesSent--
			if err != nil {
				errors = append(errors, err.Error())
			}
		default:
			if len(errors) > 0 {
				return fmt.Errorf(strings.Join(errors, "; "))
			} else {
				return nil
			}
		}
	}
}

func (instr *FileInserter) waitForWorker() error {

	if instr.CurrentBatch != nil && instr.CurrentBatch.RowCount > 0 {
		instr.BatchesIn <- instr.CurrentBatch
		instr.CurrentBatch = nil
		instr.BatchesSent++
	}

	if instr.BatchesInOpen {
		close(instr.BatchesIn)
		instr.BatchesInOpen = false
	}

	errors := make([]string, 0)
	for {
		if instr.BatchesSent == 0 {
			break
		}
		err := <-instr.ErrorsOut
		instr.BatchesSent--
		if err != nil {
			errors = append(errors, err.Error())
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf(strings.Join(errors, "; "))
	} else {
		return nil
	}
}

func (instr *FileInserter) waitForWorkerAndClose() {
	instr.waitForWorker()
	close(instr.ErrorsOut)
}

func (instr *FileInserter) add(row []interface{}) {
	if instr.CurrentBatch == nil {
		instr.CurrentBatch = newWriteFileBatch(instr.BatchCapacity)
	}
	instr.CurrentBatch.Rows[instr.CurrentBatch.RowCount] = row
	instr.CurrentBatch.RowCount++

	if instr.CurrentBatch.RowCount == instr.BatchCapacity {
		instr.BatchesIn <- instr.CurrentBatch
		instr.CurrentBatch = nil
		instr.BatchesSent++
	}
}

func (instr *FileInserter) fileInserterWorker(logger *l.Logger, actualFileUrl string, separator string) {
	logger.PushF("proc.fileInserterWorker")
	defer logger.PopF()

	f, err := os.OpenFile(actualFileUrl, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		instr.ErrorsOut <- fmt.Errorf("cannot open %s for appending: [%s]", actualFileUrl, err.Error())
	} else {
		defer f.Close()
	}

	for batch := range instr.BatchesIn {
		batchStartTime := time.Now()
		b := strings.Builder{}
		for rowIdx := 0; rowIdx < batch.RowCount; rowIdx++ {
			for i := 0; i < len(instr.FileCreator.Columns); i++ {
				var stringVal string
				switch assertedVal := batch.Rows[rowIdx][i].(type) {
				case time.Time:
					stringVal = assertedVal.Format(instr.FileCreator.Columns[i].Format)
				case decimal.Decimal:
					stringVal = fmt.Sprintf(instr.FileCreator.Columns[i].Format, assertedVal.StringFixed(2))
				default:
					stringVal = fmt.Sprintf(instr.FileCreator.Columns[i].Format, batch.Rows[rowIdx][i])
				}

				isQuote := strings.Contains(stringVal, ",")
				if isQuote {
					b.WriteString("\"")
				}
				b.WriteString(stringVal)
				if isQuote {
					b.WriteString("\"")
				}
				if i == len(instr.FileCreator.Columns)-1 {
					b.WriteString("\n")
				} else {
					b.WriteString(separator)
				}
			}
		}

		f.Sync()
		if _, err := f.WriteString(b.String()); err != nil {
			instr.ErrorsOut <- fmt.Errorf("cannot write string to %s: [%s]", actualFileUrl, err.Error())
		} else {
			instr.ErrorsOut <- nil
		}
		dur := time.Since(batchStartTime)
		logger.InfoCtx(instr.PCtx, "%d items in %.3fs (%.0f items/s)", batch.RowCount, dur.Seconds(), float64(batch.RowCount)/dur.Seconds())
	}
}
