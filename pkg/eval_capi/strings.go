package eval_capi

import (
	"fmt"
	"strings"

	"github.com/capillariesio/capillaries/pkg/eval"
)

func callStringsReplaceAll(args []any) (any, error) {
	if err := eval.CheckArgs("strings.ReplaceAll", 3, len(args)); err != nil {
		return nil, err
	}
	argString0, ok0 := args[0].(string)
	argString1, ok1 := args[1].(string)
	argString2, ok2 := args[2].(string)
	if !ok0 || !ok1 || !ok2 {
		return nil, fmt.Errorf("cannot convert strings.ReplaceAll() args %v,%v,%v to string", args[0], args[1], args[2])
	}
	return strings.ReplaceAll(argString0, argString1, argString2), nil
}
