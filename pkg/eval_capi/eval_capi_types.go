package eval_capi

type TableFieldType string

const (
	FieldTypeString   TableFieldType = "string"
	FieldTypeInt      TableFieldType = "int"      // sign+18digit string
	FieldTypeFloat    TableFieldType = "float"    // sign+64digit string, 32 digits after point
	FieldTypeBool     TableFieldType = "bool"     // F or T
	FieldTypeDecimal2 TableFieldType = "decimal2" // sign + 18digit+point+2
	FieldTypeDateTime TableFieldType = "datetime" // int unix epoch milliseconds
	FieldTypeUnknown  TableFieldType = "unknown"
)

func IsValidFieldType(fieldType TableFieldType) bool {
	return fieldType == FieldTypeString ||
		fieldType == FieldTypeInt ||
		fieldType == FieldTypeFloat ||
		fieldType == FieldTypeBool ||
		fieldType == FieldTypeDecimal2 ||
		fieldType == FieldTypeDateTime
}

// Etra massaging for capi-specific types
// func EvalCapiField(eCtx *eval.EvalCtx, exp ast.Expr, dataType TableFieldType) (any, error) {
// 	val, err := eCtx.Eval(exp)
// 	if err != nil {
// 		return val, err
// 	}

// 	if dataType == FieldTypeDecimal2 {
// 		valDecimal, ok := val.(decimal.Decimal)
// 		if !ok {
// 			return nil, fmt.Errorf("cannot accept non-decimal result of eval: (%T,%v)", val, val)
// 		}
// 		return valDecimal.Round(2), nil
// 	}
// 	return val, nil
// }
