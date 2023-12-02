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

func (instr *FileInserter) createParquetFileAndStartWorker(logger *l.Logger, codec sc.ParquetCodecType) error {
	logger.PushF("proc.createParquetFileAndStartWorker")
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

	f.Close()

	newLogger, err := l.NewLoggerFromLogger(logger)
	if err != nil {
		return err
	}
	go instr.parquetFileInserterWorker(newLogger, codec)

	return nil
}

func (instr *FileInserter) parquetFileInserterWorker(logger *l.Logger, codec sc.ParquetCodecType) {
	logger.PushF("proc.parquetFileInserterWorker")
	defer logger.PopF()

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
		errOpen = fmt.Errorf("cannot open %s(temp %s) for appending: [%s]", instr.FinalFileUrl, instr.TempFilePath, err.Error())
	} else {
		defer f.Close()

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
			instr.ErrorsOut <- errOpen
			continue
		}
		batchStartTime := time.Now()
		var errAddData error
		for rowIdx := 0; rowIdx < batch.RowCount; rowIdx++ {
			d := map[string]interface{}{}
			for i := 0; i < len(instr.FileCreator.Columns); i++ {
				switch instr.FileCreator.Columns[i].Type {
				case sc.FieldTypeString:
					typedValue, ok := batch.Rows[rowIdx][i].(string)
					if !ok {
						errAddData = fmt.Errorf("cannot convert column %s value [%v] to Parquet string", instr.FileCreator.Columns[i].Parquet.ColumnName, batch.Rows[rowIdx][i])
						break
					}
					d[instr.FileCreator.Columns[i].Parquet.ColumnName] = typedValue
				case sc.FieldTypeInt:
					typedValue, ok := batch.Rows[rowIdx][i].(int64)
					if !ok {
						errAddData = fmt.Errorf("cannot convert column %s value [%v] to Parquet int64", instr.FileCreator.Columns[i].Parquet.ColumnName, batch.Rows[rowIdx][i])
						break
					}
					d[instr.FileCreator.Columns[i].Parquet.ColumnName] = typedValue
				case sc.FieldTypeFloat:
					typedValue, ok := batch.Rows[rowIdx][i].(float64)
					if !ok {
						errAddData = fmt.Errorf("cannot convert column %s value [%v] to Parquet float64", instr.FileCreator.Columns[i].Parquet.ColumnName, batch.Rows[rowIdx][i])
						break
					}
					d[instr.FileCreator.Columns[i].Parquet.ColumnName] = typedValue
				case sc.FieldTypeBool:
					typedValue, ok := batch.Rows[rowIdx][i].(bool)
					if !ok {
						errAddData = fmt.Errorf("cannot convert column %s value [%v] to Parquet bool", instr.FileCreator.Columns[i].Parquet.ColumnName, batch.Rows[rowIdx][i])
						break
					}
					d[instr.FileCreator.Columns[i].Parquet.ColumnName] = typedValue
				case sc.FieldTypeDecimal2:
					typedValue, ok := batch.Rows[rowIdx][i].(decimal.Decimal)
					if !ok {
						errAddData = fmt.Errorf("cannot convert column %s value [%v] to Parquet decimal", instr.FileCreator.Columns[i].Parquet.ColumnName, batch.Rows[rowIdx][i])
						break
					}
					d[instr.FileCreator.Columns[i].Parquet.ColumnName] = storage.ParquetWriterDecimal2(typedValue)
				case sc.FieldTypeDateTime:
					typedValue, ok := batch.Rows[rowIdx][i].(time.Time)
					if !ok {
						errAddData = fmt.Errorf("cannot convert column %s value [%v] to Parquet datetime", instr.FileCreator.Columns[i].Parquet.ColumnName, batch.Rows[rowIdx][i])
						break
					}
					d[instr.FileCreator.Columns[i].Parquet.ColumnName] = storage.ParquetWriterMilliTs(typedValue)
				default:
					errAddData = fmt.Errorf("cannot convert column %s value [%v] to Parquet: unsupported type", instr.FileCreator.Columns[i].Parquet.ColumnName, batch.Rows[rowIdx][i])
					break //nolint:all , https://github.com/dominikh/go-tools/issues/59
				}
			}
			if err := w.FileWriter.AddData(d); err != nil {
				errAddData = err
				break
			}
		}

		if errAddData != nil {
			instr.ErrorsOut <- errAddData
		} else {
			dur := time.Since(batchStartTime)
			logger.InfoCtx(instr.PCtx, "%d items in %.3fs (%.0f items/s)", batch.RowCount, dur.Seconds(), float64(batch.RowCount)/dur.Seconds())
			instr.ErrorsOut <- nil
		}
	} // next batch

	if w != nil {
		if err := w.Close(); err != nil {
			logger.ErrorCtx(instr.PCtx, "cannot close parquet writer %s(temp %s): [%s]", instr.FinalFileUrl, instr.TempFilePath, err.Error())
		}
	}
}
