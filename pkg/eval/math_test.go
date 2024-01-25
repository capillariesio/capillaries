package eval

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

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

func TestFloatFunctions(t *testing.T) {
	varValuesMap := VarValuesMap{}
	assertEqual(t, `float.iif(true,1.0,0.0)`, float64(1), varValuesMap)
	assertEqual(t, `float.iif(false,1.0,0.0)`, float64(0), varValuesMap)
	assertEvalError(t, "float.iif(true,1.0)", "requires 3 args, 2 supplied", varValuesMap)
	assertEvalError(t, "float.iif(1.0,2.0,3.0)", "invalid args", varValuesMap)
}

func TestDecimal2Functions(t *testing.T) {
	varValuesMap := VarValuesMap{}
	assertEqual(t, `decimal2.iif(true, decimal2(1.0),decimal2(0.0))`, decimal.NewFromFloat(1), varValuesMap)
	assertEqual(t, `decimal2.iif(false,decimal2(1.0),decimal2(0.0))`, decimal.NewFromFloat(0), varValuesMap)
	assertEvalError(t, "decimal2.iif(true,decimal2(1.0))", "requires 3 args, 2 supplied", varValuesMap)
	assertEvalError(t, "decimal2.iif(decimal2(1.0),decimal2(2.0),decimal2(3.0))", "invalid args", varValuesMap)
}

func TestStringFunctions(t *testing.T) {
	varValuesMap := VarValuesMap{}
	assertEqual(t, `string.iif(true,"a","b")`, "a", varValuesMap)
	assertEqual(t, `string.iif(false,"a","b")`, "b", varValuesMap)
	assertEvalError(t, `string.iif(true,"a")`, "requires 3 args, 2 supplied", varValuesMap)
	assertEvalError(t, `string.iif("a","b","c")`, "invalid args", varValuesMap)
}

func TestTimeFunctions(t *testing.T) {
	testTimeA := time.Date(2001, 1, 1, 1, 1, 1, 100000000, time.FixedZone("", -7200))
	testTimeB := time.Date(2001, 1, 1, 0, 0, 0, 0, time.UTC)
	varValuesMap := VarValuesMap{"t": map[string]any{"test_time_a": testTimeA, "test_time_b": testTimeB}}
	assertEqual(t, `time.iif(true,t.test_time_a,t.test_time_b)`, testTimeA, varValuesMap)
	assertEqual(t, `time.iif(false,t.test_time_a,t.test_time_b)`, testTimeB, varValuesMap)
	assertEvalError(t, `time.iif(true,t.test_time_a)`, "requires 3 args, 2 supplied", varValuesMap)
	assertEvalError(t, `time.iif(t.test_time_a,t.test_time_b,t.test_time_a)`, "invalid args", varValuesMap)
}
