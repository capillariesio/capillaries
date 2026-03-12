package gocqlmem

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"hash"
	"math"
	"strconv"
	"time"

	gocql "github.com/apache/cassandra-gocql-driver/v2"
	"github.com/capillariesio/capillaries/pkg/eval"
	"github.com/shopspring/decimal"
	"github.com/twmb/murmur3"
	"gopkg.in/inf.v0"
)

// Assuming that all CQL functions are lowercase
var GocqlmemEvalFunctions = map[string]eval.EvalFunction{
	"cast":              callCast,
	"token":             callToken,
	"current_timestamp": callCurrentTimestamp,
	"current_date":      callCurrentDate,
	"current_time":      callCurrentTime,
	"abs":               callAbs,
	"exp":               callExp,
	"log":               callLog,
	"log10":             callLog10,
	"round":             callRound,
	"now":               callNow,
	"totimestamp":       callToTimestamp,
}

/*
https://cassandra.apache.org/doc/latest/cassandra/developing/cql/functions.html#cast

YES ascii	text, varchar
YES bigint	tinyint, smallint, int, float, double, decimal, varint, text, varchar
YES boolean	text, varchar
YES counter	tinyint, smallint, int, bigint, float, double, decimal, varint, text, varchar
TODO date	timestamp
YES decimal	tinyint, smallint, int, bigint, float, double, varint, text, varchar
YES double	tinyint, smallint, int, bigint, float, decimal, varint, text, varchar
YES float	tinyint, smallint, int, bigint, double, decimal, varint, text, varchar
TODO inet	text, varchar
YES int	tinyint, smallint, bigint, float, double, decimal, varint, text, varchar
YES smallint	tinyint, int, bigint, float, double, decimal, varint, text, varchar
TODO time	text, varchar
TODO timestamp	date, text, varchar
TODO timeuuid	timestamp, date, text, varchar
YES tinyint	tinyint, smallint, int, bigint, float, double, decimal, varint, text, varchar
TODO uuid	text, varchar
YES varint	tinyint, smallint, int, bigint, float, double, decimal, text, varchar
*/
func callCast(args []any) (any, error) {
	if err := eval.CheckArgs("cast", 2, len(args)); err != nil {
		return nil, err
	}

	dataType, ok := args[1].(CqlDataType)
	if !ok {
		return nil, fmt.Errorf("cannot convert cast() arg %v to DataType", args[1])
	}

	switch typedVal := args[0].(type) {
	case int, int64, int32, int16, int8:
		var typedValInt64 int64
		switch typedVal := typedVal.(type) {
		case int:
			typedValInt64 = int64(typedVal)
		case int8:
			typedValInt64 = int64(typedVal)
		case int16:
			typedValInt64 = int64(typedVal)
		case int32:
			typedValInt64 = int64(typedVal)
		case int64:
			typedValInt64 = typedVal
		}
		switch dataType {
		case DataTypeBigint, DataTypeSmallint, DataTypeTinyint, DataTypeInt, DataTypeVarint:
			return typedValInt64, nil
		case DataTypeFloat, DataTypeDouble:
			return float64(typedValInt64), nil
		case DataTypeDecimal:
			return *inf.NewDec(typedValInt64, 0), nil
		case DataTypeText, DataTypeVarchar:
			return strconv.FormatInt(typedValInt64, 10), nil
		default:
			return nil, fmt.Errorf("cannot cast int %v to %v", typedVal, dataType)
		}
	case float32, float64:
		var typedValFloat64 float64
		switch typedVal := typedVal.(type) {
		case float32:
			typedValFloat64 = float64(typedVal)
		case float64:
			typedValFloat64 = typedVal
		}
		switch dataType {
		case DataTypeBigint, DataTypeSmallint, DataTypeTinyint, DataTypeInt, DataTypeVarint:
			return int64(typedValFloat64), nil
		case DataTypeFloat, DataTypeDouble:
			return float64(typedValFloat64), nil
		case DataTypeDecimal:
			// inf.Dec
			// valDec, err := float64ToDec(typedValFloat64)
			// if !ok {
			// 	return nil, fmt.Errorf("cannot cast float %f to decimal: %s", typedValFloat64, err.Error())
			// }
			// return *valDec, nil
			return decimal.NewFromFloat(typedValFloat64), nil
		case DataTypeText, DataTypeVarchar:
			return strconv.FormatFloat(typedValFloat64, 'f', -1, 64), nil
		default:
			return nil, fmt.Errorf("cannot cast float %v to %v", typedVal, dataType)
		}

	case bool:
		switch dataType {
		case DataTypeText, DataTypeVarchar:
			if typedVal {
				return "TRUE", nil
			} else {
				return "FALSE", nil
			}
		default:
			return nil, fmt.Errorf("cannot cast bool %v to %v", typedVal, dataType)
		}
	// case inf.Dec:
	// 	switch dataType {
	// 	case DataTypeBigint, DataTypeSmallint, DataTypeTinyint, DataTypeInt, DataTypeVarint:
	// 		return int64(math.Pow(10, float64(-typedVal.Scale())) * float64(typedVal.UnscaledBig().Int64())), nil
	// 	case DataTypeFloat, DataTypeDouble:
	// 		return decToFloat64(&typedVal), nil
	// 	case DataTypeDecimal:
	// 		return typedVal, nil
	// 	case DataTypeText, DataTypeVarchar:
	// 		return typedVal.String(), nil
	// 	default:
	// 		return nil, fmt.Errorf("cannot cast decimal %v to %v", typedVal, dataType)
	// 	}

	case decimal.Decimal:
		switch dataType {
		case DataTypeBigint, DataTypeSmallint, DataTypeTinyint, DataTypeInt, DataTypeVarint:
			return typedVal.BigInt().Int64(), nil
		case DataTypeFloat, DataTypeDouble:
			f, _ := typedVal.Float64()
			return f, nil
		case DataTypeDecimal:
			return typedVal, nil
		case DataTypeText, DataTypeVarchar:
			return typedVal.String(), nil
		default:
			return nil, fmt.Errorf("cannot cast decimal %v to %v", typedVal, dataType)
		}

	case string:
		switch dataType {
		case DataTypeText, DataTypeVarchar:
			return typedVal, nil
		default:
			return nil, fmt.Errorf("cannot cast string %v to %v", typedVal, dataType)
		}

	default:
		return nil, fmt.Errorf("cannot cast %v to %v, unsupported source type", args[0], dataType)
	}
}

func callToken(args []any) (any, error) {
	if err := eval.CheckArgs("token", 1, len(args)); err != nil {
		return nil, err
	}
	switch typedVal := args[0].(type) {
	case int, int64, int32, int16, int8:
		var typedValInt64 int64
		switch typedVal := typedVal.(type) {
		case int:
			typedValInt64 = int64(typedVal)
		case int8:
			typedValInt64 = int64(typedVal)
		case int16:
			typedValInt64 = int64(typedVal)
		case int32:
			typedValInt64 = int64(typedVal)
		case int64:
			typedValInt64 = typedVal
		}
		b := make([]byte, 8)
		binary.LittleEndian.PutUint64(b, uint64(typedValInt64))
		var h64 hash.Hash64 = murmur3.New64()
		h64.Write(b)
		return h64.Sum64(), nil

	case float32, float64:
		var typedValFloat64 float64
		switch typedVal := typedVal.(type) {
		case float32:
			typedValFloat64 = float64(typedVal)
		case float64:
			typedValFloat64 = typedVal
		}
		var buf bytes.Buffer
		err := binary.Write(&buf, binary.LittleEndian, typedValFloat64)
		if err != nil {
			return nil, fmt.Errorf("cannot token float %v, binary fails: %s", typedVal, err.Error())
		}
		var h64 hash.Hash64 = murmur3.New64()
		h64.Write(buf.Bytes())
		return h64.Sum64(), nil

	case bool:
		typedValInt64 := 0
		if typedVal {
			typedValInt64 = 1
		}
		b := make([]byte, 8)
		binary.LittleEndian.PutUint64(b, uint64(typedValInt64))
		var h64 hash.Hash64 = murmur3.New64()
		h64.Write(b)
		return h64.Sum64(), nil

	// case inf.Dec:
	// 	b, err := typedVal.GobEncode()
	// 	if err != nil {
	// 		return nil, fmt.Errorf("cannot token decimal %v: %s", typedVal, err.Error())
	// 	}
	// 	var h64 hash.Hash64 = murmur3.New64()
	// 	h64.Write(b)
	// 	return h64.Sum64(), nil

	case decimal.Decimal:
		b, err := typedVal.MarshalBinary()
		if err != nil {
			return nil, fmt.Errorf("cannot token decimal %v: %s", typedVal, err.Error())
		}
		var h64 hash.Hash64 = murmur3.New64()
		h64.Write(b)
		return h64.Sum64(), nil

	case string:
		var h64 hash.Hash64 = murmur3.New64()
		h64.Write([]byte(typedVal))
		return h64.Sum64(), nil
	default:
		return nil, fmt.Errorf("cannot token %v, unsupported source type %T", args[0], args[0])
	}
}

func callCurrentTimestamp(args []any) (any, error) {
	if err := eval.CheckArgs("current_timestamp", 0, len(args)); err != nil {
		return nil, err
	}

	return time.Now().UnixMilli(), nil // Cassandra: millis from epoch
}

func callCurrentDate(args []any) (any, error) {
	if err := eval.CheckArgs("current_date", 0, len(args)); err != nil {
		return nil, err
	}

	dur := time.Since(time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC))
	return int64(dur.Hours() / 24), nil // Cassandra: days since epoch (32-bit, but we keep using int64)
}

func callCurrentTime(args []any) (any, error) {
	if err := eval.CheckArgs("current_time", 0, len(args)); err != nil {
		return nil, err
	}

	// Local time
	ti := time.Now()
	return int64(((ti.Hour()*60+ti.Minute())*60+ti.Second())*1000000000 + ti.Nanosecond()), nil // Cassandra: nanos from midnight
}

func intAbs(src int64) int64 {
	if src < 0 {
		return -src
	} else {
		return src
	}
}

func floatAbs(src float64) float64 {
	if src < 0 {
		return -src
	} else {
		return src
	}
}

func callAbs(args []any) (any, error) {
	if err := eval.CheckArgs("abs", 1, len(args)); err != nil {
		return nil, err
	}
	switch typedVal := args[0].(type) {
	case int:
		return intAbs(int64(typedVal)), nil
	case int8:
		return intAbs(int64(typedVal)), nil
	case int16:
		return intAbs(int64(typedVal)), nil
	case int32:
		return intAbs(int64(typedVal)), nil
	case int64:
		return intAbs(typedVal), nil
	case float32:
		return floatAbs(float64(typedVal)), nil
	case float64:
		return floatAbs(typedVal), nil
	// case inf.Dec:
	// 	return *typedVal.Abs(&typedVal), nil
	case decimal.Decimal:
		return typedVal.Abs(), nil
	default:
		return nil, fmt.Errorf("cannot abs %v, unsupported source type", args[0])
	}
}

func callExp(args []any) (any, error) {
	if err := eval.CheckArgs("exp", 1, len(args)); err != nil {
		return nil, err
	}
	switch typedVal := args[0].(type) {
	case int:
		return math.Exp(float64(typedVal)), nil
	case int8:
		return math.Exp(float64(typedVal)), nil
	case int16:
		return math.Exp(float64(typedVal)), nil
	case int32:
		return math.Exp(float64(typedVal)), nil
	case int64:
		return math.Exp(float64(typedVal)), nil
	case float32:
		return math.Exp(float64(typedVal)), nil
	case float64:
		return math.Exp(typedVal), nil
	// case inf.Dec:
	// 	decVal, err := float64ToDec(math.Exp(decToFloat64(&typedVal)))
	// 	if err != nil {
	// 		return nil, fmt.Errorf("cannot calculate exp for decimal %v: %s", typedVal, err.Error())
	// 	}
	// 	return *decVal, nil
	case decimal.Decimal:
		expVal, err := typedVal.ExpTaylor(17)
		if err != nil {
			return nil, fmt.Errorf("cannot calculate exp for decimal %v: %s", typedVal, err.Error())
		}
		return expVal, nil
	default:
		return nil, fmt.Errorf("cannot exp %v, unsupported source type", args[0])
	}
}

func callLog(args []any) (any, error) {
	if err := eval.CheckArgs("log", 1, len(args)); err != nil {
		return nil, err
	}
	switch typedVal := args[0].(type) {
	case int:
		return math.Log(float64(typedVal)), nil
	case int8:
		return math.Log(float64(typedVal)), nil
	case int16:
		return math.Log(float64(typedVal)), nil
	case int32:
		return math.Log(float64(typedVal)), nil
	case int64:
		return math.Log(float64(typedVal)), nil
	case float32:
		return math.Log(float64(typedVal)), nil
	case float64:
		return math.Log(typedVal), nil
	// case inf.Dec:
	// 	decVal, err := float64ToDec(math.Log(decToFloat64(&typedVal)))
	// 	if err != nil {
	// 		return nil, fmt.Errorf("cannot calculate log for decimal %v: %s", typedVal, err.Error())
	// 	}
	// 	return *decVal, nil
	case decimal.Decimal:
		logVal, err := typedVal.Ln(10)
		if err != nil {
			return nil, fmt.Errorf("cannot calculate log for decimal %v: %s", typedVal, err.Error())
		}
		return logVal, nil
	default:
		return nil, fmt.Errorf("cannot log %v, unsupported source type", args[0])
	}
}

func callLog10(args []any) (any, error) {
	if err := eval.CheckArgs("log10", 1, len(args)); err != nil {
		return nil, err
	}
	switch typedVal := args[0].(type) {
	case int:
		return math.Log10(float64(typedVal)), nil
	case int8:
		return math.Log10(float64(typedVal)), nil
	case int16:
		return math.Log10(float64(typedVal)), nil
	case int32:
		return math.Log10(float64(typedVal)), nil
	case int64:
		return math.Log10(float64(typedVal)), nil
	case float32:
		return math.Log10(float64(typedVal)), nil
	case float64:
		return math.Log10(typedVal), nil
	// case inf.Dec:
	// 	decVal, err := float64ToDec(math.Log10(decToFloat64(&typedVal)))
	// 	if err != nil {
	// 		return nil, fmt.Errorf("cannot calculate log10 for decimal %v: %s", typedVal, err.Error())
	// 	}
	// 	return *decVal, nil
	case decimal.Decimal:
		f, _ := typedVal.Float64()
		return decimal.NewFromFloat(math.Log10(f)), nil
	default:
		return nil, fmt.Errorf("cannot log10 %v, unsupported source type", args[0])
	}
}

func callRound(args []any) (any, error) {
	if err := eval.CheckArgs("round", 1, len(args)); err != nil {
		return nil, err
	}
	switch typedVal := args[0].(type) {
	case int:
		return int64(typedVal), nil
	case int8:
		return int64(typedVal), nil
	case int16:
		return int64(typedVal), nil
	case int32:
		return int64(typedVal), nil
	case int64:
		return typedVal, nil
	case float32:
		return math.Round(float64(typedVal)), nil
	case float64:
		return math.Round(typedVal), nil
	// case inf.Dec:
	// 	return *typedVal.Round(&typedVal, 0, inf.RoundHalfUp), nil
	case decimal.Decimal:
		return typedVal.Round(0), nil
	default:
		return nil, fmt.Errorf("cannot round %v, unsupported source type", args[0])
	}
}

func callNow(args []any) (any, error) {
	if err := eval.CheckArgs("now", 0, len(args)); err != nil {
		return nil, err
	}
	return gocql.TimeUUID(), nil
}

func callToTimestamp(args []any) (any, error) {
	if err := eval.CheckArgs("toTimestamp", 1, len(args)); err != nil {
		return nil, err
	}
	u, ok := args[0].(gocql.UUID)
	if !ok {
		return nil, fmt.Errorf("cannot read timeuuid from %T(%v)", args[0], args[0])
	}
	return u.Time().UnixMilli(), nil // Cassandra ts: millis from epoch
}
