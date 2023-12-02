package eval

import (
	"fmt"
)

func callFmtSprintf(args []any) (any, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("cannot evaluate fmt.Sprintf(), requires at least 2 args, %d supplied", len(args))
	}
	argString0, ok0 := args[0].(string)
	if !ok0 {
		return nil, fmt.Errorf("cannot convert fmt.Sprintf() arg %v to string", args[0])
	}
	afterStringArgs := make([]any, len(args)-1)
	copy(afterStringArgs, args[1:])
	return fmt.Sprintf(argString0, afterStringArgs...), nil
}
