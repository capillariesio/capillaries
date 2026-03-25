package evalcapi

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
