package proc

import (
	"fmt"
	"os"
	"strings"

	"github.com/capillariesio/capillaries/pkg/ctx"
	"github.com/capillariesio/capillaries/pkg/l"
	"github.com/capillariesio/capillaries/pkg/sc"
	"github.com/capillariesio/capillaries/pkg/xfer"
)

type FileInserter struct {
	PCtx          *ctx.MessageProcessingContext
	FileCreator   *sc.FileCreatorDef
	CurrentBatch  *WriteFileBatch
	BatchCapacity int
	BatchesIn     chan *WriteFileBatch
	ErrorsOut     chan error
	BatchesSent   int
	FinalFileUrl  string
	TempFilePath  string
}

const DefaultFileInserterBatchCapacity int = 1000

type WriteFileBatch struct {
	Rows     [][]any
	RowCount int
}

func newWriteFileBatch(batchCapacity int) *WriteFileBatch {
	return &WriteFileBatch{
		Rows:     make([][]any, batchCapacity),
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
		FinalFileUrl:  strings.ReplaceAll(strings.ReplaceAll(fileCreator.UrlTemplate, sc.ReservedParamRunId, fmt.Sprintf("%05d", runId)), sc.ReservedParamBatchIdx, fmt.Sprintf("%05d", batchIdx)),
	}

	return &instr
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

func (instr *FileInserter) waitForWorker(logger *l.CapiLogger, pCtx *ctx.MessageProcessingContext) error {
	logger.PushF("proc.waitForWorkers/FieInserter")
	defer logger.PopF()

	// waitForWorker may be used for writing leftovers, handle them
	if instr.CurrentBatch != nil && instr.CurrentBatch.RowCount > 0 {
		instr.BatchesIn <- instr.CurrentBatch
		instr.BatchesSent++
		instr.CurrentBatch = nil
	}

	logger.DebugCtx(pCtx, "started reading BatchesSent=%d from instr.ErrorsOut", instr.BatchesSent)
	errors := make([]string, 0)
	// It's crucial that the number of errors to receive eventually should match instr.BatchesSent
	errCount := 0
	for i := 0; i < instr.BatchesSent; i++ {
		err := <-instr.ErrorsOut
		if err != nil {
			errors = append(errors, err.Error())
			errCount++
		}
		logger.DebugCtx(pCtx, "got result for sent record %d out of %d from instr.ErrorsOut, %d errors so far", i, instr.BatchesSent, errCount)
	}
	logger.DebugCtx(pCtx, "done reading BatchesSent=%d from instr.ErrorsOut, %d errors", instr.BatchesSent, errCount)

	// Reset for the next cycle, if it ever happens
	instr.BatchesSent = 0

	// Now it's safe to close
	logger.DebugCtx(pCtx, "closing BatchesIn")
	close(instr.BatchesIn)
	logger.DebugCtx(pCtx, "closed BatchesIn")

	if len(errors) > 0 {
		return fmt.Errorf(strings.Join(errors, "; "))
	} else {
		return nil
	}
}

func (instr *FileInserter) waitForWorkerAndCloseErrorsOut(logger *l.CapiLogger, pCtx *ctx.MessageProcessingContext) error {
	logger.PushF("proc.waitForWorkersAndClose/FileInserter")
	defer logger.PopF()

	err := instr.waitForWorker(logger, pCtx)
	logger.DebugCtx(pCtx, "closing ErrorsOut")
	close(instr.ErrorsOut)
	logger.DebugCtx(pCtx, "closed ErrorsOut")
	return err
}

func (instr *FileInserter) add(row []any) {
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

func (instr *FileInserter) sendFileToFinal(logger *l.CapiLogger, pCtx *ctx.MessageProcessingContext, privateKeys map[string]string) error {
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
