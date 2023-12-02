package eval

import (
	"fmt"
	"go/ast"
	"go/parser"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func getTestValuesMap() VarValuesMap {
	return VarValuesMap{
		"t1": {
			"fieldInt":   1,
			"fieldFloat": 2.1,
			"fieldDec":   decimal.NewFromInt(1),
			"fieldStr":   "a",
		},
	}
}

func TestMissingCtxVars(t *testing.T) {
	varValuesMap := getTestValuesMap()

	var err error
	var exp ast.Expr
	var eCtx EvalCtx

	exp, _ = parser.ParseExpr("avg(t1.fieldInt)")
	eCtx = NewPlainEvalCtx(AggFuncEnabled)
	_, err = eCtx.Eval(exp)
	assert.Contains(t, err.Error(), "no variables supplied to the context")

	delete(varValuesMap["t1"], "fieldInt")
	exp, _ = parser.ParseExpr("avg(t1.fieldInt)")
	eCtx = NewPlainEvalCtxWithVars(AggFuncEnabled, &varValuesMap)
	_, err = eCtx.Eval(exp)
	assert.Contains(t, err.Error(), "variable not supplied")
}

func TestExtraAgg(t *testing.T) {
	varValuesMap := getTestValuesMap()

	var err error
	var exp ast.Expr
	var eCtx EvalCtx

	// Extra sum
	exp, _ = parser.ParseExpr("sum(min(t1.fieldFloat))")
	eCtx = NewPlainEvalCtxWithVars(AggFuncEnabled, &varValuesMap)
	_, err = eCtx.Eval(exp)
	assert.Equal(t, err.Error(), "cannot evaluate more than one aggregate functions in the expression, extra sum() found besides min()")

	// Extra avg
	exp, _ = parser.ParseExpr("avg(min(t1.fieldFloat))")
	eCtx = NewPlainEvalCtxWithVars(AggFuncEnabled, &varValuesMap)
	_, err = eCtx.Eval(exp)
	assert.Equal(t, err.Error(), "cannot evaluate more than one aggregate functions in the expression, extra avg() found besides min()")

	// Extra min
	exp, _ = parser.ParseExpr("min(min(t1.fieldFloat))")
	eCtx = NewPlainEvalCtxWithVars(AggFuncEnabled, &varValuesMap)
	_, err = eCtx.Eval(exp)
	assert.Equal(t, err.Error(), "cannot evaluate more than one aggregate functions in the expression, extra min() found besides min()")

	// Extra max
	exp, _ = parser.ParseExpr("max(min(t1.fieldFloat))")
	eCtx = NewPlainEvalCtxWithVars(AggFuncEnabled, &varValuesMap)
	_, err = eCtx.Eval(exp)
	assert.Equal(t, err.Error(), "cannot evaluate more than one aggregate functions in the expression, extra max() found besides min()")

	// Extra count
	exp, _ = parser.ParseExpr("min(t1.fieldFloat)+count())")
	eCtx = NewPlainEvalCtxWithVars(AggFuncEnabled, &varValuesMap)
	_, err = eCtx.Eval(exp)
	assert.Equal(t, err.Error(), "cannot evaluate more than one aggregate functions in the expression, extra count() found besides min()")
}

func assertFuncTypeAndArgs(t *testing.T, expression string, aggFuncEnabled AggEnabledType, expectedAggFuncType AggFuncType, expectedNumberOfArgs int) {
	exp, _ := parser.ParseExpr(expression)
	aggEnabledType, aggFuncType, aggFuncArgs := DetectRootAggFunc(exp)
	assert.Equal(t, aggFuncEnabled, aggEnabledType, "expected AggFuncEnabled for "+expression)
	assert.Equal(t, expectedAggFuncType, aggFuncType, fmt.Sprintf("expected %s for %s", expectedAggFuncType, expression))
	assert.Equal(t, expectedNumberOfArgs, len(aggFuncArgs), fmt.Sprintf("expected %d args for %s", expectedNumberOfArgs, expression))
}

func TestDetectRootArgFunc(t *testing.T) {
	assertFuncTypeAndArgs(t, `string_agg(t1.fieldStr,",")`, AggFuncEnabled, AggStringAgg, 2)
	assertFuncTypeAndArgs(t, `sum(t1.fieldFloat)`, AggFuncEnabled, AggSum, 1)
	assertFuncTypeAndArgs(t, `avg(t1.fieldFloat)`, AggFuncEnabled, AggAvg, 1)
	assertFuncTypeAndArgs(t, `min(t1.fieldFloat)`, AggFuncEnabled, AggMin, 1)
	assertFuncTypeAndArgs(t, `max(t1.fieldFloat)`, AggFuncEnabled, AggMax, 1)
	assertFuncTypeAndArgs(t, `count()`, AggFuncEnabled, AggCount, 0)
	assertFuncTypeAndArgs(t, `some_func(t1.fieldFloat)`, AggFuncDisabled, AggUnknown, 0)
}

func TestStringAgg(t *testing.T) {
	varValuesMap := getTestValuesMap()

	var exp ast.Expr
	var result any

	varValuesMap["t1"]["fieldStr"] = "a"

	exp, _ = parser.ParseExpr(`string_agg(t1.fieldStr,"-")`)
	eCtx, _ := NewPlainEvalCtxWithVarsAndInitializedAgg(AggFuncEnabled, &varValuesMap, AggStringAgg, exp.(*ast.CallExpr).Args)
	result, _ = eCtx.Eval(exp)
	assert.Equal(t, "a", result)
	varValuesMap["t1"]["fieldStr"] = "b"
	result, _ = eCtx.Eval(exp)
	assert.Equal(t, "a-b", result)

	// Empty str
	exp, _ = parser.ParseExpr(`string_agg(t1.fieldStr,",")`)
	eCtx, _ = NewPlainEvalCtxWithVarsAndInitializedAgg(AggFuncEnabled, &varValuesMap, AggStringAgg, exp.(*ast.CallExpr).Args)
	assert.Equal(t, "", eCtx.StringAgg.Sb.String())

	var err error

	// Bad number of args
	exp, _ = parser.ParseExpr(`string_agg(t1.fieldStr)`)
	_, err = NewPlainEvalCtxWithVarsAndInitializedAgg(AggFuncEnabled, &varValuesMap, AggStringAgg, exp.(*ast.CallExpr).Args)
	assert.Contains(t, err.Error(), "string_agg must have two parameters")

	// Bad separators
	exp, _ = parser.ParseExpr(`string_agg(t1.fieldStr, t2.someBadField)`)
	_, err = NewPlainEvalCtxWithVarsAndInitializedAgg(AggFuncEnabled, &varValuesMap, AggStringAgg, exp.(*ast.CallExpr).Args)
	assert.Contains(t, err.Error(), "string_agg second parameter must be a basic literal")

	exp, _ = parser.ParseExpr(`string_agg(t1.fieldStr, 123)`)
	_, err = NewPlainEvalCtxWithVarsAndInitializedAgg(AggFuncEnabled, &varValuesMap, AggStringAgg, exp.(*ast.CallExpr).Args)
	assert.Contains(t, err.Error(), "string_agg second parameter must be a constant string")

	// Bad data type
	exp, _ = parser.ParseExpr(`string_agg(t1.fieldFloat, ",")`)
	eCtx, err = NewPlainEvalCtxWithVarsAndInitializedAgg(AggFuncEnabled, &varValuesMap, AggStringAgg, exp.(*ast.CallExpr).Args)
	assert.Nil(t, err)
	// TODO: can we check expression type before Eval?
	_, err = eCtx.Eval(exp)
	assert.Contains(t, err.Error(), "unsupported type float64")

	// Bad ctx with disabled agg func calling string_agg()
	exp, _ = parser.ParseExpr(`string_agg(t1.fieldStr,"-")`)
	badCtx := NewPlainEvalCtxWithVars(AggFuncDisabled, &varValuesMap)
	_, err = badCtx.Eval(exp)
	assert.Contains(t, err.Error(), "context aggregate not enabled")
}

func TestSum(t *testing.T) {
	varValuesMap := getTestValuesMap()

	var exp ast.Expr
	var eCtx EvalCtx
	var result any
	var err error

	// Sum float
	exp, _ = parser.ParseExpr("5 + sum(t1.fieldFloat)")
	eCtx = NewPlainEvalCtxWithVars(AggFuncEnabled, &varValuesMap)
	varValuesMap["t1"]["fieldFloat"] = 2.1
	result, _ = eCtx.Eval(exp)
	assert.Equal(t, 5+2.1, result)
	result, _ = eCtx.Eval(exp)
	assert.Equal(t, 5+4.2, result)

	// float -> dec
	varValuesMap["t1"]["fieldFloat"] = decimal.NewFromInt(1)
	_, err = eCtx.Eval(exp)
	assert.Equal(t, "cannot evaluate sum(), it started with type float, now got decimal value 1", err.Error())

	// Sum int
	exp, _ = parser.ParseExpr("5 + sum(t1.fieldInt)")
	eCtx = NewPlainEvalCtxWithVars(AggFuncEnabled, &varValuesMap)
	varValuesMap["t1"]["fieldInt"] = 1
	result, _ = eCtx.Eval(exp)
	assert.Equal(t, int64(6), result)
	result, _ = eCtx.Eval(exp)
	assert.Equal(t, int64(7), result)

	// int -> float
	varValuesMap["t1"]["fieldInt"] = float64(1)
	_, err = eCtx.Eval(exp)
	assert.Equal(t, "cannot evaluate sum(), it started with type int, now got float value 1.000000", err.Error())

	// Sum dec
	exp, _ = parser.ParseExpr("5 + sum(t1.fieldDec)")
	eCtx = NewPlainEvalCtxWithVars(AggFuncEnabled, &varValuesMap)
	varValuesMap["t1"]["fieldDec"] = decimal.NewFromInt(1)
	result, _ = eCtx.Eval(exp)
	assert.Equal(t, decimal.New(600, -2), result)
	result, _ = eCtx.Eval(exp)
	assert.Equal(t, decimal.New(700, -2), result)

	// dec -> int
	varValuesMap["t1"]["fieldDec"] = int64(1)
	_, err = eCtx.Eval(exp)
	assert.Equal(t, "cannot evaluate sum(), it started with type decimal, now got int value 1", err.Error())

	eCtx = NewPlainEvalCtx(AggFuncEnabled)
	// Sum int empty
	assert.Equal(t, int64(0), eCtx.Sum.Int)
	// Sum float empty
	assert.Equal(t, float64(0), eCtx.Sum.Float)
	// Sum dec empty
	assert.Equal(t, defaultDecimal(), eCtx.Sum.Dec)
}

func TestAvg(t *testing.T) {
	varValuesMap := getTestValuesMap()

	var exp ast.Expr
	var eCtx EvalCtx
	var result any
	var err error

	// Avg int
	exp, _ = parser.ParseExpr("avg(t1.fieldInt)")
	eCtx = NewPlainEvalCtxWithVars(AggFuncEnabled, &varValuesMap)

	varValuesMap["t1"]["fieldInt"] = 1
	_, err = eCtx.Eval(exp)
	assert.Nil(t, err)
	_, err = eCtx.Eval(exp)
	assert.Nil(t, err)
	varValuesMap["t1"]["fieldInt"] = 2
	result, _ = eCtx.Eval(exp)
	assert.Equal(t, int64(1), result)

	// int -> float
	varValuesMap["t1"]["fieldInt"] = float64(1)
	_, err = eCtx.Eval(exp)
	assert.Equal(t, "cannot evaluate avg(), it started with type int, now got float value 1.000000", err.Error())

	// Avg float
	exp, _ = parser.ParseExpr("avg(t1.fieldFloat)")
	eCtx = NewPlainEvalCtxWithVars(AggFuncEnabled, &varValuesMap)
	varValuesMap["t1"]["fieldFloat"] = float64(1)
	_, err = eCtx.Eval(exp)
	assert.Nil(t, err)
	_, err = eCtx.Eval(exp)
	assert.Nil(t, err)
	varValuesMap["t1"]["fieldFloat"] = float64(2)
	result, _ = eCtx.Eval(exp)
	assert.Equal(t, float64(1.3333333333333333), result)

	// float -> dec
	varValuesMap["t1"]["fieldFloat"] = decimal.NewFromInt(1)
	_, err = eCtx.Eval(exp)
	assert.Equal(t, "cannot evaluate avg(), it started with type float, now got decimal value 1", err.Error())

	// Avg dec
	exp, _ = parser.ParseExpr("avg(t1.fieldDec)")
	eCtx = NewPlainEvalCtxWithVars(AggFuncEnabled, &varValuesMap)
	varValuesMap["t1"]["fieldDec"] = decimal.NewFromInt(1)
	_, err = eCtx.Eval(exp)
	assert.Nil(t, err)
	_, err = eCtx.Eval(exp)
	assert.Nil(t, err)
	varValuesMap["t1"]["fieldDec"] = decimal.NewFromInt(2)
	result, _ = eCtx.Eval(exp)
	assert.Equal(t, decimal.NewFromFloat32(1.33), result)

	// dec -> int
	varValuesMap["t1"]["fieldDec"] = int64(1)
	_, err = eCtx.Eval(exp)
	assert.Equal(t, "cannot evaluate avg(), it started with type decimal, now got int value 1", err.Error())

	eCtx = NewPlainEvalCtx(AggFuncEnabled)

	// Avg int empty
	assert.Equal(t, int64(0), eCtx.Avg.Int)
	// Avg float empty
	assert.Equal(t, float64(0), eCtx.Avg.Float)
	// Avg dec empty
	assert.Equal(t, defaultDecimal(), eCtx.Avg.Dec)
}

func TestMin(t *testing.T) {
	varValuesMap := getTestValuesMap()

	var exp ast.Expr
	var eCtx EvalCtx
	var result any
	var err error

	// Min float
	eCtx = NewPlainEvalCtxWithVars(AggFuncEnabled, &varValuesMap)
	varValuesMap["t1"]["fieldFloat"] = 1.0
	exp, _ = parser.ParseExpr("min(t1.fieldFloat)")
	result, _ = eCtx.Eval(exp)
	assert.Equal(t, 1.0, result)
	varValuesMap["t1"]["fieldFloat"] = 2.0
	result, _ = eCtx.Eval(exp)
	assert.Equal(t, 1.0, result)

	// float -> dec
	varValuesMap["t1"]["fieldFloat"] = decimal.NewFromInt(1)
	_, err = eCtx.Eval(exp)
	assert.Equal(t, "cannot evaluate min(), it started with type float, now got decimal value 1", err.Error())

	// float -> string
	varValuesMap["t1"]["fieldFloat"] = "a"
	_, err = eCtx.Eval(exp)
	assert.Equal(t, "cannot evaluate min(), it started with type float, now got string value a", err.Error())

	// Min int
	eCtx = NewPlainEvalCtxWithVars(AggFuncEnabled, &varValuesMap)
	varValuesMap["t1"]["fieldInt"] = 1
	exp, _ = parser.ParseExpr("min(t1.fieldInt)")
	result, _ = eCtx.Eval(exp)
	assert.Equal(t, int64(1), result)
	varValuesMap["t1"]["fieldInt"] = 2
	result, _ = eCtx.Eval(exp)
	assert.Equal(t, int64(1), result)

	// int -> float
	varValuesMap["t1"]["fieldInt"] = float64(1)
	_, err = eCtx.Eval(exp)
	assert.Equal(t, "cannot evaluate min(), it started with type int, now got float value 1.000000", err.Error())

	// Min dec
	eCtx = NewPlainEvalCtxWithVars(AggFuncEnabled, &varValuesMap)
	varValuesMap["t1"]["fieldDec"] = decimal.NewFromInt(1)
	exp, _ = parser.ParseExpr("min(t1.fieldDec)")
	result, _ = eCtx.Eval(exp)
	assert.Equal(t, decimal.NewFromInt(1), result)
	varValuesMap["t1"]["fieldDec"] = decimal.NewFromInt(2)
	result, _ = eCtx.Eval(exp)
	assert.Equal(t, decimal.NewFromInt(1), result)

	// dec -> int
	varValuesMap["t1"]["fieldDec"] = int64(1)
	_, err = eCtx.Eval(exp)
	assert.Equal(t, "cannot evaluate min(), it started with type decimal, now got int value 1", err.Error())

	// Min str
	eCtx = NewPlainEvalCtxWithVars(AggFuncEnabled, &varValuesMap)
	varValuesMap["t1"]["fieldStr"] = "a"
	exp, _ = parser.ParseExpr("min(t1.fieldStr)")
	result, _ = eCtx.Eval(exp)
	assert.Equal(t, "a", result)
	varValuesMap["t1"]["fieldStr"] = "b"
	result, _ = eCtx.Eval(exp)
	assert.Equal(t, "a", result)

	eCtx = NewPlainEvalCtx(AggFuncEnabled)
	// Empty int
	assert.Equal(t, maxSupportedInt, eCtx.Min.Int)
	// Empty float
	assert.Equal(t, maxSupportedFloat, eCtx.Min.Float)
	// Empty dec
	assert.Equal(t, maxSupportedDecimal(), eCtx.Min.Dec)
	// Empty str
	assert.Equal(t, "", eCtx.Min.Str)
}

func TestMax(t *testing.T) {
	varValuesMap := getTestValuesMap()

	var exp ast.Expr
	var eCtx EvalCtx
	var result any
	var err error

	// Max float
	eCtx = NewPlainEvalCtxWithVars(AggFuncEnabled, &varValuesMap)
	varValuesMap["t1"]["fieldFloat"] = 10.0
	exp, _ = parser.ParseExpr("max(t1.fieldFloat)")
	result, _ = eCtx.Eval(exp)
	assert.Equal(t, 10.0, result)
	varValuesMap["t1"]["fieldFloat"] = 2.0
	result, _ = eCtx.Eval(exp)
	assert.Equal(t, 10.0, result)

	// float -> dec
	varValuesMap["t1"]["fieldFloat"] = decimal.NewFromInt(1)
	_, err = eCtx.Eval(exp)
	assert.Equal(t, "cannot evaluate max(), it started with type float, now got decimal value 1", err.Error())

	// float -> string
	varValuesMap["t1"]["fieldFloat"] = "a"
	_, err = eCtx.Eval(exp)
	assert.Equal(t, "cannot evaluate max(), it started with type float, now got string value a", err.Error())

	// Max int
	eCtx = NewPlainEvalCtxWithVars(AggFuncEnabled, &varValuesMap)
	varValuesMap["t1"]["fieldInt"] = 1
	exp, _ = parser.ParseExpr("max(t1.fieldInt)")
	result, _ = eCtx.Eval(exp)
	assert.Equal(t, int64(1), result)
	varValuesMap["t1"]["fieldInt"] = 2
	result, _ = eCtx.Eval(exp)
	assert.Equal(t, int64(2), result)

	// int -> float
	varValuesMap["t1"]["fieldInt"] = float64(1)
	_, err = eCtx.Eval(exp)
	assert.Equal(t, "cannot evaluate max(), it started with type int, now got float value 1.000000", err.Error())

	// Max dec
	eCtx = NewPlainEvalCtxWithVars(AggFuncEnabled, &varValuesMap)
	varValuesMap["t1"]["fieldDec"] = decimal.NewFromInt(1)
	exp, _ = parser.ParseExpr("max(t1.fieldDec)")
	result, _ = eCtx.Eval(exp)
	assert.Equal(t, decimal.NewFromInt(1), result)
	varValuesMap["t1"]["fieldDec"] = decimal.NewFromInt(2)
	result, _ = eCtx.Eval(exp)
	assert.Equal(t, decimal.NewFromInt(2), result)

	// dec -> int
	varValuesMap["t1"]["fieldDec"] = int64(1)
	_, err = eCtx.Eval(exp)
	assert.Equal(t, "cannot evaluate max(), it started with type decimal, now got int value 1", err.Error())

	// Max str
	eCtx = NewPlainEvalCtxWithVars(AggFuncEnabled, &varValuesMap)
	varValuesMap["t1"]["fieldStr"] = "a"
	exp, _ = parser.ParseExpr("max(t1.fieldStr)")
	result, _ = eCtx.Eval(exp)
	assert.Equal(t, "a", result)
	varValuesMap["t1"]["fieldStr"] = "b"
	result, _ = eCtx.Eval(exp)
	assert.Equal(t, "b", result)

	eCtx = NewPlainEvalCtx(AggFuncEnabled)
	// Empty int
	assert.Equal(t, minSupportedInt, eCtx.Max.Int)
	// Empty float
	assert.Equal(t, minSupportedFloat, eCtx.Max.Float)
	// Empty dec
	assert.Equal(t, minSupportedDecimal(), eCtx.Max.Dec)
	// Empty str
	assert.Equal(t, "", eCtx.Max.Str)
}

func TestCount(t *testing.T) {

	varValuesMap := getTestValuesMap()

	var exp ast.Expr
	var eCtx EvalCtx
	var result any

	eCtx = NewPlainEvalCtxWithVars(AggFuncEnabled, &varValuesMap)
	exp, _ = parser.ParseExpr("count()")
	result, _ = eCtx.Eval(exp)
	assert.Equal(t, int64(1), result)
	result, _ = eCtx.Eval(exp)
	assert.Equal(t, int64(2), result)

	// Empty
	eCtx = NewPlainEvalCtx(AggFuncEnabled)
	assert.Equal(t, int64(0), eCtx.Count)
}

func TestNoVars(t *testing.T) {

	var exp ast.Expr
	var eCtx EvalCtx
	var result any

	eCtx = NewPlainEvalCtx(AggFuncEnabled)
	exp, _ = parser.ParseExpr("sum(5)")
	result, _ = eCtx.Eval(exp)
	assert.Equal(t, int64(5), result)
	result, _ = eCtx.Eval(exp)
	assert.Equal(t, int64(5+5), result)
}
