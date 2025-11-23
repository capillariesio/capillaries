package proc

import (
	"fmt"
	"net/url"
	"os"
	"time"

	"github.com/capillariesio/capillaries/pkg/l"
	"github.com/capillariesio/capillaries/pkg/sc"
	"github.com/capillariesio/capillaries/pkg/storage"
	"github.com/capillariesio/capillaries/pkg/xfer"
	"github.com/shopspring/decimal"
)

func (instr *FileInserter) createParquetFileAndStartWorker(logger *l.CapiLogger, codec sc.ParquetCodecType, u *url.URL) error {
	logger.PushF("proc.createParquetFileAndStartWorker")
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

	f.Close()

	newLogger, err := l.NewLoggerFromLogger(logger)
	if err != nil {
		return err
	}
	go instr.parquetFileInserterWorker(newLogger, codec)

	return nil
}

func (instr *FileInserter) generateMapToAdd(batch *WriteFileBatch, rowIdx int) (map[string]any, error) {
	d := map[string]any{}
	for i := 0; i < len(instr.FileCreator.Columns); i++ {
		switch instr.FileCreator.Columns[i].Type {
		case sc.FieldTypeString:
			typedValue, ok := batch.Rows[rowIdx][i].(string)
			if !ok {
				return nil, fmt.Errorf("cannot convert column %s value [%v] to Parquet string", instr.FileCreator.Columns[i].Parquet.ColumnName, batch.Rows[rowIdx][i])
			}
			d[instr.FileCreator.Columns[i].Parquet.ColumnName] = typedValue
		case sc.FieldTypeInt:
			typedValue, ok := batch.Rows[rowIdx][i].(int64)
			if !ok {
				return nil, fmt.Errorf("cannot convert column %s value [%v] to Parquet int64", instr.FileCreator.Columns[i].Parquet.ColumnName, batch.Rows[rowIdx][i])
			}
			d[instr.FileCreator.Columns[i].Parquet.ColumnName] = typedValue
		case sc.FieldTypeFloat:
			typedValue, ok := batch.Rows[rowIdx][i].(float64)
			if !ok {
				return nil, fmt.Errorf("cannot convert column %s value [%v] to Parquet float64", instr.FileCreator.Columns[i].Parquet.ColumnName, batch.Rows[rowIdx][i])
			}
			d[instr.FileCreator.Columns[i].Parquet.ColumnName] = typedValue
		case sc.FieldTypeBool:
			typedValue, ok := batch.Rows[rowIdx][i].(bool)
			if !ok {
				return nil, fmt.Errorf("cannot convert column %s value [%v] to Parquet bool", instr.FileCreator.Columns[i].Parquet.ColumnName, batch.Rows[rowIdx][i])
			}
			d[instr.FileCreator.Columns[i].Parquet.ColumnName] = typedValue
		case sc.FieldTypeDecimal2:
			typedValue, ok := batch.Rows[rowIdx][i].(decimal.Decimal)
			if !ok {
				return nil, fmt.Errorf("cannot convert column %s value [%v] to Parquet decimal", instr.FileCreator.Columns[i].Parquet.ColumnName, batch.Rows[rowIdx][i])
			}
			d[instr.FileCreator.Columns[i].Parquet.ColumnName] = storage.ParquetWriterDecimal2(typedValue)
		case sc.FieldTypeDateTime:
			typedValue, ok := batch.Rows[rowIdx][i].(time.Time)
			if !ok {
				return nil, fmt.Errorf("cannot convert column %s value [%v] to Parquet datetime", instr.FileCreator.Columns[i].Parquet.ColumnName, batch.Rows[rowIdx][i])
			}
			d[instr.FileCreator.Columns[i].Parquet.ColumnName] = storage.ParquetWriterMilliTs(typedValue)
		default:
			return nil, fmt.Errorf("cannot convert column %s value [%v] to Parquet: unsupported type", instr.FileCreator.Columns[i].Parquet.ColumnName, batch.Rows[rowIdx][i])
		}
	}

	return d, nil
}

func (instr *FileInserter) parquetFileInserterWorker(logger *l.CapiLogger, codec sc.ParquetCodecType) {
	logger.PushF("proc.parquetFileInserterWorker")
	defer logger.Close()

	var localFilePath string
	if instr.TempFilePath != "" {
		localFilePath = instr.TempFilePath
	} else {
		localFilePath = instr.FinalFileUrl
	}

	var errOpen error
	var w *storage.ParquetWriter
	f, err := os.OpenFile(localFilePath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		errOpen = fmt.Errorf("cannot open parquet %s(temp %s) for appending: [%s]", instr.FinalFileUrl, instr.TempFilePath, err.Error())
	} else {
		if f == nil {
			errOpen = fmt.Errorf("cannot open parquet %s(temp %s) for appending: unknown error", instr.FinalFileUrl, instr.TempFilePath)
		} else {
			defer f.Close()
		}

		w, err = storage.NewParquetWriter(f, codec)
		if err != nil {
			errOpen = err
		} else {
			for i := 0; i < len(instr.FileCreator.Columns); i++ {
				if err := w.AddColumn(instr.FileCreator.Columns[i].Parquet.ColumnName, instr.FileCreator.Columns[i].Type); err != nil {
					errOpen = err
					break
				}
			}
		}
	}

	for batch := range instr.BatchesIn {
		if errOpen != nil {
			instr.RecordWrittenStatuses <- errOpen
			continue
		}
		batchStartTime := time.Now()
		var errAddData error
		for rowIdx := 0; rowIdx < batch.RowCount; rowIdx++ {
			var d map[string]any
			d, errAddData = instr.generateMapToAdd(batch, rowIdx)
			if errAddData != nil {
				break
			}
			if err := w.FileWriter.AddData(d); err != nil {
				errAddData = err
				break
			}
		}

		if errAddData != nil {
			instr.RecordWrittenStatuses <- errAddData
		} else {
			dur := time.Since(batchStartTime)
			logger.InfoCtx(instr.PCtx, "%d items in %.3fs (%.0f items/s)", batch.RowCount, dur.Seconds(), float64(batch.RowCount)/dur.Seconds())
			instr.RecordWrittenStatuses <- nil
		}
		instr.PCtx.SendHeartbeat()
	} // next batch

	if w != nil {
		if err := w.Close(); err != nil {
			logger.ErrorCtx(instr.PCtx, "cannot close parquet writer %s(temp %s): [%s]", instr.FinalFileUrl, instr.TempFilePath, err.Error())
		}
	}
}
