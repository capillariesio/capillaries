package proc

import (
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/capillariesio/capillaries/pkg/l"
	"github.com/capillariesio/capillaries/pkg/xfer"
	"github.com/shopspring/decimal"
)

func (instr *FileInserter) createCsvFileAndStartWorker(logger *l.CapiLogger, u *url.URL) error {
	logger.PushF("proc.createCsvFileAndStartWorker")
	defer logger.PopF()

	var err error
	var f *os.File
	if u.Scheme == xfer.UrlSchemeSftp || u.Scheme == xfer.UrlSchemeS3 {
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
		if strings.Contains(instr.FileCreator.Columns[i].Csv.Header, ",") {
			b.WriteString("\"")
		}
		b.WriteString(instr.FileCreator.Columns[i].Csv.Header)
		if strings.Contains(instr.FileCreator.Columns[i].Csv.Header, ",") {
			b.WriteString("\"")
		}
		if i == len(instr.FileCreator.Columns)-1 {
			b.WriteString("\r\n")
		} else {
			b.WriteString(instr.FileCreator.Csv.Separator)
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
	go instr.csvFileInserterWorker(newLogger)

	return nil
}

func (instr *FileInserter) csvFileInserterWorker(logger *l.CapiLogger) {
	logger.PushF("proc.csvFileInserterWorker")
	defer logger.Close()

	var localFilePath string
	if instr.TempFilePath != "" {
		localFilePath = instr.TempFilePath
	} else {
		localFilePath = instr.FinalFileUrl
	}

	var errOpen error
	f, err := os.OpenFile(localFilePath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		errOpen = fmt.Errorf("cannot open csv %s(temp %s) for appending: [%s]", instr.FinalFileUrl, instr.TempFilePath, err.Error())
	}
	if f == nil {
		errOpen = fmt.Errorf("cannot open csv %s(temp %s) for appending: unknown error", instr.FinalFileUrl, instr.TempFilePath)
	} else {
		defer f.Close()
	}

	for batch := range instr.BatchesIn {
		if errOpen != nil {
			instr.RecordWrittenStatuses <- errOpen
			continue
		}
		batchStartTime := time.Now()
		b := strings.Builder{}
		for rowIdx := 0; rowIdx < batch.RowCount; rowIdx++ {
			for i := 0; i < len(instr.FileCreator.Columns); i++ {
				var stringVal string
				switch assertedVal := batch.Rows[rowIdx][i].(type) {
				case time.Time:
					stringVal = assertedVal.Format(instr.FileCreator.Columns[i].Csv.Format)
				case decimal.Decimal:
					stringVal = fmt.Sprintf(instr.FileCreator.Columns[i].Csv.Format, assertedVal.StringFixed(2))
				default:
					stringVal = fmt.Sprintf(instr.FileCreator.Columns[i].Csv.Format, batch.Rows[rowIdx][i])
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
					b.WriteString(instr.FileCreator.Csv.Separator)
				}
			}
		}

		if err = f.Sync(); err == nil {
			if _, err = f.WriteString(b.String()); err != nil {
				instr.RecordWrittenStatuses <- fmt.Errorf("cannot write string to %s(temp %s): [%s]", instr.FinalFileUrl, instr.TempFilePath, err.Error())
			} else {
				dur := time.Since(batchStartTime)
				logger.InfoCtx(instr.PCtx, "%d items in %.3fs (%.0f items/s)", batch.RowCount, dur.Seconds(), float64(batch.RowCount)/dur.Seconds())
				instr.RecordWrittenStatuses <- nil
			}
		} else {
			instr.RecordWrittenStatuses <- fmt.Errorf("cannot sync file %s(temp %s): [%s]", instr.FinalFileUrl, instr.TempFilePath, err.Error())
		}
		instr.PCtx.SendHeartbeat()
	} // next batch
}
