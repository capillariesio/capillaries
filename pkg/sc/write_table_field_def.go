package sc

import (
	"fmt"
	"go/ast"
)

type WriteTableFieldDef struct {
	RawExpression    string         `json:"expression" yaml:"expression"`
	Type             TableFieldType `json:"type" yaml:"type"`
	DefaultValue     string         `json:"default_value,omitempty" yaml:"default_value,omitempty"` // Optional. If omitted, default zero value is used
	ParsedExpression ast.Expr       `json:"-"`
	UsedFields       FieldRefs      `json:"-"`
}

func GetFieldRefsUsedInAllTargetExpressions(fieldDefMap map[string]*WriteTableFieldDef) FieldRefs {
	fieldRefMap := map[string]FieldRef{}
	for _, targetFieldDef := range fieldDefMap {
		for i := 0; i < len(targetFieldDef.UsedFields); i++ {
			hash := fmt.Sprintf("%s.%s", targetFieldDef.UsedFields[i].TableName, targetFieldDef.UsedFields[i].FieldName)
			if _, ok := fieldRefMap[hash]; !ok {
				fieldRefMap[hash] = targetFieldDef.UsedFields[i]
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
