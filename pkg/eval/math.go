package eval

import (
	"fmt"
	"math"
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
