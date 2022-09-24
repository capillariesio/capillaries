package sc

import (
	"fmt"
	"time"

	"github.com/shopspring/decimal"
	"gopkg.in/inf.v0"
)

const (
	FieldNameUnknown = "unknown_field_name"
)

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

// Cassandra timestamps are milliseconds. No microsecond support.
// On writes:
// - allows (but not requires) ":" in the timezone
// - allows (but not requires) "T" in as date/time separator
const CassandraDatetimeFormat string = "2006-01-02T15:04:05.000-07:00"

const DefaultInt int64 = int64(0)
const DefaultFloat float64 = float64(0.0)
const DefaultString string = ""
const DefaultBool bool = false

func DefaultDecimal2() decimal.Decimal   { return decimal.NewFromFloat(0.0) }
func DefaultCassandraDecimal2() *inf.Dec { return inf.NewDec(0, 0) }
func DefaultDateTime() time.Time         { return time.Date(1901, 1, 1, 0, 0, 0, 0, time.UTC) }

func GetDefaultFieldTypeValue(fieldType TableFieldType) interface{} {
	switch fieldType {
	case FieldTypeInt:
		return DefaultInt
	case FieldTypeFloat:
		return DefaultFloat
	case FieldTypeString:
		return DefaultString
	case FieldTypeDecimal2:
		return DefaultDecimal2()
	case FieldTypeBool:
		return DefaultBool
	case FieldTypeDateTime:
		return DefaultDateTime()
	default:
		return nil
	}
}

func CheckValueType(val interface{}, fieldType TableFieldType) error {
	switch assertedValue := val.(type) {
	case int64:
		if fieldType != FieldTypeInt {
			return fmt.Errorf("expected type %s, but got int64 (%d)", fieldType, assertedValue)
		}
	case float64:
		if fieldType != FieldTypeFloat {
			return fmt.Errorf("expected type %s, but got float64 (%f)", fieldType, assertedValue)
		}
	case string:
		if fieldType != FieldTypeString {
			return fmt.Errorf("expected type %s, but got string (%s)", fieldType, assertedValue)
		}
	case bool:
		if fieldType != FieldTypeBool {
			return fmt.Errorf("expected type %s, but got bool (%v)", fieldType, assertedValue)
		}
	case time.Time:
		if fieldType != FieldTypeDateTime {
			return fmt.Errorf("expected type %s, but got datetime (%s)", fieldType, assertedValue.String())
		}
	case decimal.Decimal:
		if fieldType != FieldTypeDecimal2 {
			return fmt.Errorf("expected type %s, but got decimal (%s)", fieldType, assertedValue.String())
		}
	default:
		return fmt.Errorf("expected type %s, but got unexpected type %T(%v)", fieldType, assertedValue, assertedValue)
	}
	return nil
}
