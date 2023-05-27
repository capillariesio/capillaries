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
)

type WriteCsvColumnSettings struct {
	Format string `json:"format"`
	Header string `json:"header"`
}

type WriteFileColumnDef struct {
	RawExpression    string                 `json:"expression"`
	Name             string                 `json:"name"` // To be used in Having
	Type             TableFieldType         `json:"type"` // To be checked when checking expressions and to be used in Having
	Csv              WriteCsvColumnSettings `json:"csv,omitempty"`
	ParsedExpression ast.Expr
	UsedFields       FieldRefs
}

type TopDef struct {
	Limit       int    `json:"limit"`
	RawOrder    string `json:"order"`
	OrderIdxDef IdxDef // Not an index really, we just re-use IdxDef infrastructure
}

type CsvCreatorSettings struct {
	Separator string `json:"separator"`
}

type FileCreatorDef struct {
	RawHaving                     string `json:"having"`
	Having                        ast.Expr
	UsedInHavingFields            FieldRefs
	UsedInTargetExpressionsFields FieldRefs
	Columns                       []WriteFileColumnDef `json:"columns"`
	UrlTemplate                   string               `json:"url_template"`
	Top                           TopDef               `json:"top"`
	Csv                           CsvCreatorSettings   `json:"csv,omitempty"`
	CreatorFileType               int
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

	// TODO: add more file types here
	if len(creatorDef.Csv.Separator) > 0 {
		creatorDef.CreatorFileType = CreatorFileTypeCsv
	} else {
		// By default it's a CSV writer
		creatorDef.CreatorFileType = CreatorFileTypeCsv
	}

	if creatorDef.CreatorFileType == CreatorFileTypeCsv {
		// Default CSV field Separator
		if len(creatorDef.Csv.Separator) == 0 {
			creatorDef.Csv.Separator = ","
		}
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
		idxDefMap.parseRawIndexDefMap(rawIndexes, creatorDef.getFieldRefs())
		creatorDef.Top.OrderIdxDef = *idxDefMap["top"]
	}

	creatorDef.UsedInTargetExpressionsFields = creatorDef.GetFieldRefsUsedInAllTargetFileExpressions()
	return nil
}

func (creatorDef *FileCreatorDef) CalculateFileRecordFromSrcVars(srcVars eval.VarValuesMap) ([]interface{}, error) {
	errors := make([]string, 0, 2)

	fileRecord := make([]interface{}, len(creatorDef.Columns))

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
	} else {
		return fileRecord, nil
	}
}

func (creatorDef *FileCreatorDef) CheckFileRecordHavingCondition(fileRecord []interface{}) (bool, error) {
	if creatorDef.Having == nil {
		return true, nil
	}
	vars := eval.VarValuesMap{}
	vars[CreatorAlias] = map[string]interface{}{}
	if len(fileRecord) != len(creatorDef.Columns) {
		return false, fmt.Errorf("file record length %d does not match file creator column list length %d", len(fileRecord), len(creatorDef.Columns))
	}
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
