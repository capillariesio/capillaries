package eval

import (
	"fmt"
	"go/parser"
	"go/token"
	"math"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func assertEqual(t *testing.T, expString string, expectedResult interface{}, varValuesMap VarValuesMap) {
	exp, err1 := parser.ParseExpr(expString)
	if err1 != nil {
		t.Error(fmt.Errorf("%s: %s", expString, err1.Error()))
		return
	}
	eCtx := NewPlainEvalCtxWithVars(AggFuncDisabled, &varValuesMap)
	result, err2 := eCtx.Eval(exp)
	if err2 != nil {
		t.Error(fmt.Errorf("%s: %s", expString, err2.Error()))
		return
	}

	assert.Equal(t, expectedResult, result, fmt.Sprintf("Unmatched: %v = %v: %s ", expectedResult, result, expString))
}

func assertFloatNan(t *testing.T, expString string, varValuesMap VarValuesMap) {
	exp, err1 := parser.ParseExpr(expString)
	if err1 != nil {
		t.Error(fmt.Errorf("%s: %s", expString, err1.Error()))
		return
	}
	eCtx := NewPlainEvalCtxWithVars(AggFuncDisabled, &varValuesMap)
	result, err2 := eCtx.Eval(exp)
	if err2 != nil {
		t.Error(fmt.Errorf("%s: %s", expString, err2.Error()))
		return
	}
	floatResult, _ := result.(float64)
	assert.True(t, math.IsNaN(floatResult))
}

func assertEvalError(t *testing.T, expString string, expectedErrorMsg string, varValuesMap VarValuesMap) {
	exp, err1 := parser.ParseExpr(expString)
	if err1 != nil {
		assert.Equal(t, expectedErrorMsg, err1.Error(), fmt.Sprintf("Unmatched: %v = %v: %s ", expectedErrorMsg, err1.Error(), expString))
		return
	}
	eCtx := NewPlainEvalCtxWithVars(AggFuncDisabled, &varValuesMap)
	_, err2 := eCtx.Eval(exp)

	assert.Equal(t, expectedErrorMsg, err2.Error(), fmt.Sprintf("Unmatched: %v = %v: %s ", expectedErrorMsg, err2.Error(), expString))
}

func TestBad(t *testing.T) {
	// Missing identifier
	assertEvalError(t, "some(", "1:6: expected ')', found 'EOF'", VarValuesMap{})

	// Missing identifier
	assertEvalError(t, "someident", "cannot evaluate identifier someident", VarValuesMap{})
	assertEvalError(t, "somefunc()", "cannot evaluate unsupported func 'somefunc'", VarValuesMap{})
	assertEvalError(t, "t2.aaa == 1", "cannot evaluate expression 't2', variable not supplied, check table/alias name", VarValuesMap{})

	// Unsupported binary operators
	assertEvalError(t, "2 ^ 1", "cannot perform binary expression unknown op ^", VarValuesMap{})   // TODO: implement ^ xor
	assertEvalError(t, "2 << 1", "cannot perform binary expression unknown op <<", VarValuesMap{}) // TODO: implement >> and <<
	assertEvalError(t, "1 &^ 2", "cannot perform binary expression unknown op &^", VarValuesMap{}) // No plans to support this op

	// Unsupported unary operators
	assertEvalError(t, "&1", "cannot evaluate unary op &, unkown op", VarValuesMap{})

	// Unsupported selector expr
	assertEvalError(t, "t1.fieldInt.w", "cannot evaluate selector expression &{t1 fieldInt}, unknown type of X: *ast.SelectorExpr", VarValuesMap{"t1": {"fieldInt": 1}})
}

func TestConvertEval(t *testing.T) {
	varValuesMap := VarValuesMap{
		"t1": {
			"fieldInt":      1,
			"fieldInt16":    int16(1),
			"fieldInt32":    int32(1),
			"fieldInt64":    int16(1),
			"fieldFloat32":  float32(1.0),
			"fieldFloat64":  float64(1.0),
			"fieldDecimal2": decimal.NewFromInt(1),
		},
	}

	// Number to number
	for fldName := range varValuesMap["t1"] {
		assertEqual(t, fmt.Sprintf("decimal2(t1.%s) == 1", fldName), true, varValuesMap)
		assertEqual(t, fmt.Sprintf("float(t1.%s) == 1.0", fldName), true, varValuesMap)
		assertEqual(t, fmt.Sprintf("int(t1.%s) == 1", fldName), true, varValuesMap)
	}

	// String to number
	assertEqual(t, `int("1") == 1`, true, varValuesMap)
	assertEqual(t, `float("1.0") == 1.0`, true, varValuesMap)
	assertEqual(t, `decimal2("1.0") == 1.0`, true, varValuesMap)

	// Number to string
	assertEqual(t, `string(1) == "1"`, true, varValuesMap)
	assertEqual(t, `string(1.1) == "1.1"`, true, varValuesMap)
	assertEqual(t, `string(decimal2(1.1)) == "1.1"`, true, varValuesMap)
}

func TestArithmetic(t *testing.T) {
	varValuesMap := VarValuesMap{
		"t1": {
			"fieldInt":      1,
			"fieldInt16":    int16(1),
			"fieldInt32":    int32(1),
			"fieldInt64":    int16(1),
			"fieldFloat32":  float32(1.0),
			"fieldFloat64":  float64(1.0),
			"fieldDecimal2": decimal.NewFromInt(1),
		},
		"t2": {
			"fieldInt":      2,
			"fieldInt16":    int16(2),
			"fieldInt32":    int32(2),
			"fieldInt64":    int16(2),
			"fieldFloat32":  float32(2.0),
			"fieldFloat64":  float64(2.0),
			"fieldDecimal2": decimal.NewFromInt(2),
		},
	}
	for k1 := range varValuesMap["t1"] {
		for k2 := range varValuesMap["t2"] {
			assertEqual(t, fmt.Sprintf("t1.%s + t2.%s == 3", k1, k2), true, varValuesMap)
			assertEqual(t, fmt.Sprintf("t1.%s - t2.%s == -1", k1, k2), true, varValuesMap)
			assertEqual(t, fmt.Sprintf("t1.%s * t2.%s == 2", k1, k2), true, varValuesMap)
			assertEqual(t, fmt.Sprintf("t2.%s / t1.%s == 2", k1, k2), true, varValuesMap)
		}
	}

	// Integer div
	assertEqual(t, "t1.fieldInt / t2.fieldInt == 0", true, varValuesMap)
	assertEqual(t, "t1.fieldInt % t2.fieldInt == 1", true, varValuesMap)

	// Float div
	assertEqual(t, "t1.fieldInt / t2.fieldFloat32 == 0.5", true, varValuesMap)
	assertEqual(t, "t1.fieldInt / t2.fieldDecimal2 == 0.5", true, varValuesMap)
	assertEqual(t, "t1.fieldInt / t2.fieldFloat32 == 0.5", true, varValuesMap)
	assertEqual(t, "t1.fieldInt / t2.fieldDecimal2 == 0.5", true, varValuesMap)
	assertEqual(t, "t1.fieldDecimal2 / t2.fieldInt == 0.5", true, varValuesMap)
	assertEqual(t, "t1.fieldInt / t2.fieldDecimal2 == 0.5", true, varValuesMap)

	// Div by zero
	assertEvalError(t, "t1.fieldInt / 0", "runtime error: integer divide by zero", varValuesMap)
	assertEqual(t, "t1.fieldFloat32 / 0", math.Inf(1), varValuesMap)
	assertEvalError(t, "t1.fieldDecimal2 / 0", "decimal division by 0", varValuesMap)

	// Bad types
	assertEvalError(t, "t1.fieldDecimal2 / `a`", "cannot perform binary arithmetic op, incompatible arg types '1(decimal.Decimal)' / 'a(string)' ", varValuesMap)
	assertEvalError(t, "-`a`", "cannot evaluate unary minus expression '-a(string)', unsupported type", varValuesMap)

	// String
	varValuesMap = VarValuesMap{
		"t1": {
			"field1": "aaa",
			"field2": `c"cc`,
		},
	}
	assertEqual(t, `t1.field1+t1.field2+"d"`, `aaac"ccd`, varValuesMap)

}

func TestCompare(t *testing.T) {
	varValuesMap := VarValuesMap{
		"t1": {
			"fieldInt":      1,
			"fieldInt16":    int16(1),
			"fieldInt32":    int32(1),
			"fieldInt64":    int16(1),
			"fieldFloat32":  float32(1.0),
			"fieldFloat64":  float64(1.0),
			"fieldDecimal2": decimal.NewFromInt(1),
		},
		"t2": {
			"fieldInt":      2,
			"fieldInt16":    int16(2),
			"fieldInt32":    int32(2),
			"fieldInt64":    int16(2),
			"fieldFloat32":  float32(2.0),
			"fieldFloat64":  float64(2.0),
			"fieldDecimal2": decimal.NewFromInt(2),
		},
	}
	for k1 := range varValuesMap["t1"] {
		for k2 := range varValuesMap["t2"] {
			assertEqual(t, fmt.Sprintf("t1.%s == t2.%s", k1, k2), false, varValuesMap)
			assertEqual(t, fmt.Sprintf("t1.%s != t2.%s", k1, k2), true, varValuesMap)
			assertEqual(t, fmt.Sprintf("t1.%s < t2.%s", k1, k2), true, varValuesMap)
			assertEqual(t, fmt.Sprintf("t1.%s <= t2.%s", k1, k2), true, varValuesMap)
			assertEqual(t, fmt.Sprintf("t2.%s > t1.%s", k1, k2), true, varValuesMap)
			assertEqual(t, fmt.Sprintf("t2.%s >= t1.%s", k1, k2), true, varValuesMap)
		}
	}

	// Bool
	assertEqual(t, "false == false", true, varValuesMap)
	assertEqual(t, "false == !true", true, varValuesMap)
	assertEqual(t, "false == true", false, varValuesMap)

	// String
	assertEqual(t, `"aaa" != "b"`, true, varValuesMap)
	assertEqual(t, `"aaa" == "b"`, false, varValuesMap)
	assertEqual(t, `"aaa" < "b"`, true, varValuesMap)
	assertEqual(t, `"aaa" <= "b"`, true, varValuesMap)
	assertEqual(t, `"aaa" > "b"`, false, varValuesMap)
	assertEqual(t, `"aaa" >= "b"`, false, varValuesMap)
	assertEqual(t, `"aaa" == "aaa"`, true, varValuesMap)
	assertEqual(t, `"aaa" != "aaa"`, false, varValuesMap)

	assertEvalError(t, "1 > true", "cannot perform binary comp op, incompatible arg types '1(int64)' > 'true(bool)' ", varValuesMap)
}

func TestBool(t *testing.T) {
	varValuesMap := VarValuesMap{}
	assertEqual(t, `true && true`, true, varValuesMap)
	assertEqual(t, `true && false`, false, varValuesMap)
	assertEqual(t, `true || true`, true, varValuesMap)
	assertEqual(t, `true || false`, true, varValuesMap)
	assertEqual(t, `!false`, true, varValuesMap)
	assertEqual(t, `!true`, false, varValuesMap)

	assertEvalError(t, `!123`, "cannot evaluate unary bool not expression with int64 on the right", varValuesMap)
	assertEvalError(t, "true || 1", "cannot evaluate binary bool expression 'true(bool) || 1(int64)', invalid right arg", varValuesMap)
	assertEvalError(t, "1 || true", "cannot perform binary op || against int64 left", varValuesMap)
}

func TestUnaryMinus(t *testing.T) {
	varValuesMap := VarValuesMap{
		"t1": {
			"fieldInt":      1,
			"fieldInt16":    int16(1),
			"fieldInt32":    int32(1),
			"fieldInt64":    int16(1),
			"fieldFloat32":  float32(1.0),
			"fieldFloat64":  float64(1.0),
			"fieldDecimal2": decimal.NewFromInt(1),
		},
	}
	for k := range varValuesMap["t1"] {
		assertEqual(t, fmt.Sprintf("-t1.%s == -1", k), true, varValuesMap)
	}
}

func TestTime(t *testing.T) {
	varValuesMap := VarValuesMap{
		"t1": {
			"fTime": time.Date(1, 1, 1, 1, 1, 1, 1, time.UTC),
		},
		"t2": {
			"fTime": time.Date(1, 1, 1, 1, 1, 1, 2, time.UTC),
		},
	}
	assertEqual(t, `t1.fTime < t2.fTime`, true, varValuesMap)
	assertEqual(t, `t1.fTime <= t2.fTime`, true, varValuesMap)
	assertEqual(t, `t2.fTime >= t1.fTime`, true, varValuesMap)
	assertEqual(t, `t2.fTime > t1.fTime`, true, varValuesMap)
	assertEqual(t, `t1.fTime == t2.fTime`, false, varValuesMap)
	assertEqual(t, `t1.fTime != t2.fTime`, true, varValuesMap)
}

func TestNewPlainEvalCtxAndInitializedAgg(t *testing.T) {
	varValuesMap := getTestValuesMap()
	varValuesMap["t1"]["fieldStr"] = "a"

	exp, _ := parser.ParseExpr(`string_agg(t1.fieldStr,",")`)
	aggEnabledType, aggFuncType, aggFuncArgs := DetectRootAggFunc(exp)
	eCtx, err := NewPlainEvalCtxAndInitializedAgg(aggEnabledType, aggFuncType, aggFuncArgs)
	assert.Equal(t, AggTypeString, eCtx.AggType)
	assert.Nil(t, err)

	exp, _ = parser.ParseExpr(`string_agg(t1.fieldStr,1)`)
	aggEnabledType, aggFuncType, aggFuncArgs = DetectRootAggFunc(exp)
	_, err = NewPlainEvalCtxAndInitializedAgg(aggEnabledType, aggFuncType, aggFuncArgs)
	assert.Equal(t, "string_agg second parameter must be a constant string", err.Error())

	exp, _ = parser.ParseExpr(`string_agg(t1.fieldStr, a)`)
	aggEnabledType, aggFuncType, aggFuncArgs = DetectRootAggFunc(exp)
	_, err = NewPlainEvalCtxAndInitializedAgg(aggEnabledType, aggFuncType, aggFuncArgs)
	assert.Equal(t, "string_agg second parameter must be a basic literal", err.Error())
}

type EvalFunc int

const (
	BinaryIntFunc = iota
	BinaryIntToBoolFunc
	BinaryFloat64Func
	BinaryFloat64ToBoolFunc
	BinaryDecimal2Func
	BinaryDecimal2ToBoolFunc
	BinaryTimeToBoolFunc
	BinaryBoolFunc
	BinaryBoolToBoolFunc
	BinaryStringFunc
	BinaryStringToBoolFunc
)

func assertBinaryEval(t *testing.T, evalFunc EvalFunc, valLeftVolatile interface{}, op token.Token, valRightVolatile interface{}, errorMessage string) {
	var err error
	eCtx := NewPlainEvalCtx(AggFuncDisabled)
	switch evalFunc {
	case BinaryIntFunc:
		_, err = eCtx.EvalBinaryInt(valLeftVolatile, op, valRightVolatile)
	case BinaryIntToBoolFunc:
		_, err = eCtx.EvalBinaryIntToBool(valLeftVolatile, op, valRightVolatile)
	case BinaryFloat64Func:
		_, err = eCtx.EvalBinaryFloat64(valLeftVolatile, op, valRightVolatile)
	case BinaryFloat64ToBoolFunc:
		_, err = eCtx.EvalBinaryFloat64ToBool(valLeftVolatile, op, valRightVolatile)
	case BinaryDecimal2Func:
		_, err = eCtx.EvalBinaryDecimal2(valLeftVolatile, op, valRightVolatile)
	case BinaryDecimal2ToBoolFunc:
		_, err = eCtx.EvalBinaryDecimal2ToBool(valLeftVolatile, op, valRightVolatile)
	case BinaryTimeToBoolFunc:
		_, err = eCtx.EvalBinaryTimeToBool(valLeftVolatile, op, valRightVolatile)
	case BinaryBoolFunc:
		_, err = eCtx.EvalBinaryBool(valLeftVolatile, op, valRightVolatile)
	case BinaryBoolToBoolFunc:
		_, err = eCtx.EvalBinaryBoolToBool(valLeftVolatile, op, valRightVolatile)
	case BinaryStringFunc:
		_, err = eCtx.EvalBinaryString(valLeftVolatile, op, valRightVolatile)
	case BinaryStringToBoolFunc:
		_, err = eCtx.EvalBinaryStringToBool(valLeftVolatile, op, valRightVolatile)
	default:
		assert.Fail(t, "unsupported EvalFunc")
	}
	assert.Equal(t, errorMessage, err.Error())
}

func TestBadEvalBinaryInt(t *testing.T) {
	goodVal := int64(1)
	badVal := "a"
	assertBinaryEval(t, BinaryIntFunc, badVal, token.ADD, goodVal, "cannot evaluate binary int64 expression '+' with 'a(string)' on the left")
	assertBinaryEval(t, BinaryIntFunc, goodVal, token.ADD, badVal, "cannot evaluate binary int64 expression '1(int64) + a(string)', invalid right arg")
	assertBinaryEval(t, BinaryIntFunc, goodVal, token.AND, goodVal, "cannot perform int op & against int 1 and int 1")
}

func TestBadEvalBinaryIntToBool(t *testing.T) {
	goodVal := int64(1)
	badVal := "a"
	assertBinaryEval(t, BinaryIntToBoolFunc, badVal, token.LSS, goodVal, "cannot evaluate binary int64 expression '<' with 'a(string)' on the left")
	assertBinaryEval(t, BinaryIntToBoolFunc, goodVal, token.LSS, badVal, "cannot evaluate binary int64 expression '1(int64) < a(string)', invalid right arg")
	assertBinaryEval(t, BinaryIntToBoolFunc, goodVal, token.ADD, int64(1), "cannot perform bool op + against int 1 and int 1")
}

func TestBadEvalBinaryFloat64(t *testing.T) {
	goodVal := float64(1)
	badVal := "a"
	assertBinaryEval(t, BinaryFloat64Func, badVal, token.ADD, goodVal, "cannot evaluate binary float64 expression '+' with 'a(string)' on the left")
	assertBinaryEval(t, BinaryFloat64Func, goodVal, token.ADD, badVal, "cannot evaluate binary float expression '1(float64) + a(string)', invalid right arg")
	assertBinaryEval(t, BinaryFloat64Func, goodVal, token.AND, goodVal, "cannot perform float64 op & against float64 1.000000 and float64 1.000000")
}

func TestBadEvalBinaryFloat64ToBool(t *testing.T) {
	goodVal := float64(1)
	badVal := "a"
	assertBinaryEval(t, BinaryFloat64ToBoolFunc, badVal, token.LSS, goodVal, "cannot evaluate binary foat64 expression '<' with 'a(string)' on the left")
	assertBinaryEval(t, BinaryFloat64ToBoolFunc, goodVal, token.LSS, badVal, "cannot evaluate binary float64 expression '1(float64) < a(string)', invalid right arg")
	assertBinaryEval(t, BinaryFloat64ToBoolFunc, goodVal, token.ADD, goodVal, "cannot perform bool op + against float 1.000000 and float 1.000000")
}

func TestBadEvalBinaryDecimal2(t *testing.T) {
	goodVal := decimal.NewFromFloat(1)
	badVal := "a"
	assertBinaryEval(t, BinaryDecimal2Func, badVal, token.ADD, goodVal, "cannot evaluate binary decimal2 expression '+' with 'a(string)' on the left")
	assertBinaryEval(t, BinaryDecimal2Func, goodVal, token.ADD, badVal, "cannot evaluate binary decimal2 expression '1(decimal.Decimal) + a(string)', invalid right arg")
	assertBinaryEval(t, BinaryDecimal2Func, goodVal, token.AND, goodVal, "cannot perform decimal2 op & against decimal2 1 and float64 1")
}

func TestBadEvalBinaryDecimal2Bool(t *testing.T) {
	goodVal := decimal.NewFromFloat(1)
	badVal := "a"
	assertBinaryEval(t, BinaryDecimal2ToBoolFunc, badVal, token.LSS, goodVal, "cannot evaluate binary decimal2 expression '<' with 'a(string)' on the left")
	assertBinaryEval(t, BinaryDecimal2ToBoolFunc, goodVal, token.LSS, badVal, "cannot evaluate binary decimal2 expression '1(decimal.Decimal) < a(string)', invalid right arg")
	assertBinaryEval(t, BinaryDecimal2ToBoolFunc, goodVal, token.ADD, goodVal, "cannot perform bool op + against decimal2 1 and decimal2 1")
}

func TestBadEvalBinaryTimeBool(t *testing.T) {
	goodVal := time.Date(2000, 1, 1, 0, 0, 0, 0, time.FixedZone("", -7200))
	badVal := "a"
	assertBinaryEval(t, BinaryTimeToBoolFunc, badVal, token.LSS, goodVal, "cannot evaluate binary time expression '<' with 'a(string)' on the left")
	assertBinaryEval(t, BinaryTimeToBoolFunc, goodVal, token.LSS, badVal, "cannot evaluate binary time expression '2000-01-01 00:00:00 -0200 -0200(time.Time) < a(string)', invalid right arg")
	assertBinaryEval(t, BinaryTimeToBoolFunc, goodVal, token.ADD, goodVal, "cannot perform bool op + against time 2000-01-01 00:00:00 -0200 -0200 and time 2000-01-01 00:00:00 -0200 -0200")
}

func TestBadEvalBinaryBool(t *testing.T) {
	goodVal := true
	badVal := "a"
	assertBinaryEval(t, BinaryBoolFunc, badVal, token.LAND, goodVal, "cannot evaluate binary bool expression '&&' with 'a(string)' on the left")
	assertBinaryEval(t, BinaryBoolFunc, goodVal, token.LOR, badVal, "cannot evaluate binary bool expression 'true(bool) || a(string)', invalid right arg")
	assertBinaryEval(t, BinaryBoolFunc, goodVal, token.ADD, goodVal, "cannot perform bool op + against bool true and bool true")
}

func TestBadEvalBinaryBoolToBool(t *testing.T) {
	goodVal := true
	badVal := "a"
	assertBinaryEval(t, BinaryBoolToBoolFunc, badVal, token.EQL, goodVal, "cannot evaluate binary bool expression == with string on the left")
	assertBinaryEval(t, BinaryBoolToBoolFunc, goodVal, token.NEQ, badVal, "cannot evaluate binary bool expression 'true(bool) != a(string)', invalid right arg")
	assertBinaryEval(t, BinaryBoolToBoolFunc, goodVal, token.ADD, goodVal, "cannot evaluate binary bool expression, op + not supported (and will never be)")
}

func TestBadEvalBinaryString(t *testing.T) {
	goodVal := "good"
	badVal := 1
	assertBinaryEval(t, BinaryStringFunc, badVal, token.ADD, goodVal, "cannot evaluate binary string expression + with int on the left")
	assertBinaryEval(t, BinaryStringFunc, goodVal, token.ADD, badVal, "cannot evaluate binary string expression 'good(string) + 1(int)', invalid right arg")
	assertBinaryEval(t, BinaryStringFunc, goodVal, token.AND, goodVal, "cannot perform string op & against string 'good' and string 'good', op not supported")
}

func TestBadEvalBinaryStringToBool(t *testing.T) {
	goodVal := "good"
	badVal := 1
	assertBinaryEval(t, BinaryStringToBoolFunc, badVal, token.LSS, goodVal, "cannot evaluate binary string expression < with '1(int)' on the left")
	assertBinaryEval(t, BinaryStringToBoolFunc, goodVal, token.GTR, badVal, "cannot evaluate binary decimal2 expression 'good(string) > 1(int)', invalid right arg")
	assertBinaryEval(t, BinaryStringToBoolFunc, goodVal, token.AND, goodVal, "cannot perform bool op & against string good and string good")
}
