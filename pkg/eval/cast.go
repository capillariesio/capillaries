package eval

import (
	"fmt"

	"github.com/shopspring/decimal"
)

func castNumberToStandardType(arg interface{}) (interface{}, error) {
	switch typedArg := arg.(type) {
	case int:
		return int64(typedArg), nil
	case int16:
		return int64(typedArg), nil
	case int32:
		return int64(typedArg), nil
	case int64:
		return typedArg, nil
	case float32:
		return float64(typedArg), nil
	case float64:
		return typedArg, nil
	case decimal.Decimal:
		return typedArg, nil
	default:
		return 0.0, fmt.Errorf("cannot cast %v(%T) to standard number type, unsuported type", typedArg, typedArg)
	}
}

func castNumberPairToCommonType(argLeft interface{}, argRight interface{}) (interface{}, interface{}, error) {
	stdArgLeft, err := castNumberToStandardType(argLeft)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid left arg: %s", err.Error())
	}
	stdArgRight, err := castNumberToStandardType(argRight)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid right arg: %s", err.Error())
	}
	// Check for float64
	_, floatLeft := stdArgLeft.(float64)
	_, floatRight := stdArgRight.(float64)
	if floatLeft || floatRight {
		finalArgLeft, err := castToFloat64(stdArgLeft)
		if err != nil {
			return nil, nil, fmt.Errorf("unexpectedly cannot cast left arg to float64: %s", err.Error())
		}
		finalArgRight, err := castToFloat64(stdArgRight)
		if err != nil {
			return nil, nil, fmt.Errorf("unexpectedly cannot cast right arg to float64: %s", err.Error())
		}
		return finalArgLeft, finalArgRight, nil
	}

	// Check for decimal2
	_, decLeft := stdArgLeft.(decimal.Decimal)
	_, decRight := stdArgRight.(decimal.Decimal)
	if decLeft || decRight {
		finalArgLeft, err := castToDecimal2(stdArgLeft)
		if err != nil {
			return nil, nil, fmt.Errorf("unexpectedly cannot cast left arg to decimal2: %s", err.Error())
		}
		finalArgRight, err := castToDecimal2(stdArgRight)
		if err != nil {
			return nil, nil, fmt.Errorf("unexpectedly cannot cast right arg to decimal2: %s", err.Error())
		}
		return finalArgLeft, finalArgRight, nil
	}

	// Cast both to int64
	finalArgLeft, err := castToInt64(stdArgLeft)
	if err != nil {
		return nil, nil, fmt.Errorf("unexpectedly cannot cast left arg to int64: %s", err.Error())
	}
	finalArgRight, err := castToInt64(stdArgRight)
	if err != nil {
		return nil, nil, fmt.Errorf("unexpectedly cannot cast right arg to int64: %s", err.Error())
	}
	return finalArgLeft, finalArgRight, nil
}

func castToInt64(arg interface{}) (int64, error) {
	switch typedArg := arg.(type) {
	case int:
		return int64(typedArg), nil
	case int16:
		return int64(typedArg), nil
	case int32:
		return int64(typedArg), nil
	case int64:
		return typedArg, nil
	case float32:
		return int64(typedArg), nil
	case float64:
		return int64(typedArg), nil
	case decimal.Decimal:
		if typedArg.IsInteger() {
			return typedArg.BigInt().Int64(), nil
		} else {
			return 0.0, fmt.Errorf("cannot cast decimal '%v' to int64, exact conversion impossible", typedArg)
		}
	default:
		return 0.0, fmt.Errorf("cannot cast %v(%T) to int64, unsuported type", typedArg, typedArg)
	}
}

func castToFloat64(arg interface{}) (float64, error) {
	switch typedArg := arg.(type) {
	case int:
		return float64(typedArg), nil
	case int16:
		return float64(typedArg), nil
	case int32:
		return float64(typedArg), nil
	case int64:
		return float64(typedArg), nil
	case float32:
		return float64(typedArg), nil
	case float64:
		return typedArg, nil
	case decimal.Decimal:
		valFloat, _ := typedArg.Float64()
		return valFloat, nil
	default:
		return 0.0, fmt.Errorf("cannot cast %v(%T) to float64, unsuported type", typedArg, typedArg)
	}
}

func castToDecimal2(arg interface{}) (decimal.Decimal, error) {
	switch typedArg := arg.(type) {
	case int:
		return decimal.NewFromInt(int64(typedArg)), nil
	case int16:
		return decimal.NewFromInt(int64(typedArg)), nil
	case int32:
		return decimal.NewFromInt(int64(typedArg)), nil
	case int64:
		return decimal.NewFromInt(typedArg), nil
	case float32:
		return decimal.NewFromFloat32(typedArg), nil
	case float64:
		return decimal.NewFromFloat(typedArg), nil
	case decimal.Decimal:
		return typedArg, nil
	default:
		return decimal.NewFromInt(0), fmt.Errorf("cannot cast %v(%T) to decimal2, unsuported type", typedArg, typedArg)
	}
}
