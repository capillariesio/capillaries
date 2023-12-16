package sc

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"strings"

	"github.com/capillariesio/capillaries/pkg/eval"
)

const (
	CreatorFileTypeUnknown int = 0
	CreatorFileTypeCsv     int = 1
	CreatorFileTypeParquet int = 2
)

type ParquetCodecType string

const (
	ParquetCodecGzip         ParquetCodecType = "gzip"
	ParquetCodecSnappy       ParquetCodecType = "snappy"
	ParquetCodecUncompressed ParquetCodecType = "uncompressed"
)

type WriteCsvColumnSettings struct {
	Format string `json:"format"`
	Header string `json:"header"`
}

type WriteParquetColumnSettings struct {
	ColumnName string `json:"column_name"`
}

type WriteFileColumnDef struct {
	RawExpression    string                     `json:"expression"`
	Name             string                     `json:"name"` // To be used in Having
	Type             TableFieldType             `json:"type"` // To be checked when checking expressions and to be used in Having
	Csv              WriteCsvColumnSettings     `json:"csv,omitempty"`
	Parquet          WriteParquetColumnSettings `json:"parquet,omitempty"`
	ParsedExpression ast.Expr                   `json:"-"`
	UsedFields       FieldRefs                  `json:"-"`
}

type TopDef struct {
	Limit       int    `json:"limit"`
	RawOrder    string `json:"order"`
	OrderIdxDef IdxDef // Not an index really, we just re-use IdxDef infrastructure
}

type CsvCreatorSettings struct {
	Separator string `json:"separator"`
}

type ParquetCreatorSettings struct {
	Codec ParquetCodecType `json:"codec"`
}

type FileCreatorDef struct {
	UrlTemplate                   string                 `json:"url_template"`
	RawHaving                     string                 `json:"having,omitempty"`
	Top                           TopDef                 `json:"top,omitempty"`
	Csv                           CsvCreatorSettings     `json:"csv,omitempty"`
	Parquet                       ParquetCreatorSettings `json:"parquet,omitempty"`
	Columns                       []WriteFileColumnDef   `json:"columns"`
	Having                        ast.Expr               `json:"-"`
	UsedInHavingFields            FieldRefs              `json:"-"`
	UsedInTargetExpressionsFields FieldRefs              `json:"-"`
	CreatorFileType               int                    `json:"-"`
}

const MaxFileCreatorTopLimit int = 500000

func (creatorDef *FileCreatorDef) getFieldRefs() *FieldRefs {
	fieldRefs := make(FieldRefs, len(creatorDef.Columns))
	for i := 0; i < len(creatorDef.Columns); i++ {
		fieldRefs[i] = FieldRef{
			TableName: CreatorAlias,
			FieldName: creatorDef.Columns[i].Name,
			FieldType: creatorDef.Columns[i].Type}
	}
	return &fieldRefs
}

func (creatorDef *FileCreatorDef) GetFieldRefsUsedInAllTargetFileExpressions() FieldRefs {
	fieldRefMap := map[string]FieldRef{}
	for colIdx := 0; colIdx < len(creatorDef.Columns); colIdx++ {
		targetColDef := &creatorDef.Columns[colIdx]
		for i := 0; i < len((*targetColDef).UsedFields); i++ {
			hash := fmt.Sprintf("%s.%s", (*targetColDef).UsedFields[i].TableName, (*targetColDef).UsedFields[i].FieldName)
			if _, ok := fieldRefMap[hash]; !ok {
				fieldRefMap[hash] = (*targetColDef).UsedFields[i]
			}
		}
	}

	// Map to FieldRefs
	fieldRefs := make([]FieldRef, len(fieldRefMap))
	i := 0
	for _, fieldRef := range fieldRefMap {
		fieldRefs[i] = fieldRef
		i++
	}

	return fieldRefs
}

func (creatorDef *FileCreatorDef) HasTop() bool {
	return len(strings.TrimSpace(creatorDef.Top.RawOrder)) > 0
}

func (creatorDef *FileCreatorDef) Deserialize(rawWriter json.RawMessage) error {
	if err := json.Unmarshal(rawWriter, creatorDef); err != nil {
		return fmt.Errorf("cannot unmarshal file creator: [%s]", err.Error())
	}

	if len(creatorDef.Columns) > 0 && creatorDef.Columns[0].Parquet.ColumnName != "" {
		creatorDef.CreatorFileType = CreatorFileTypeParquet
		if creatorDef.Parquet.Codec == "" {
			creatorDef.Parquet.Codec = ParquetCodecGzip
		}
	} else if len(creatorDef.Columns) > 0 && creatorDef.Columns[0].Csv.Header != "" {
		creatorDef.CreatorFileType = CreatorFileTypeCsv
		if len(creatorDef.Csv.Separator) == 0 {
			creatorDef.Csv.Separator = ","
		}
	} else {
		return fmt.Errorf("cannot cannot detect file creator type: parquet should have column_name, csv should have header etc")
	}

	// Having
	var err error
	creatorDef.Having, err = ParseRawGolangExpressionStringAndHarvestFieldRefs(creatorDef.RawHaving, &creatorDef.UsedInHavingFields)
	if err != nil {
		return fmt.Errorf("cannot parse file creator 'having' condition [%s]: [%s]", creatorDef.RawHaving, err.Error())
	}

	// Columns
	for i := 0; i < len(creatorDef.Columns); i++ {
		colDef := &creatorDef.Columns[i]
		if (*colDef).ParsedExpression, err = ParseRawGolangExpressionStringAndHarvestFieldRefs((*colDef).RawExpression, &(*colDef).UsedFields); err != nil {
			return fmt.Errorf("cannot parse column expression [%s]: [%s]", (*colDef).RawExpression, err.Error())
		}
		if !IsValidFieldType(colDef.Type) {
			return fmt.Errorf("invalid column type [%s]", colDef.Type)
		}
	}

	// Top
	if creatorDef.HasTop() {
		if creatorDef.Top.Limit <= 0 {
			creatorDef.Top.Limit = MaxFileCreatorTopLimit
		} else if creatorDef.Top.Limit > MaxFileCreatorTopLimit {
			return fmt.Errorf("top.limit cannot exceed %d", MaxFileCreatorTopLimit)
		}
		idxDefMap := IdxDefMap{}
		rawIndexes := map[string]string{"top": fmt.Sprintf("non_unique(%s)", creatorDef.Top.RawOrder)}
		if err := idxDefMap.parseRawIndexDefMap(rawIndexes, creatorDef.getFieldRefs()); err != nil {
			return fmt.Errorf("cannot parse raw index definition(s) for top: %s", err.Error())
		}
		creatorDef.Top.OrderIdxDef = *idxDefMap["top"]
	}

	creatorDef.UsedInTargetExpressionsFields = creatorDef.GetFieldRefsUsedInAllTargetFileExpressions()
	return nil
}

func (creatorDef *FileCreatorDef) CalculateFileRecordFromSrcVars(srcVars eval.VarValuesMap) ([]any, error) {
	errors := make([]string, 0, 2)

	fileRecord := make([]any, len(creatorDef.Columns))

	for colIdx := 0; colIdx < len(creatorDef.Columns); colIdx++ {
		eCtx := eval.NewPlainEvalCtxWithVars(eval.AggFuncDisabled, &srcVars)
		valVolatile, err := eCtx.Eval(creatorDef.Columns[colIdx].ParsedExpression)
		if err != nil {
			errors = append(errors, fmt.Sprintf("cannot evaluate expression for column %s: [%s]", creatorDef.Columns[colIdx].Name, err.Error()))
		}
		if err := CheckValueType(valVolatile, creatorDef.Columns[colIdx].Type); err != nil {
			errors = append(errors, fmt.Sprintf("invalid field %s type: [%s]", creatorDef.Columns[colIdx].Name, err.Error()))
		}
		fileRecord[colIdx] = valVolatile
	}

	if len(errors) > 0 {
		return nil, fmt.Errorf(strings.Join(errors, "; "))
	}
	return fileRecord, nil
}

func (creatorDef *FileCreatorDef) CheckFileRecordHavingCondition(fileRecord []any) (bool, error) {
	if len(fileRecord) != len(creatorDef.Columns) {
		return false, fmt.Errorf("file record length %d does not match file creator column list length %d", len(fileRecord), len(creatorDef.Columns))
	}
	if creatorDef.Having == nil {
		return true, nil
	}
	vars := eval.VarValuesMap{}
	vars[CreatorAlias] = map[string]any{}
	for colIdx := 0; colIdx < len(creatorDef.Columns); colIdx++ {
		fieldName := creatorDef.Columns[colIdx].Name
		fieldValue := fileRecord[colIdx]
		vars[CreatorAlias][fieldName] = fieldValue
	}

	eCtx := eval.NewPlainEvalCtxWithVars(eval.AggFuncDisabled, &vars)
	valVolatile, err := eCtx.Eval(creatorDef.Having)
	if err != nil {
		return false, fmt.Errorf("cannot evaluate 'having' expression: [%s]", err.Error())
	}
	valBool, ok := valVolatile.(bool)
	if !ok {
		return false, fmt.Errorf("cannot get bool when evaluating having expression, got %v(%T) instead", valVolatile, valVolatile)
	}

	return valBool, nil
}
