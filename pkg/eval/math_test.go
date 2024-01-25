package eval

import "testing"

func TestMathFunctions(t *testing.T) {
	varValuesMap := VarValuesMap{}
	assertEqual(t, `len("aaa")`, 3, varValuesMap)
	assertEvalError(t, "len(123)", "cannot convert len() arg 123 to string", varValuesMap)
	assertEvalError(t, "len(123,567)", "cannot evaluate len(), requires 1 args, 2 supplied", varValuesMap)

	assertEqual(t, "math.Sqrt(5)", 2.23606797749979, varValuesMap)
	assertEvalError(t, `math.Sqrt("aa")`, "cannot evaluate math.Sqrt(), invalid args [aa]: [cannot cast aa(string) to float64, unsuported type]", varValuesMap)
	assertFloatNan(t, "math.Sqrt(-1)", varValuesMap)
	assertEvalError(t, "math.Sqrt(123,567)", "cannot evaluate math.Sqrt(), requires 1 args, 2 supplied", varValuesMap)

	assertEqual(t, "math.Round(5.1)", 5.0, varValuesMap)
	assertEvalError(t, `math.Round("aa")`, "cannot evaluate math.Round(), invalid args [aa]: [cannot cast aa(string) to float64, unsuported type]", varValuesMap)
	assertEvalError(t, "math.Round(5,1)", "cannot evaluate math.Round(), requires 1 args, 2 supplied", varValuesMap)
}

func TestIntFunctions(t *testing.T) {
	varValuesMap := VarValuesMap{}
	assertEqual(t, `int.iif(true,1,0)`, int64(1), varValuesMap)
	assertEqual(t, `int.iif(false,1,0)`, int64(0), varValuesMap)
	assertEvalError(t, "int.iif(true,1)", "requires 3 args, 2 supplied", varValuesMap)
	assertEvalError(t, "int.iif(1,2,3)", "invalid args", varValuesMap)
}
