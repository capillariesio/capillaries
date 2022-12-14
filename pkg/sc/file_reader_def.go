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

type FileReaderColumnDef struct {
	SrcColIdx    int            `json:"col_idx"`
	SrcColHeader string         `json:"col_hdr"`
	SrcColFormat string         `json:"col_format"`        // Optional for all except datetime
	DefaultValue string         `json:"col_default_value"` // Optional. If omitted, zero value is used
	Type         TableFieldType `json:"col_type"`
}

type FileReaderDef struct {
	SrcFileUrls             []string                        `json:"urls"`
	SrcFileHdrLineIdx       int                             `json:"hdr_line_idx"`
	SrcFileFirstDataLineIdx int                             `json:"first_data_line_idx"`
	Columns                 map[string]*FileReaderColumnDef `json:"columns"` // Keys are names used in table writer
	Separator               string                          `json:"separator"`
	ColumnIndexingMode      FileColumnIndexingMode
}

func (frDef *FileReaderDef) getFieldRefs() *FieldRefs {
	fieldRefs := make(FieldRefs, len(frDef.Columns))
	i := 0
	for fieldName, colDef := range frDef.Columns {
		fieldRefs[i] = FieldRef{
			TableName: ReaderAlias,
			FieldName: fieldName,
			FieldType: colDef.Type}
		i += 1
	}
	return &fieldRefs
}

type FileColumnIndexingMode string

const (
	FileColumnIndexingName    FileColumnIndexingMode = "name"
	FileColumnIndexingIdx     FileColumnIndexingMode = "idx"
	FileColumnIndexingUnknown FileColumnIndexingMode = "unknown"
)

func (frDef *FileReaderDef) getColumnIndexingMode() (FileColumnIndexingMode, error) {
	usesIdxCount := 0
	usesHdrNameCount := 0
	for _, colDef := range frDef.Columns {
		if len(colDef.SrcColHeader) > 0 {
			usesHdrNameCount++ // We have a name, ignore col idx, it's probably zero (default)
		} else if colDef.SrcColIdx >= 0 {
			usesIdxCount++
		} else {
			if colDef.SrcColIdx < 0 {
				return "", fmt.Errorf("file reader column definition cannot use negative column index: %d", colDef.SrcColIdx)
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

	// File urls - substitute template values

	if len(frDef.SrcFileUrls) == 0 {
		errors = append(errors, "no source file urls specified, need at least one")
	}

	// Detect column indexing mode: by idx or by name

	var err error
	frDef.ColumnIndexingMode, err = frDef.getColumnIndexingMode()
	if err != nil {
		errors = append(errors, fmt.Sprintf("cannot detect column indexing mode: [%s]", err.Error()))
	}

	// CSV field Separator
	if len(frDef.Separator) == 0 {
		frDef.Separator = ","
	}

	if len(errors) > 0 {
		return fmt.Errorf(strings.Join(errors, "; "))
	} else {
		return nil
	}
}

func (frDef *FileReaderDef) ResolveColumnIndexesFromNames(srcHdrLine []string) error {
	columnsResolved := 0
	for _, colDef := range frDef.Columns {
		for i := 0; i < len(srcHdrLine); i++ {
			if len(colDef.SrcColHeader) > 0 && srcHdrLine[i] == colDef.SrcColHeader {
				colDef.SrcColIdx = i
				columnsResolved++
			}
		}
	}
	if columnsResolved < len(frDef.Columns) {
		return fmt.Errorf("cannot resove all %d source file column indexes, resolved only %d", len(frDef.Columns), columnsResolved)
	}
	return nil
}

func (frDef *FileReaderDef) ReadLineToValuesMap(line *[]string, colVars eval.VarValuesMap) error {
	colVars[ReaderAlias] = map[string]interface{}{}
	for colName, colDef := range frDef.Columns {
		colData := (*line)[colDef.SrcColIdx]
		switch colDef.Type {
		case FieldTypeString:
			if len(colDef.SrcColFormat) > 0 {
				return fmt.Errorf("cannot read string column %s, data '%s': format '%s' was specified, but string fields do not accept format specifier, remove this setting", colName, colData, colDef.SrcColFormat)
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

		case FieldTypeBool:
			if len(colDef.SrcColFormat) > 0 {
				return fmt.Errorf("cannot read bool column %s, data '%s': format '%s' was specified, but bool fields do not accept format specifier, remove this setting", colName, colData, colDef.SrcColFormat)
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

		case FieldTypeInt:
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
				if len(colDef.SrcColFormat) > 0 {
					var valInt int64
					_, err := fmt.Sscanf(colData, colDef.SrcColFormat, &valInt)
					if err != nil {
						return fmt.Errorf("cannot read int64 column %s, data '%s', format '%s': %s", colName, colData, colDef.SrcColFormat, err.Error())
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

		case FieldTypeDateTime:
			if len(strings.TrimSpace(colData)) == 0 {
				if len(strings.TrimSpace(colDef.DefaultValue)) > 0 {
					valTime, err := time.Parse(colDef.SrcColFormat, colDef.DefaultValue)
					if err != nil {
						return fmt.Errorf("cannot read time column %s from default value string '%s': %s", colName, colDef.DefaultValue, err.Error())
					}
					colVars[ReaderAlias][colName] = valTime
				} else {
					colVars[ReaderAlias][colName] = GetDefaultFieldTypeValue(FieldTypeDateTime)
				}
			} else {
				if len(colDef.SrcColFormat) == 0 {
					return fmt.Errorf("cannot read datetime column %s, data '%s': column format is missing, consider specifying something like 2006-01-02T15:04:05.000-0700, see go datetime format documentation for details", colName, colData)
				}

				valTime, err := time.Parse(colDef.SrcColFormat, colData)
				if err != nil {
					return fmt.Errorf("cannot read datetime column %s, data '%s', format '%s': %s", colName, colData, colDef.SrcColFormat, err.Error())
				}
				colVars[ReaderAlias][colName] = valTime
			}

		case FieldTypeFloat:
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
				if len(colDef.SrcColFormat) > 0 {
					var valFloat float64
					_, err := fmt.Sscanf(colData, colDef.SrcColFormat, &valFloat)
					if err != nil {
						return fmt.Errorf("cannot read float64 column %s, data '%s', format '%s': %s", colName, colData, colDef.SrcColFormat, err.Error())
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

		case FieldTypeDecimal2:
			if len(strings.TrimSpace(colData)) == 0 {
				if len(strings.TrimSpace(colDef.DefaultValue)) > 0 {
					valDec, err := decimal.NewFromString(colDef.DefaultValue)
					if err != nil {
						return fmt.Errorf("cannot read decimal2 column %s from default value string '%s': %s", colName, colDef.DefaultValue, err.Error())
					}
					// TODO: round to 2 digits after decimal point
					colVars[ReaderAlias][colName] = valDec
				} else {
					colVars[ReaderAlias][colName] = GetDefaultFieldTypeValue(FieldTypeDecimal2)
				}
			} else {
				var valFloat float64
				if len(colDef.SrcColFormat) > 0 {
					// Decimal type does not support sscanf, so sscanf string first
					_, err := fmt.Sscanf(colData, colDef.SrcColFormat, &valFloat)
					if err != nil {
						return fmt.Errorf("cannot read decimal2 column %s, data '%s', format '%s': %s", colName, colData, colDef.SrcColFormat, err.Error())
					}
					colVars[ReaderAlias][colName] = decimal.NewFromFloat(valFloat)
				} else {
					var err error
					colVars[ReaderAlias][colName], err = decimal.NewFromString(colData)
					if err != nil {
						return fmt.Errorf("cannot read decimal2 column %s, cannot parse data '%s': %s", colName, colData, err.Error())
					}
				}
			}

		default:
			return fmt.Errorf("cannot read column %s, data '%s': unsupported column type '%s'", colName, colData, colDef.Type)
		}
	}
	return nil
}
