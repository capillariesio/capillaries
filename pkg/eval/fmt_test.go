package eval

import (
	"testing"
)

func TestFmtFunctions(t *testing.T) {
	varValuesMap := VarValuesMap{}
	assertEqual(t, `fmt.Sprintf("%s %d %.1f","a",2,3.3)`, "a 2 3.3", varValuesMap)
	assertEqual(t, `fmt.Sprintf("%s %d %.1f","a",2)`, "a 2 %!f(MISSING)", varValuesMap)
	assertEvalError(t, `fmt.Sprintf("bla")`, "cannot evaluate fmt.Sprintf(), requires at least 2 args, 1 supplied", varValuesMap)
	assertEvalError(t, `fmt.Sprintf(1,2)`, "cannot convert fmt.Sprintf() arg 1 to string", varValuesMap)
}
