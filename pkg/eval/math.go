package eval

import (
	"fmt"
	"math"
	"time"

	"github.com/shopspring/decimal"
)

func callLen(args []any) (any, error) {
	if err := checkArgs("len", 1, len(args)); err != nil {
		return nil, err
	}
	argString, ok := args[0].(string)
	if !ok {
		return nil, fmt.Errorf("cannot convert len() arg %v to string", args[0])
	}
	return len(argString), nil
}

func callIntIif(args []any) (any, error) {
	if err := checkArgs("iif", 3, len(args)); err != nil {
		return nil, err
	}
	arg0, ok0 := args[0].(bool)
	arg1, ok1 := args[1].(int64)
	arg2, ok2 := args[2].(int64)
	if !ok0 || !ok1 || !ok2 {
		return nil, fmt.Errorf("cannot evaluate int.Iif(), invalid args %v", args)
	}
	if arg0 {
		return arg1, nil
	}
	return arg2, nil
}

func callFloatIif(args []any) (any, error) {
	if err := checkArgs("iif", 3, len(args)); err != nil {
		return nil, err
	}
	arg0, ok0 := args[0].(bool)
	arg1, ok1 := args[1].(float64)
	arg2, ok2 := args[2].(float64)
	if !ok0 || !ok1 || !ok2 {
		return nil, fmt.Errorf("cannot evaluate float.Iif(), invalid args %v", args)
	}
	if arg0 {
		return arg1, nil
	}
	return arg2, nil
}

func callDecimal2Iif(args []any) (any, error) {
	if err := checkArgs("iif", 3, len(args)); err != nil {
		return nil, err
	}
	arg0, ok0 := args[0].(bool)
	arg1, ok1 := args[1].(decimal.Decimal)
	arg2, ok2 := args[2].(decimal.Decimal)
	if !ok0 || !ok1 || !ok2 {
		return nil, fmt.Errorf("cannot evaluate decimal2.Iif(), invalid args %v", args)
	}
	if arg0 {
		return arg1, nil
	}
	return arg2, nil
}

func callStringIif(args []any) (any, error) {
	if err := checkArgs("iif", 3, len(args)); err != nil {
		return nil, err
	}
	arg0, ok0 := args[0].(bool)
	arg1, ok1 := args[1].(string)
	arg2, ok2 := args[2].(string)
	if !ok0 || !ok1 || !ok2 {
		return nil, fmt.Errorf("cannot evaluate string.Iif(), invalid args %v", args)
	}
	if arg0 {
		return arg1, nil
	}
	return arg2, nil
}

func callTimeIif(args []any) (any, error) {
	if err := checkArgs("iif", 3, len(args)); err != nil {
		return nil, err
	}
	arg0, ok0 := args[0].(bool)
	arg1, ok1 := args[1].(time.Time)
	arg2, ok2 := args[2].(time.Time)
	if !ok0 || !ok1 || !ok2 {
		return nil, fmt.Errorf("cannot evaluate time.Iif(), invalid args %v", args)
	}
	if arg0 {
		return arg1, nil
	}
	return arg2, nil
}

func callMathSqrt(args []any) (any, error) {
	if err := checkArgs("math.Sqrt", 1, len(args)); err != nil {
		return nil, err
	}
	argFloat, err := castToFloat64(args[0])
	if err != nil {
		return nil, fmt.Errorf("cannot evaluate math.Sqrt(), invalid args %v: [%s]", args, err.Error())
	}

	return math.Sqrt(argFloat), nil
}

func callMathRound(args []any) (any, error) {
	if err := checkArgs("math.Round", 1, len(args)); err != nil {
		return nil, err
	}
	argFloat, err := castToFloat64(args[0])
	if err != nil {
		return nil, fmt.Errorf("cannot evaluate math.Round(), invalid args %v: [%s]", args, err.Error())
	}

	return math.Round(argFloat), nil
}
