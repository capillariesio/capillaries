package proc

import (
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/capillariesio/capillaries/pkg/ctx"
	"github.com/capillariesio/capillaries/pkg/l"
	"github.com/capillariesio/capillaries/pkg/sc"
	"github.com/capillariesio/capillaries/pkg/xfer"
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
	FinalFileUrl  string
	TempFilePath  string
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

func newFileInserter(pCtx *ctx.MessageProcessingContext, fileCreator *sc.FileCreatorDef, runId int16, batchIdx int16) *FileInserter {
	instr := FileInserter{
		PCtx:          pCtx,
		FileCreator:   fileCreator,
		BatchCapacity: DefaultFileInserterBatchCapacity,
		BatchesIn:     make(chan *WriteFileBatch, sc.MaxFileCreatorTopLimit/DefaultFileInserterBatchCapacity),
		ErrorsOut:     make(chan error, 1),
		BatchesSent:   0,
		BatchesInOpen: true,
		FinalFileUrl:  strings.ReplaceAll(strings.ReplaceAll(fileCreator.UrlTemplate, sc.ReservedParamRunId, fmt.Sprintf("%05d", runId)), sc.ReservedParamBatchIdx, fmt.Sprintf("%05d", batchIdx)),
	}

	return &instr
}

func (instr *FileInserter) createFileAndStartWorker(logger *l.Logger) error {
	logger.PushF("proc.createFileAndStartWorker")
	defer logger.PopF()

	u, err := url.Parse(instr.FinalFileUrl)
	if err != nil {
		return fmt.Errorf("cannot parse file uri %s: %s", instr.FinalFileUrl, err.Error())
	}

	var f *os.File
	if u.Scheme == xfer.UriSchemeSftp {
		f, err = os.CreateTemp("", "capi")
		if err != nil {
			return fmt.Errorf("cannot create temp file for %s: %s", instr.FinalFileUrl, err.Error())
		}
		instr.TempFilePath = f.Name()
	} else {
		f, err = os.Create(instr.FinalFileUrl)
		if err != nil {
			return err
		}
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
		f.Close()
		return fmt.Errorf("cannot write file [%s] header line: [%s]", instr.FinalFileUrl, err.Error())
	}

	f.Close()

	newLogger, err := l.NewLoggerFromLogger(logger)
	if err != nil {
		return err
	}
	go instr.fileInserterWorker(newLogger)

	return nil
}

func (instr *FileInserter) checkWorkerOutputForErrors() error {
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

func (instr *FileInserter) waitForWorker(logger *l.Logger, pCtx *ctx.MessageProcessingContext) error {
	logger.PushF("proc.waitForWorkers/FieInserter")
	defer logger.PopF()

	// waitForWorker may be used for writing leftovers, handle them
	if instr.CurrentBatch != nil && instr.CurrentBatch.RowCount > 0 {
		instr.BatchesIn <- instr.CurrentBatch
		instr.BatchesSent++
		instr.CurrentBatch = nil
	}

	if instr.BatchesInOpen {
		logger.DebugCtx(pCtx, "closing BatchesIn")
		close(instr.BatchesIn)
		logger.DebugCtx(pCtx, "closed BatchesIn")
		instr.BatchesInOpen = false
	}

	logger.DebugCtx(pCtx, "started reading from BatchesSent")
	errors := make([]string, 0)
	// It's crucial that the number of errors to receive eventually should match instr.BatchesSent
	for i := 0; i < instr.BatchesSent; i++ {
		err := <-instr.ErrorsOut
		if err != nil {
			errors = append(errors, err.Error())
		}
	}
	logger.DebugCtx(pCtx, "done reading from BatchesSent")

	// Reset for the next cycle, if it ever happens
	instr.BatchesSent = 0

	if len(errors) > 0 {
		return fmt.Errorf(strings.Join(errors, "; "))
	} else {
		return nil
	}
}

func (instr *FileInserter) waitForWorkerAndClose(logger *l.Logger, pCtx *ctx.MessageProcessingContext) {
	logger.PushF("proc.waitForWorkersAndClose/FileInserter")
	defer logger.PopF()

	instr.waitForWorker(logger, pCtx)
	logger.DebugCtx(pCtx, "closing ErrorsOut")
	close(instr.ErrorsOut)
	logger.DebugCtx(pCtx, "closed ErrorsOut")
}

func (instr *FileInserter) add(row []interface{}) {
	if instr.CurrentBatch == nil {
		instr.CurrentBatch = newWriteFileBatch(instr.BatchCapacity)
	}
	instr.CurrentBatch.Rows[instr.CurrentBatch.RowCount] = row
	instr.CurrentBatch.RowCount++

	if instr.CurrentBatch.RowCount == instr.BatchCapacity {
		instr.BatchesIn <- instr.CurrentBatch
		instr.BatchesSent++
		instr.CurrentBatch = nil
	}
}

func (instr *FileInserter) fileInserterWorker(logger *l.Logger) {
	logger.PushF("proc.fileInserterWorker")
	defer logger.PopF()

	var localFilePath string
	if instr.TempFilePath != "" {
		localFilePath = instr.TempFilePath
	} else {
		localFilePath = instr.FinalFileUrl
	}

	f, err := os.OpenFile(localFilePath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		instr.ErrorsOut <- fmt.Errorf("cannot open %s(temp %s) for appending: [%s]", instr.FinalFileUrl, instr.TempFilePath, err.Error())
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
					b.WriteString(instr.FileCreator.Separator)
				}
			}
		}

		f.Sync()
		if _, err := f.WriteString(b.String()); err != nil {
			instr.ErrorsOut <- fmt.Errorf("cannot write string to %s(temp %s): [%s]", instr.FinalFileUrl, instr.TempFilePath, err.Error())
		} else {
			instr.ErrorsOut <- nil
		}
		dur := time.Since(batchStartTime)
		logger.InfoCtx(instr.PCtx, "%d items in %.3fs (%.0f items/s)", batch.RowCount, dur.Seconds(), float64(batch.RowCount)/dur.Seconds())
	}
}

func (instr *FileInserter) sendFileToFinal(logger *l.Logger, pCtx *ctx.MessageProcessingContext, privateKeys map[string]string) error {
	logger.PushF("proc.sendFileToFinal")
	defer logger.PopF()

	if instr.TempFilePath == "" {
		// Nothing to do, the file is already at its destination
		return nil
	}
	defer os.Remove(instr.TempFilePath)

	logger.InfoCtx(pCtx, "uploading %s to %s...", instr.TempFilePath, instr.FinalFileUrl)

	return xfer.UploadSftpFile(instr.TempFilePath, instr.FinalFileUrl, privateKeys)
}
