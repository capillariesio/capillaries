package sc

import (
	"fmt"
	"time"

	"github.com/capillariesio/capillaries/pkg/eval_capi"
	"github.com/shopspring/decimal"
	"gopkg.in/inf.v0"
)

const (
	FieldNameUnknown = "unknown_field_name"
)

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
func DefaultDateTime() time.Time         { return time.Date(1, 1, 1, 0, 0, 0, 0, time.UTC) } // Same as time.Time default

func GetDefaultFieldTypeValue(fieldType eval_capi.TableFieldType) any {
	switch fieldType {
	case eval_capi.FieldTypeInt:
		return DefaultInt
	case eval_capi.FieldTypeFloat:
		return DefaultFloat
	case eval_capi.FieldTypeString:
		return DefaultString
	case eval_capi.FieldTypeDecimal2:
		return DefaultDecimal2()
	case eval_capi.FieldTypeBool:
		return DefaultBool
	case eval_capi.FieldTypeDateTime:
		return DefaultDateTime()
	default:
		return nil
	}
}

func CheckValueType(val any, fieldType eval_capi.TableFieldType) error {
	switch assertedValue := val.(type) {
	case int64:
		if fieldType != eval_capi.FieldTypeInt {
			return fmt.Errorf("expected type %s, but got int64 (%d)", fieldType, assertedValue)
		}
	case float64:
		if fieldType != eval_capi.FieldTypeFloat {
			return fmt.Errorf("expected type %s, but got float64 (%f)", fieldType, assertedValue)
		}
	case string:
		if fieldType != eval_capi.FieldTypeString {
			return fmt.Errorf("expected type %s, but got string (%s)", fieldType, assertedValue)
		}
	case bool:
		if fieldType != eval_capi.FieldTypeBool {
			return fmt.Errorf("expected type %s, but got bool (%v)", fieldType, assertedValue)
		}
	case time.Time:
		if fieldType != eval_capi.FieldTypeDateTime {
			return fmt.Errorf("expected type %s, but got datetime (%s)", fieldType, assertedValue.String())
		}
	case decimal.Decimal:
		if fieldType != eval_capi.FieldTypeDecimal2 {
			return fmt.Errorf("expected type %s, but got decimal (%s)", fieldType, assertedValue.String())
		}
	default:
		return fmt.Errorf("expected type %s, but got unexpected type %T(%v)", fieldType, assertedValue, assertedValue)
	}
	return nil
}
