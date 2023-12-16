package sc

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/capillariesio/capillaries/pkg/eval"
	"github.com/shopspring/decimal"
)

type CsvReaderColumnSettings struct {
	SrcColIdx    int    `json:"col_idx,omitempty"`
	SrcColHeader string `json:"col_hdr,omitempty"`
	SrcColFormat string `json:"col_format,omitempty"` // Optional for all except datetime
}

type ParquetReaderColumnSettings struct {
	SrcColName string `json:"col_name"`
}

type FileReaderColumnDef struct {
	DefaultValue string                      `json:"col_default_value,omitempty"` // Optional. If omitted, zero value is used
	Type         TableFieldType              `json:"col_type"`
	Csv          CsvReaderColumnSettings     `json:"csv,omitempty"`
	Parquet      ParquetReaderColumnSettings `json:"parquet,omitempty"`
}

type CsvReaderSettings struct {
	SrcFileHdrLineIdx       int                    `json:"hdr_line_idx"`
	SrcFileFirstDataLineIdx int                    `json:"first_data_line_idx,omitempty"`
	Separator               string                 `json:"separator,omitempty"`
	ColumnIndexingMode      FileColumnIndexingMode `json:"-"`
}

const (
	ReaderFileTypeUnknown int = 0
	ReaderFileTypeCsv     int = 1
	ReaderFileTypeParquet int = 2
)

type FileReaderDef struct {
	SrcFileUrls    []string                        `json:"urls"`
	Columns        map[string]*FileReaderColumnDef `json:"columns"` // Keys are names used in table writer
	Csv            CsvReaderSettings               `json:"csv,omitempty"`
	ReaderFileType int                             `json:"-"`
}

func (frDef *FileReaderDef) getFieldRefs() *FieldRefs {
	fieldRefs := make(FieldRefs, len(frDef.Columns))
	i := 0
	for fieldName, colDef := range frDef.Columns {
		fieldRefs[i] = FieldRef{
			TableName: ReaderAlias,
			FieldName: fieldName,
			FieldType: colDef.Type}
		i++
	}
	return &fieldRefs
}

type FileColumnIndexingMode string

const (
	FileColumnIndexingName    FileColumnIndexingMode = "name"
	FileColumnIndexingIdx     FileColumnIndexingMode = "idx"
	FileColumnIndexingUnknown FileColumnIndexingMode = "unknown"
)

func (frDef *FileReaderDef) getCsvColumnIndexingMode() (FileColumnIndexingMode, error) {
	usesIdxCount := 0
	usesHdrNameCount := 0
	for _, colDef := range frDef.Columns {
		if len(colDef.Csv.SrcColHeader) > 0 {
			usesHdrNameCount++ // We have a name, ignore col idx, it's probably zero (default)
		} else if colDef.Csv.SrcColIdx >= 0 {
			usesIdxCount++
		} else {
			if colDef.Csv.SrcColIdx < 0 {
				return "", fmt.Errorf("file reader column definition cannot use negative column index: %d", colDef.Csv.SrcColIdx)
			}
		}
	}

	if usesIdxCount > 0 && usesHdrNameCount > 0 {
		return "", fmt.Errorf("file reader column definitions cannot use both indexes and names, pick one method: col_hdr or col_idx")
	}

	if usesIdxCount > 0 {
		return FileColumnIndexingIdx, nil
	} else if usesHdrNameCount > 0 {
		return FileColumnIndexingName, nil
	}

	// Never land here
	return "", fmt.Errorf("file reader column indexing mode dev error")

}

func (frDef *FileReaderDef) Deserialize(rawReader json.RawMessage) error {
	errors := make([]string, 0, 2)

	// Unmarshal

	if err := json.Unmarshal(rawReader, frDef); err != nil {
		errors = append(errors, err.Error())
	}

	if len(frDef.SrcFileUrls) == 0 {
		errors = append(errors, "no source file urls specified, need at least one")
	}

	frDef.ReaderFileType = ReaderFileTypeUnknown
	for _, colDef := range frDef.Columns {
		if colDef.Parquet.SrcColName != "" {
			frDef.ReaderFileType = ReaderFileTypeParquet
			break
		} else if (colDef.Csv.SrcColHeader != "" || colDef.Csv.SrcColIdx > 0) ||
			len(frDef.Columns) == 1 { // Special CSV case: no headers, only one column
			frDef.ReaderFileType = ReaderFileTypeCsv

			// Detect column indexing mode: by idx or by name
			var err error
			frDef.Csv.ColumnIndexingMode, err = frDef.getCsvColumnIndexingMode()
			if err != nil {
				errors = append(errors, fmt.Sprintf("cannot detect csv column indexing mode: [%s]", err.Error()))
			}

			// Default CSV field Separator
			if len(frDef.Csv.Separator) == 0 {
				frDef.Csv.Separator = ","
			}
			break
		}
	}

	if frDef.ReaderFileType == ReaderFileTypeUnknown {
		errors = append(errors, "cannot detect file reader type: parquet should have col_name, csv should have col_hdr or col_idx etc")
	}

	if len(errors) > 0 {
		return fmt.Errorf(strings.Join(errors, "; "))
	}
	return nil
}

func (frDef *FileReaderDef) ResolveCsvColumnIndexesFromNames(srcHdrLine []string) error {
	columnsResolved := 0
	for _, colDef := range frDef.Columns {
		for i := 0; i < len(srcHdrLine); i++ {
			if len(colDef.Csv.SrcColHeader) > 0 && srcHdrLine[i] == colDef.Csv.SrcColHeader {
				colDef.Csv.SrcColIdx = i
				columnsResolved++
			}
		}
	}
	if columnsResolved < len(frDef.Columns) {
		return fmt.Errorf("cannot resove all %d source file column indexes, resolved only %d", len(frDef.Columns), columnsResolved)
	}
	return nil
}

func toString(colName string, colData string, colDef *FileReaderColumnDef, colVars eval.VarValuesMap) error {
	if len(colDef.Csv.SrcColFormat) > 0 {
		return fmt.Errorf("cannot read string column %s, data '%s': format '%s' was specified, but string fields do not accept format specifier, remove this setting", colName, colData, colDef.Csv.SrcColFormat)
	}
	if len(colData) == 0 {
		if len(colDef.DefaultValue) > 0 {
			colVars[ReaderAlias][colName] = colDef.DefaultValue
		} else {
			colVars[ReaderAlias][colName] = GetDefaultFieldTypeValue(FieldTypeString)
		}
	} else {
		colVars[ReaderAlias][colName] = colData
	}
	return nil
}

func toBool(colName string, colData string, colDef *FileReaderColumnDef, colVars eval.VarValuesMap) error {
	if len(colDef.Csv.SrcColFormat) > 0 {
		return fmt.Errorf("cannot read bool column %s, data '%s': format '%s' was specified, but bool fields do not accept format specifier, remove this setting", colName, colData, colDef.Csv.SrcColFormat)
	}

	var err error
	if len(strings.TrimSpace(colData)) == 0 {
		if len(strings.TrimSpace(colDef.DefaultValue)) > 0 {
			colVars[ReaderAlias][colName], err = strconv.ParseBool(colDef.DefaultValue)
			if err != nil {
				return fmt.Errorf("cannot read bool column %s, from default value string '%s', allowed values are true,false,T,F,0,1: %s", colName, colDef.DefaultValue, err.Error())
			}
		} else {
			colVars[ReaderAlias][colName] = GetDefaultFieldTypeValue(FieldTypeBool)
		}
	} else {
		colVars[ReaderAlias][colName], err = strconv.ParseBool(colData)
		if err != nil {
			return fmt.Errorf("cannot read bool column %s, data '%s', allowed values are true,false,T,F,0,1: %s", colName, colData, err.Error())
		}
	}
	return nil
}

func toInt(colName string, colData string, colDef *FileReaderColumnDef, colVars eval.VarValuesMap) error {
	if len(strings.TrimSpace(colData)) == 0 {
		if len(strings.TrimSpace(colDef.DefaultValue)) > 0 {
			valInt, err := strconv.ParseInt(colDef.DefaultValue, 10, 64)
			if err != nil {
				return fmt.Errorf("cannot read int64 column %s from default value string '%s': %s", colName, colDef.DefaultValue, err.Error())
			}
			colVars[ReaderAlias][colName] = valInt
		} else {
			colVars[ReaderAlias][colName] = GetDefaultFieldTypeValue(FieldTypeInt)
		}
	} else {
		if len(colDef.Csv.SrcColFormat) > 0 {
			var valInt int64
			_, err := fmt.Sscanf(colData, colDef.Csv.SrcColFormat, &valInt)
			if err != nil {
				return fmt.Errorf("cannot read int64 column %s, data '%s', format '%s': %s", colName, colData, colDef.Csv.SrcColFormat, err.Error())
			}
			colVars[ReaderAlias][colName] = valInt
		} else {
			valInt, err := strconv.ParseInt(colData, 10, 64)
			if err != nil {
				return fmt.Errorf("cannot read int64 column %s, data '%s', no format: %s", colName, colData, err.Error())
			}
			colVars[ReaderAlias][colName] = valInt
		}
	}
	return nil
}

func toDateTime(colName string, colData string, colDef *FileReaderColumnDef, colVars eval.VarValuesMap) error {
	if len(strings.TrimSpace(colData)) == 0 {
		if len(strings.TrimSpace(colDef.DefaultValue)) > 0 {
			valTime, err := time.Parse(colDef.Csv.SrcColFormat, colDef.DefaultValue)
			if err != nil {
				return fmt.Errorf("cannot read time column %s from default value string '%s': %s", colName, colDef.DefaultValue, err.Error())
			}
			colVars[ReaderAlias][colName] = valTime
		} else {
			colVars[ReaderAlias][colName] = GetDefaultFieldTypeValue(FieldTypeDateTime)
		}
	} else {
		if len(colDef.Csv.SrcColFormat) == 0 {
			return fmt.Errorf("cannot read datetime column %s, data '%s': column format is missing, consider specifying something like 2006-01-02T15:04:05.000-0700, see go datetime format documentation for details", colName, colData)
		}

		valTime, err := time.Parse(colDef.Csv.SrcColFormat, colData)
		if err != nil {
			return fmt.Errorf("cannot read datetime column %s, data '%s', format '%s': %s", colName, colData, colDef.Csv.SrcColFormat, err.Error())
		}
		colVars[ReaderAlias][colName] = valTime
	}
	return nil
}

func toFloat(colName string, colData string, colDef *FileReaderColumnDef, colVars eval.VarValuesMap) error {
	if len(strings.TrimSpace(colData)) == 0 {
		if len(strings.TrimSpace(colDef.DefaultValue)) > 0 {
			valFloat, err := strconv.ParseFloat(colDef.DefaultValue, 64)
			if err != nil {
				return fmt.Errorf("cannot read float64 column %s from default value string '%s': %s", colName, colDef.DefaultValue, err.Error())
			}
			colVars[ReaderAlias][colName] = valFloat
		} else {
			colVars[ReaderAlias][colName] = GetDefaultFieldTypeValue(FieldTypeFloat)
		}
	} else {
		if len(colDef.Csv.SrcColFormat) > 0 {
			var valFloat float64
			_, err := fmt.Sscanf(colData, colDef.Csv.SrcColFormat, &valFloat)
			if err != nil {
				return fmt.Errorf("cannot read float64 column %s, data '%s', format '%s': %s", colName, colData, colDef.Csv.SrcColFormat, err.Error())
			}
			colVars[ReaderAlias][colName] = valFloat
		} else {
			valFloat, err := strconv.ParseFloat(colData, 64)
			if err != nil {
				return fmt.Errorf("cannot read float64 column %s, data '%s', no format: %s", colName, colData, err.Error())
			}
			colVars[ReaderAlias][colName] = valFloat
		}
	}
	return nil
}

func toDecimal2(colName string, colData string, colDef *FileReaderColumnDef, colVars eval.VarValuesMap) error {
	// Round to 2 digits after decimal point right away
	if len(strings.TrimSpace(colData)) == 0 {
		if len(strings.TrimSpace(colDef.DefaultValue)) > 0 {
			valDec, err := decimal.NewFromString(colDef.DefaultValue)
			if err != nil {
				return fmt.Errorf("cannot read decimal2 column %s from default value string '%s': %s", colName, colDef.DefaultValue, err.Error())
			}
			colVars[ReaderAlias][colName] = valDec.Round(2)
		} else {
			colVars[ReaderAlias][colName] = GetDefaultFieldTypeValue(FieldTypeDecimal2)
		}
	} else {
		var valFloat float64
		if len(colDef.Csv.SrcColFormat) > 0 {
			// Decimal type does not support sscanf, so sscanf string first
			_, err := fmt.Sscanf(colData, colDef.Csv.SrcColFormat, &valFloat)
			if err != nil {
				return fmt.Errorf("cannot read decimal2 column %s, data '%s', format '%s': %s", colName, colData, colDef.Csv.SrcColFormat, err.Error())
			}
			colVars[ReaderAlias][colName] = decimal.NewFromFloat(valFloat).Round(2)
		} else {
			valDec, err := decimal.NewFromString(colData)
			if err != nil {
				return fmt.Errorf("cannot read decimal2 column %s, cannot parse data '%s': %s", colName, colData, err.Error())
			}
			colVars[ReaderAlias][colName] = valDec.Round(2)
		}
	}
	return nil
}

func (frDef *FileReaderDef) ReadCsvLineToValuesMap(line *[]string, colVars eval.VarValuesMap) error {
	colVars[ReaderAlias] = map[string]any{}
	for colName, colDef := range frDef.Columns {
		colData := (*line)[colDef.Csv.SrcColIdx]
		switch colDef.Type {
		case FieldTypeString:
			if err := toString(colName, colData, colDef, colVars); err != nil {
				return err
			}
		case FieldTypeBool:
			if err := toBool(colName, colData, colDef, colVars); err != nil {
				return err
			}
		case FieldTypeInt:
			if err := toInt(colName, colData, colDef, colVars); err != nil {
				return err
			}
		case FieldTypeDateTime:
			if err := toDateTime(colName, colData, colDef, colVars); err != nil {
				return err
			}
		case FieldTypeFloat:
			if err := toFloat(colName, colData, colDef, colVars); err != nil {
				return err
			}
		case FieldTypeDecimal2:
			if err := toDecimal2(colName, colData, colDef, colVars); err != nil {
				return err
			}
		default:
			return fmt.Errorf("cannot read column %s, data '%s': unsupported column type '%s'", colName, colData, colDef.Type)
		}
	}
	return nil
}
