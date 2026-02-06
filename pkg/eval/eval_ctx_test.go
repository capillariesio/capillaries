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

func TestVarValuesMapUtils(t *testing.T) {
	varValuesMap := VarValuesMap{"some_table": map[string]any{"some_field": 1}}
	assert.Equal(t, "[some_table ]", varValuesMap.Tables())
	assert.Equal(t, "[some_table.some_field ]", varValuesMap.Names())
}

func TestBad(t *testing.T) {
	// Missing identifier
	assertEvalError(t, "some(", "1:6: expected ')', found 'EOF'", VarValuesMap{})

	// Missing identifier
	assertEvalError(t, "someident", "cannot evaluate ident expression 'someident', no empty object", VarValuesMap{})
	assertEvalError(t, "somefunc()", "cannot evaluate unsupported func 'somefunc'", VarValuesMap{})
	assertEvalError(t, "t2.aaa == 1", "cannot evaluate selector ident expression 't2', variable not supplied, check table/alias name", VarValuesMap{})

	// Unsupported binary operators
	assertEvalError(t, "2 ^ 1", "cannot perform binary expression unknown op ^", VarValuesMap{})   // TODO: implement ^ xor
	assertEvalError(t, "2 << 1", "cannot perform binary expression unknown op <<", VarValuesMap{}) // TODO: implement >> and <<
	assertEvalError(t, "1 &^ 2", "cannot perform binary expression unknown op &^", VarValuesMap{}) // No plans to support this op

	// Unsupported unary operators
	assertEvalError(t, "&1", "cannot evaluate unary op &, unknown op", VarValuesMap{})

	// Unsupported selector expr
	assertEvalError(t, "t1.fieldInt.w", "cannot evaluate selector expression &{t1 fieldInt}, unknown type of X: *ast.SelectorExpr", VarValuesMap{"t1": {"fieldInt": 1}})
}

func TestIdent(t *testing.T) {
	varValuesMap := VarValuesMap{
		"": {
			"fieldInt": 2,
		},
	}
	assertEqual(t, `(fieldInt == 2) == true`, true, varValuesMap)
	assertEqual(t, `(fieldInt == 3) == false`, true, varValuesMap)
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

	// Parenthesis
	assertEqual(t, "(t1.fieldInt + 1)/ t2.fieldDecimal2 == 1.0", true, varValuesMap)

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
		"": {
			"nil": nil,
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
		assertEqual(t, fmt.Sprintf("t1.%s == nil", k1), false, varValuesMap)
		assertEqual(t, fmt.Sprintf("t1.%s != nil", k1), true, varValuesMap)
		assertEqual(t, fmt.Sprintf("t1.%s < nil", k1), false, varValuesMap)
		assertEqual(t, fmt.Sprintf("t1.%s <= nil", k1), false, varValuesMap)
		assertEqual(t, fmt.Sprintf("t2.%s > nil", k1), false, varValuesMap)
		assertEqual(t, fmt.Sprintf("t2.%s >= nil", k1), false, varValuesMap)
	}
	assertEqual(t, "nil == nil", true, varValuesMap)
	assertEqual(t, "nil != nil", false, varValuesMap)
	assertEqual(t, "nil < nil", false, varValuesMap)
	assertEqual(t, "nil <= nil", false, varValuesMap)
	assertEqual(t, "nil > nil", false, varValuesMap)
	assertEqual(t, "nil >= nil", false, varValuesMap)

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

func TestFunc(t *testing.T) {
	functions := map[string]EvalFunction{
		"package1.Mul2": func(args []any) (any, error) {
			if len(args) != 1 {
				return nil, fmt.Errorf("package1.Mul2 srequires one arg, not %d", len(args))
			}
			switch typedArg := args[0].(type) {
			case float64:
				return typedArg * 2.0, nil
			default:
				return nil, fmt.Errorf("package1.Mul2 does not support type %T (%v)", args[0], args[0])
			}
		},
		"mul3": func(args []any) (any, error) {
			if len(args) != 1 {
				return nil, fmt.Errorf("mul2 requires one arg, not %d", len(args))
			}
			switch typedArg := args[0].(type) {
			case float64:
				return typedArg * 3.0, nil
			default:
				return nil, fmt.Errorf("mul3 does not support type %T (%v)", args[0], args[0])
			}
		},
	}
	eCtx := NewPlainEvalCtx(functions, nil, nil)

	exp, err := parser.ParseExpr("package1.Mul2(1.0)")
	assert.Nil(t, err)
	result, err := eCtx.Eval(exp)
	assert.Equal(t, float64(2.0), result)

	exp, err = parser.ParseExpr("mul3(1.0)")
	assert.Nil(t, err)
	result, err = eCtx.Eval(exp)
	assert.Equal(t, float64(3.0), result)
}

func TestConst(t *testing.T) {
	constants := map[string]any{
		"const1":          float64(1.0),
		"package2.const2": float64(2.0),
	}
	eCtx := NewPlainEvalCtx(nil, constants, nil)

	exp, err := parser.ParseExpr("const1*2")
	assert.Nil(t, err)
	result, err := eCtx.Eval(exp)
	assert.Equal(t, float64(2.0), result)

	exp, err = parser.ParseExpr("package2.const2*2")
	assert.Nil(t, err)
	result, err = eCtx.Eval(exp)
	assert.Equal(t, float64(4.0), result)
}

func TestNewPlainEvalCtxAndInitializedAgg(t *testing.T) {
	varValuesMap := getTestValuesMap()
	varValuesMap["t1"]["fieldStr"] = "a"

	exp, _ := parser.ParseExpr(`string_agg(t1.fieldStr,",")`)
	aggEnabledType, aggFuncType, aggFuncArgs := DetectRootAggFunc(exp)
	assert.Equal(t, AggFuncEnabled, aggEnabledType)
	eCtx, err := NewAggEvalCtx(aggFuncType, aggFuncArgs, nil, nil, nil)
	assert.Equal(t, AggTypeString, eCtx.aggType)
	assert.Nil(t, err)

	exp, _ = parser.ParseExpr(`string_agg(t1.fieldStr,1)`)
	aggEnabledType, aggFuncType, aggFuncArgs = DetectRootAggFunc(exp)
	assert.Equal(t, AggFuncEnabled, aggEnabledType)
	_, err = NewAggEvalCtx(aggFuncType, aggFuncArgs, nil, nil, nil)
	assert.Equal(t, "string_agg/if second parameter must be a constant string", err.Error())

	exp, _ = parser.ParseExpr(`string_agg(t1.fieldStr, a)`)
	aggEnabledType, aggFuncType, aggFuncArgs = DetectRootAggFunc(exp)
	assert.Equal(t, AggFuncEnabled, aggEnabledType)
	_, err = NewAggEvalCtx(aggFuncType, aggFuncArgs, nil, nil, nil)
	assert.Equal(t, "string_agg/if second parameter must be a basic literal", err.Error())
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

func assertBinaryEval(t *testing.T, evalFunc EvalFunc, valLeftVolatile any, op token.Token, valRightVolatile any, errorMessage string) {
	var err error
	eCtx := newPlainEvalCtx(AggFuncDisabled)
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
		_, err = eCtx.EvalBinaryDecimal(valLeftVolatile, op, valRightVolatile)
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
	assertBinaryEval(t, BinaryDecimal2Func, badVal, token.ADD, goodVal, "cannot evaluate binary decimal expression '+' with 'a(string)' on the left")
	assertBinaryEval(t, BinaryDecimal2Func, goodVal, token.ADD, badVal, "cannot evaluate binary decimal expression '1(decimal.Decimal) + a(string)', invalid right arg")
	assertBinaryEval(t, BinaryDecimal2Func, goodVal, token.AND, goodVal, "cannot perform decimal op & against decimal 1 and float64 1")
}

func TestBadEvalBinaryDecimal2Bool(t *testing.T) {
	goodVal := decimal.NewFromFloat(1)
	badVal := "a"
	assertBinaryEval(t, BinaryDecimal2ToBoolFunc, badVal, token.LSS, goodVal, "cannot evaluate binary decimal expression '<' with 'a(string)' on the left")
	assertBinaryEval(t, BinaryDecimal2ToBoolFunc, goodVal, token.LSS, badVal, "cannot evaluate binary decimal expression '1(decimal.Decimal) < a(string)', invalid right arg")
	assertBinaryEval(t, BinaryDecimal2ToBoolFunc, goodVal, token.ADD, goodVal, "cannot perform bool op + against decimal 1 and decimal 1")
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
	assertBinaryEval(t, BinaryStringToBoolFunc, goodVal, token.GTR, badVal, "cannot evaluate binary decimal expression 'good(string) > 1(int)', invalid right arg")
	assertBinaryEval(t, BinaryStringToBoolFunc, goodVal, token.AND, goodVal, "cannot perform bool op & against string good and string good")
}

func TestUnsupported(t *testing.T) {
	varValuesMap := VarValuesMap{
		"t1": {
			"fTime": time.Date(1, 1, 1, 1, 1, 1, 1, time.UTC),
		},
		"t2": {
			"fTime": time.Date(1, 1, 1, 1, 1, 1, 2, time.UTC),
		},
	}
	assertEvalError(t, `t1.fTime[i] < 2`, "unsupported type *ast.IndexExpr", varValuesMap)
}

func TestGetSafeValue(t *testing.T) {

	eCtx := NewPlainEvalCtx(nil, nil, nil)
	assert.Equal(t, nil, eCtx.GetValue())
	assert.Equal(t, int64(35), eCtx.GetSafeValue(int64(35)))

	constants := map[string]any{
		"const1": float64(1.0),
	}
	eCtx = NewPlainEvalCtx(nil, constants, nil)
	exp, err := parser.ParseExpr("const1*2")
	assert.Nil(t, err)
	eCtx.Eval(exp)
	assert.Equal(t, float64(2.0), eCtx.GetValue())
	assert.Equal(t, float64(2.0), eCtx.GetSafeValue(int64(35)))
}
