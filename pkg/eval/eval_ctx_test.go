package eval

import (
	"fmt"
	"go/parser"
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
	for k, _ := range varValuesMap["t1"] {
		assertEqual(t, fmt.Sprintf("decimal2(t1.%s) == 1", k), true, varValuesMap)
		assertEqual(t, fmt.Sprintf("float(t1.%s) == 1.0", k), true, varValuesMap)
		assertEqual(t, fmt.Sprintf("int(t1.%s) == 1", k), true, varValuesMap)
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
	for k1, _ := range varValuesMap["t1"] {
		for k2, _ := range varValuesMap["t2"] {
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
	for k1, _ := range varValuesMap["t1"] {
		for k2, _ := range varValuesMap["t2"] {
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
	for k, _ := range varValuesMap["t1"] {
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
	assert.Equal(t, nil, err)
}
