package eval

import (
	"testing"
)

func TestStringsFunctions(t *testing.T) {
	varValuesMap := VarValuesMap{}
	assertEqual(t, `strings.ReplaceAll("abc","a","b")`, "bbc", varValuesMap)
	assertEvalError(t, `strings.ReplaceAll("a","b")`, "cannot evaluate strings.ReplaceAll(), requires 3 args, 2 supplied", varValuesMap)
	assertEvalError(t, `strings.ReplaceAll("a","b",1)`, "cannot convert strings.ReplaceAll() args a,b,1 to string", varValuesMap)
}
