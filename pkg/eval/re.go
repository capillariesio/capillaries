package eval

import (
	"fmt"
	"regexp"
)

func callReMatchString(args []any) (any, error) {
	if err := checkArgs("re.MatchString", 2, len(args)); err != nil {
		return nil, err
	}
	argString0, ok0 := args[0].(string)
	argString1, ok1 := args[1].(string)
	if !ok0 || !ok1 {
		return nil, fmt.Errorf("cannot convert re.MatchString() args %v and %v to string", args[0], args[1])
	}
	return regexp.MatchString(argString0, argString1)
}
