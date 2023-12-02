package eval

import (
	"fmt"
	"strconv"

	"github.com/shopspring/decimal"
)

func callString(args []any) (any, error) {
	if err := checkArgs("string", 1, len(args)); err != nil {
		return nil, err
	}
	return fmt.Sprintf("%v", args[0]), nil
}

func callInt(args []any) (any, error) {
	if err := checkArgs("int", 1, len(args)); err != nil {
		return nil, err
	}

	switch typedArg0 := args[0].(type) {
	case string:
		retVal, err := strconv.ParseInt(typedArg0, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("cannot eval int(%s):%s", typedArg0, err.Error())
		}
		return retVal, nil

	case bool:
		if typedArg0 {
			return int64(1), nil
		} else {
			return int64(0), nil
		}

	case int:
		return int64(typedArg0), nil

	case int32:
		return int64(typedArg0), nil

	case int16:
		return int64(typedArg0), nil

	case int64:
		return typedArg0, nil

	case float32:
		return int64(typedArg0), nil

	case float64:
		return int64(typedArg0), nil

	case decimal.Decimal:
		return (*typedArg0.BigInt()).Int64(), nil

	default:
		return nil, fmt.Errorf("unsupported arg type for int(%v):%T", typedArg0, typedArg0)
	}
}

func callDecimal2(args []any) (any, error) {
	if err := checkArgs("decimal2", 1, len(args)); err != nil {
		return nil, err
	}

	switch typedArg0 := args[0].(type) {
	case string:
		retVal, err := decimal.NewFromString(typedArg0)
		if err != nil {
			return nil, fmt.Errorf("cannot eval decimal2(%s):%s", typedArg0, err.Error())
		}
		return retVal, nil

	case bool:
		if typedArg0 {
			return decimal.NewFromInt(1), nil
		} else {
			return decimal.NewFromInt(0), nil
		}

	case int:
		return decimal.NewFromInt(int64(typedArg0)), nil

	case int16:
		return decimal.NewFromInt(int64(typedArg0)), nil

	case int32:
		return decimal.NewFromInt(int64(typedArg0)), nil

	case int64:
		return decimal.NewFromInt(typedArg0), nil

	case float32:
		return decimal.NewFromFloat(float64(typedArg0)), nil

	case float64:
		return decimal.NewFromFloat(typedArg0), nil

	case decimal.Decimal:
		return typedArg0, nil

	default:
		return nil, fmt.Errorf("unsupported arg type for decimal2(%v):%T", typedArg0, typedArg0)
	}
}

func callFloat(args []any) (any, error) {
	if err := checkArgs("float", 1, len(args)); err != nil {
		return nil, err
	}
	switch typedArg0 := args[0].(type) {
	case string:
		retVal, err := strconv.ParseFloat(typedArg0, 64)
		if err != nil {
			return nil, fmt.Errorf("cannot eval float(%s):%s", typedArg0, err.Error())
		}
		return retVal, nil
	case bool:
		if typedArg0 {
			return float64(1), nil
		} else {
			return float64(0), nil
		}

	case int:
		return float64(typedArg0), nil

	case int16:
		return float64(typedArg0), nil

	case int32:
		return float64(typedArg0), nil

	case int64:
		return float64(typedArg0), nil

	case float32:
		return float64(typedArg0), nil

	case float64:
		return typedArg0, nil

	case decimal.Decimal:
		valFloat, _ := typedArg0.Float64()
		return valFloat, nil
	default:
		return nil, fmt.Errorf("unsupported arg type for float(%v):%T", typedArg0, typedArg0)
	}
}
