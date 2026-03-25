package gocqlmem

import (
	"bytes"
	"cmp"
	"errors"
	"fmt"
	"math"
	"math/big"
	"strconv"
	"strings"
	"time"

	gocql "github.com/apache/cassandra-gocql-driver/v2"
	"github.com/capillariesio/capillaries/pkg/eval"
	"github.com/shopspring/decimal"
	"gopkg.in/inf.v0"
)

func stringToType(s string) (gocql.Type, error) {
	switch strings.ToLower(s) {
	case string(DataTypeAscii):
		return gocql.TypeAscii, nil
	case string(DataTypeBigint):
		return gocql.TypeBigInt, nil
	case string(DataTypeBlob):
		return gocql.TypeBlob, nil
	case string(DataTypeBoolean):
		return gocql.TypeBoolean, nil
	case string(DataTypeCounter):
		return gocql.TypeCounter, nil
	case string(DataTypeDate):
		return gocql.TypeDate, nil
	case string(DataTypeDecimal):
		return gocql.TypeDecimal, nil
	case string(DataTypeDouble):
		return gocql.TypeDouble, nil
	case string(DataTypeFloat):
		return gocql.TypeFloat, nil
	case string(DataTypeInt):
		return gocql.TypeInt, nil
	case string(DataTypeSmallint):
		return gocql.TypeSmallInt, nil
	case string(DataTypeText):
		return gocql.TypeText, nil
	case string(DataTypeTime):
		return gocql.TypeTime, nil
	case string(DataTypeTimestamp):
		return gocql.TypeTimestamp, nil
	case string(DataTypeTimeuuid):
		return gocql.TypeTimeUUID, nil
	case string(DataTypeTinyint):
		return gocql.TypeTinyInt, nil
	case string(DataTypeUuid):
		return gocql.TypeUUID, nil
	case string(DataTypeVarchar):
		return gocql.TypeVarchar, nil
	case string(DataTypeVarint):
		return gocql.TypeVarint, nil
	default:
		return gocql.TypeCustom, fmt.Errorf("unknown type %s", s)
	}
}

func isValidDataType(typ string) bool {
	_, err := stringToType(typ)
	return err == nil
}

// Used in UPDATE and INSERT, when gocql type info is avalable
func sanitizeToInternalKnownType(val any, cqlType gocql.Type) (any, error) {
	// We assume that nils are allowed for this column
	if val == nil {
		return nil, nil
	}

	switch cqlType {
	case gocql.TypeInt, gocql.TypeBigInt, gocql.TypeTinyInt, gocql.TypeSmallInt, gocql.TypeVarint, gocql.TypeCounter, gocql.TypeDate, gocql.TypeTime:
		switch typedVal := val.(type) {
		case inf.Dec, *inf.Dec:
			return 0, fmt.Errorf("cannot implicitly cast inf.Dec(%v) to int64, use cast() function", typedVal)
			// return int64(math.Pow(10, float64(-typedVal.Scale())) * float64(typedVal.UnscaledBig().Int64())), nil
		default:
			return eval.CastToInt64(val)
		}

	case gocql.TypeDouble, gocql.TypeFloat:
		switch typedVal := val.(type) {
		case inf.Dec:
			return decToFloat64(&typedVal), nil
		case *inf.Dec:
			return decToFloat64(typedVal), nil
		default:
			return eval.CastToFloat64(val)
		}

	case gocql.TypeDecimal:
		switch typedVal := val.(type) {
		case inf.Dec:
			// Not sure if gocql does this, maybe unnecessary, but should not hurt
			return decimal.NewFromFloat(decToFloat64(&typedVal)), nil
		case *inf.Dec:
			// Gocql does this, it seems
			return decimal.NewFromFloat(decToFloat64(typedVal)), nil
		default:
			return eval.CastToDecimal(val)
		}

	case gocql.TypeBoolean:
		typedVal, ok := any(val).(bool)
		if !ok {
			return 0, fmt.Errorf("cast %v to bool failed", val)
		}
		return typedVal, nil

	case gocql.TypeTimeUUID, gocql.TypeUUID:
		typedVal, ok := any(val).(gocql.UUID)
		if !ok {
			return 0, fmt.Errorf("cast %v to UUID failed", val)
		}
		return typedVal.Bytes(), nil

	case gocql.TypeBlob:
		typedVal, ok := any(val).([]byte)
		if !ok {
			return 0, fmt.Errorf("cast %v to []byte failed", val)
		}
		return typedVal, nil

	case gocql.TypeText, gocql.TypeVarchar, gocql.TypeAscii:
		typedVal, ok := any(val).(string)
		if !ok {
			return 0, fmt.Errorf("cast %v to string failed", val)
		}
		return typedVal, nil

	case gocql.TypeTimestamp:
		typedVal, ok := any(val).(time.Time)
		if !ok {
			typedValInt64, ok := any(val).(int64)
			if !ok {
				return 0, fmt.Errorf("cast %v to time.Time failed", val)
			}
			return time.UnixMilli(typedValInt64), nil

		}
		return typedVal, nil

	default:
		return 0, fmt.Errorf("unknown column type %v", cqlType)
	}
}

// Used when looking for a place to upsert
func compareInternalKnownType(left any, right any, cqlType gocql.Type) (int, error) {
	if left == nil {
		return 0, errors.New("left is nil, not allowed in partition/clustering key comparison, dev error")
	}
	if right == nil {
		return 0, errors.New("right is nil, not allowed in partition/clustering key comparison, dev error")
	}
	switch cqlType {
	case gocql.TypeInt, gocql.TypeBigInt, gocql.TypeTinyInt, gocql.TypeSmallInt, gocql.TypeCounter, gocql.TypeTime, gocql.TypeDate:
		typedLeft, okLeft := any(left).(int64)
		if !okLeft {
			return 0, fmt.Errorf("left cast %v to int64 failed", left)
		}
		typedRight, okRight := any(right).(int64)
		if !okRight {
			return 0, fmt.Errorf("right cast %v to int64 failed", right)
		}
		return cmp.Compare(typedLeft, typedRight), nil

	case gocql.TypeDouble, gocql.TypeFloat:
		typedLeft, okLeft := any(left).(float64)
		if !okLeft {
			return 0, fmt.Errorf("left cast %v to float64 failed", left)
		}
		typedRight, okRight := any(right).(float64)
		if !okRight {
			return 0, fmt.Errorf("right cast %v to float64 failed", right)
		}
		return cmp.Compare(typedLeft, typedRight), nil

	// case gocql.TypeDecimal:
	// 	typedLeft, okLeft := any(left).(inf.Dec)
	// 	if !okLeft {
	// 		return 0, fmt.Errorf("left cast %v to decimal failed", left)
	// 	}
	// 	typedRight, okRight := any(right).(inf.Dec)
	// 	if !okRight {
	// 		return 0, fmt.Errorf("right cast %v to decimal failed", right)
	// 	}
	// 	return typedLeft.Cmp(&typedRight), nil

	case gocql.TypeDecimal:
		typedLeft, okLeft := any(left).(decimal.Decimal)
		if !okLeft {
			return 0, fmt.Errorf("left cast %v to decimal failed", left)
		}
		typedRight, okRight := any(right).(decimal.Decimal)
		if !okRight {
			return 0, fmt.Errorf("right cast %v to decimal failed", right)
		}
		return typedLeft.Compare(typedRight), nil

	case gocql.TypeTimeUUID, gocql.TypeUUID:
		// Internal representation is []byte, so stick with it
		typedLeft, okLeft := any(left).([]byte)
		if !okLeft {
			return 0, fmt.Errorf("left cast %v to uuid failed", left)
		}
		typedRight, okRight := any(right).([]byte)
		if !okRight {
			return 0, fmt.Errorf("right cast %v to uuid failed", right)
		}
		return bytes.Compare(typedLeft, typedRight), nil

	case gocql.TypeBlob:
		typedLeft, okLeft := any(left).([]byte)
		if !okLeft {
			return 0, fmt.Errorf("left cast %v to []byte failed", left)
		}
		typedRight, okRight := any(right).([]byte)
		if !okRight {
			return 0, fmt.Errorf("right cast %v to []byte failed", right)
		}
		return bytes.Compare(typedLeft, typedRight), nil

	case gocql.TypeTimestamp:
		typedLeft, okLeft := any(left).(time.Time)
		if !okLeft {
			return 0, fmt.Errorf("left cast %v to timestamp failed", left)
		}
		typedRight, okRight := any(right).(time.Time)
		if !okRight {
			return 0, fmt.Errorf("right cast %v to timestamp failed", right)
		}
		return typedLeft.Compare(typedRight), nil

	case gocql.TypeBoolean:
		typedLeft, okLeft := any(left).(bool)
		if !okLeft {
			return 0, fmt.Errorf("left cast %v to bool failed", left)
		}
		typedRight, okRight := any(right).(bool)
		if !okRight {
			return 0, fmt.Errorf("right cast %v to bool failed", right)
		}
		if typedLeft && !typedRight {
			return 1, nil
		} else if !typedLeft && typedRight {
			return -1, nil
		}
		return 0, nil

	case gocql.TypeText, gocql.TypeVarchar, gocql.TypeAscii:
		typedLeft, okLeft := any(left).(string)
		if !okLeft {
			return 0, fmt.Errorf("left cast %v to string failed", left)
		}
		typedRight, okRight := any(right).(string)
		if !okRight {
			return 0, fmt.Errorf("right cast %v to string failed", right)
		}
		return cmp.Compare(typedLeft, typedRight), nil

	default:
		return 0, fmt.Errorf("unsupported column type %v", cqlType)
	}
}

// gocql type matching on data return
func clientTypedValueToProvidedPtr(src any, destPtr any) error {
	switch typedSrc := src.(type) {
	case int64:
		switch typedDestPtr := destPtr.(type) {
		case *int64:
			*typedDestPtr = typedSrc
		default:
			return fmt.Errorf("cannot store int64 %v(%T) to %T", typedSrc, typedSrc, destPtr)
		}
	case int32:
		switch typedDestPtr := destPtr.(type) {
		case *int:
			*typedDestPtr = int(typedSrc)
		case *int32:
			*typedDestPtr = typedSrc
		case *int64:
			*typedDestPtr = int64(typedSrc)
		default:
			return fmt.Errorf("cannot store int32/int %v(%T) to %T", typedSrc, typedSrc, destPtr)
		}
	case int16:
		switch typedDestPtr := destPtr.(type) {
		case *int16:
			*typedDestPtr = typedSrc
		case *int:
			*typedDestPtr = int(typedSrc)
		case *int32:
			*typedDestPtr = int32(typedSrc)
		case *int64:
			*typedDestPtr = int64(typedSrc)
		default:
			return fmt.Errorf("cannot store int16 %v(%T) to %T", typedSrc, typedSrc, destPtr)
		}
	case int8:
		switch typedDestPtr := destPtr.(type) {
		case *int8:
			*typedDestPtr = typedSrc
		case *int16:
			*typedDestPtr = int16(typedSrc)
		case *int:
			*typedDestPtr = int(typedSrc)
		case *int32:
			*typedDestPtr = int32(typedSrc)
		case *int64:
			*typedDestPtr = int64(typedSrc)
		default:
			return fmt.Errorf("cannot store int8 %v(%T) to %T", typedSrc, typedSrc, destPtr)
		}
	case float64:
		switch typedDestPtr := destPtr.(type) {
		case *float64:
			*typedDestPtr = typedSrc
		default:
			return fmt.Errorf("cannot store float64 %v(%T) to %T", typedSrc, typedSrc, destPtr)
		}
	case float32:
		switch typedDestPtr := destPtr.(type) {
		case *float32:
			*typedDestPtr = typedSrc
		case *float64:
			*typedDestPtr = float64(typedSrc)
		default:
			return fmt.Errorf("cannot store float32 %v(%T) to %T", typedSrc, typedSrc, destPtr)
		}
	case bool:
		switch typedDestPtr := destPtr.(type) {
		case *bool:
			*typedDestPtr = typedSrc
		default:
			return fmt.Errorf("cannot store bool %v(%T) to %T", typedSrc, typedSrc, destPtr)
		}
	case string:
		switch typedDestPtr := destPtr.(type) {
		case *string:
			*typedDestPtr = typedSrc
		default:
			return fmt.Errorf("cannot store string %v(%T) to %T", typedSrc, typedSrc, destPtr)
		}
	case inf.Dec:
		switch typedDestPtr := destPtr.(type) {
		case *inf.Dec:
			*typedDestPtr = typedSrc
		default:
			return fmt.Errorf("cannot store decimal %v(%T) to %T", typedSrc, typedSrc, destPtr)
		}
	case time.Time:
		switch typedDestPtr := destPtr.(type) {
		case *time.Time:
			*typedDestPtr = typedSrc
		default:
			return fmt.Errorf("cannot store time.Time  %v(%T) to %T", typedSrc, typedSrc, destPtr)
		}
	case gocql.UUID:
		switch typedDestPtr := destPtr.(type) {
		case *gocql.UUID:
			*typedDestPtr = typedSrc
		default:
			return fmt.Errorf("cannot store UUID/TimeUUID  %v(%T) to %T", typedSrc, typedSrc, destPtr)
		}
	case []byte:
		switch typedDestPtr := destPtr.(type) {
		case *[]byte:
			*typedDestPtr = bytes.Clone(typedSrc)
		// Unnecessary
		// case *gocql.UUID:
		// 	uuidVal, err := gocql.UUIDFromBytes(typedSrc)
		// 	if err != nil {
		// 		return fmt.Errorf("cannot convert []byte  %v(%T) to UUID/TimeUUID", typedSrc, typedSrc)
		// 	}
		// 	*typedDestPtr = uuidVal
		default:
			return fmt.Errorf("cannot store []byte  %v(%T) to %T", typedSrc, typedSrc, destPtr)
		}
	default:
		return fmt.Errorf("cannot store %v(%T) to %T, type not supported", src, src, destPtr)
	}
	return nil
}

func guessInternalValueType(val any) (gocql.Type, error) {
	switch val.(type) {
	case int64:
		return gocql.TypeBigInt, nil // int8, int16 etc?
	case float64:
		return gocql.TypeDouble, nil // float32?
	case bool:
		return gocql.TypeBoolean, nil
	case string:
		return gocql.TypeText, nil // Varchar, ascii?
	case decimal.Decimal:
		return gocql.TypeDecimal, nil
	case []byte:
		return gocql.TypeBlob, nil // What if it's UUID/TimeUUID?
	default:
		return gocql.TypeCustom, fmt.Errorf("unexpected internal type %T", val)
	}
}

/*
func guessClientValueType(val any) (gocql.Type, error) {
	switch val.(type) {
	case int:
		return gocql.TypeInt, nil
	case int8:
		return gocql.TypeCustom, fmt.Errorf("bla unexpected client type %T(%v)", val, val)
		// return gocql.TypeTinyInt, nil
	case int16:
		return gocql.TypeSmallInt, nil
	case int32:
		return gocql.TypeInt, nil
	case int64:
		return gocql.TypeBigInt, nil
	case float32:
		return gocql.TypeFloat, nil
	case float64:
		return gocql.TypeDouble, nil
	case bool:
		return gocql.TypeBoolean, nil
	case string:
		return gocql.TypeText, nil
	case inf.Dec:
		return gocql.TypeDecimal, nil
	case gocql.UUID:
		return gocql.TypeUUID, nil // Or TypeTimeUUID?
	case []byte:
		return gocql.TypeBlob, nil
	case time.Time:
		return gocql.TypeTimestamp, nil
	default:
		return gocql.TypeCustom, fmt.Errorf("unexpected client type %T(%v)", val, val)
	}
}
*/

func columnInfosToColumnNames(columnInfos []gocql.ColumnInfo) []string {
	result := make([]string, len(columnInfos))
	for i := range len(columnInfos) {
		result[i] = columnInfos[i].Name
	}
	return result
}

func namesAndTypeInfosTocolumnInfos(ks string, table string, columnNames []string, typeInfos []gocql.TypeInfo) []gocql.ColumnInfo {
	result := make([]gocql.ColumnInfo, len(columnNames))
	for i := range len(columnNames) {
		result[i] = gocql.ColumnInfo{
			Keyspace: ks,
			Table:    table,
			Name:     columnNames[i],
			TypeInfo: typeInfos[i],
		}
	}
	return result
}

func decToFloat64(d *inf.Dec) float64 {
	unscaled := d.UnscaledBig()
	scale := d.Scale()
	unscaledFloat := new(big.Float).SetInt(unscaled)
	f64, _ := unscaledFloat.Float64() // Convert big.Float to float64
	divisor := math.Pow10(int(scale))
	return f64 / divisor
}

func float64ToDec(f float64) (*inf.Dec, error) {
	valDec, ok := new(inf.Dec).SetString(strconv.FormatFloat(f, 'f', -1, 64))
	if !ok {
		return nil, fmt.Errorf("cannot convert float64 %f to decimal", f)
	}
	return valDec, nil
}

func float64ToDecNoCheck(f float64) *inf.Dec {
	d, _ := float64ToDec(f)
	return d
}

// Used when execSelect/Update/delete adds prepared query params to value map for a parsed CQL expression
func sanitizeToInternalType(val any) (any, error) {
	// We assume that nils are allowed for this column
	if val == nil {
		return nil, nil
	}

	switch typedVal := val.(type) {
	case int, int8, int16, int32, int64:
		return eval.CastToInt64(val)

	case float32, float64:
		return eval.CastToFloat64(val)

	case inf.Dec:
		return decimal.NewFromString(typedVal.String())

	case *inf.Dec:
		return decimal.NewFromString(typedVal.String())

	case bool:
		typedVal, ok := any(val).(bool)
		if !ok {
			return 0, fmt.Errorf("cast %v to bool failed", val)
		}
		return typedVal, nil

	case gocql.UUID:
		return typedVal.Bytes(), nil

	case []byte:
		return typedVal, nil

	case string:
		return typedVal, nil

	case time.Time:
		return typedVal, nil

	default:
		return 0, fmt.Errorf("cannot sanitize to internal type %T(%v)", typedVal, typedVal)
	}
}

// Used when calculating IN and NOT IN expressions, arg types can be anything
func compareInternalInExpressions(left any, right any) bool {
	if left == nil && right == nil {
		return true
	}
	if left == nil && right != nil || left != nil && right == nil {
		return true
	}

	// left and right are guaranteed to be internal, so no int8 mess
	switch typedLeft := left.(type) {
	case int64:
		switch typedRight := right.(type) {
		case int64:
			return typedLeft == typedRight
		case float64:
			return float64(typedLeft) == typedRight
		case decimal.Decimal:
			return decimal.NewFromInt(typedLeft).Equal(typedRight)
		default:
			return false
		}

	case float64:
		switch typedRight := right.(type) {
		case float64:
			return typedLeft == typedRight
		case int64:
			return typedLeft == float64(typedRight)
		case decimal.Decimal:
			return decimal.NewFromFloat(typedLeft).Equal(typedRight)
		default:
			return false
		}

	case decimal.Decimal:
		switch typedRight := right.(type) {
		case int64:
			return typedLeft.Equal(decimal.NewFromInt(typedRight))
		case float64:
			return typedLeft.Equal(decimal.NewFromFloat(typedRight))
		case decimal.Decimal:
			return typedLeft.Equal(typedRight)
		default:
			return false
		}

	case gocql.UUID:
		switch typedRight := right.(type) {
		case gocql.UUID:
			return typedLeft.String() == typedRight.String()
		default:
			return false
		}

	case []byte:
		switch typedRight := right.(type) {
		case []byte:
			return bytes.Equal(typedLeft, typedRight)
		default:
			return false
		}

	case bool:
		switch typedRight := right.(type) {
		case bool:
			return typedLeft == typedRight
		default:
			return false
		}

	case string:
		switch typedRight := right.(type) {
		case string:
			return typedLeft == typedRight
		default:
			return false
		}
	default:
		return false
	}
}

func internalValueToClientType(val any, typ gocql.Type) (any, error) {
	switch typedInternalVal := val.(type) {
	case int64:
		switch typ {
		case gocql.TypeTinyInt:
			return int8(typedInternalVal), nil
		case gocql.TypeSmallInt:
			return int16(typedInternalVal), nil
		case gocql.TypeInt, gocql.TypeDate:
			return int32(typedInternalVal), nil
		case gocql.TypeBigInt, gocql.TypeVarint, gocql.TypeCounter, gocql.TypeTime:
			return typedInternalVal, nil
		default:
			// Give up and pray
			return val, nil
		}

	case float64:
		switch typ {
		case gocql.TypeFloat:
			return float32(typedInternalVal), nil
		case gocql.TypeDouble:
			return typedInternalVal, nil
		default:
			// Give up and pray
			return val, nil
		}

	case decimal.Decimal:
		switch typ {
		case gocql.TypeDecimal:
			s := typedInternalVal.String()
			infDecVal, ok := new(inf.Dec).SetString(s)
			if !ok {
				return nil, fmt.Errorf("cannot convert decimal %v(%T) to inf.Dec from string %s", typedInternalVal, typedInternalVal, s)
			}
			return *infDecVal, nil
		default:
			// Give up and pray
			return val, nil
		}
	case []byte:
		switch typ {
		case gocql.TypeBlob:
			return typedInternalVal, nil
		case gocql.TypeUUID, gocql.TypeTimeUUID:
			uuid, err := gocql.UUIDFromBytes(typedInternalVal)
			if err != nil {
				return nil, fmt.Errorf("cannot []byte %v(%T) to UUID/TimeUUID: %s", typedInternalVal, typedInternalVal, err.Error())
			}
			return uuid, nil
		default:
			// Give up and pray
			return val, nil
		}
	default:
		// Give up and pray
		return val, nil

	}
}
