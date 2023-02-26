package eval

import (
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

func TestDetectRootArgFunc(t *testing.T) {
	varValuesMap := getTestValuesMap()

	var exp ast.Expr
	varValuesMap["t1"]["fieldStr"] = "a"

	exp, _ = parser.ParseExpr(`string_agg(t1.fieldStr,",")`)
	aggEnabledType, aggFuncType, aggFuncArgs := DetectRootAggFunc(exp)
	assert.Equal(t, AggFuncEnabled, aggEnabledType)
	assert.Equal(t, AggStringAgg, aggFuncType)
	assert.Equal(t, 2, len(aggFuncArgs))

	exp, _ = parser.ParseExpr(`sum(t1.fieldFloat)`)
	aggEnabledType, aggFuncType, aggFuncArgs = DetectRootAggFunc(exp)
	assert.Equal(t, AggFuncEnabled, aggEnabledType)
	assert.Equal(t, AggSum, aggFuncType)
	assert.Equal(t, 1, len(aggFuncArgs))

	exp, _ = parser.ParseExpr(`some_func(t1.fieldFloat)`)
	aggEnabledType, aggFuncType, aggFuncArgs = DetectRootAggFunc(exp)
	assert.Equal(t, AggFuncDisabled, aggEnabledType)
	assert.Equal(t, AggUnknown, aggFuncType)
	assert.Equal(t, 0, len(aggFuncArgs))
}
func TestStringAgg(t *testing.T) {
	varValuesMap := getTestValuesMap()

	var exp ast.Expr
	var result interface{}

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
	eCtx, err = NewPlainEvalCtxWithVarsAndInitializedAgg(AggFuncEnabled, &varValuesMap, AggStringAgg, exp.(*ast.CallExpr).Args)
	assert.Contains(t, err.Error(), "agg_string must have two parameters")

	// Bad separators
	exp, _ = parser.ParseExpr(`string_agg(t1.fieldStr, t2.someBadField)`)
	eCtx, err = NewPlainEvalCtxWithVarsAndInitializedAgg(AggFuncEnabled, &varValuesMap, AggStringAgg, exp.(*ast.CallExpr).Args)
	assert.Contains(t, err.Error(), "agg_string second parameter must be a basic literal")

	exp, _ = parser.ParseExpr(`string_agg(t1.fieldStr, 123)`)
	eCtx, err = NewPlainEvalCtxWithVarsAndInitializedAgg(AggFuncEnabled, &varValuesMap, AggStringAgg, exp.(*ast.CallExpr).Args)
	assert.Contains(t, err.Error(), "agg_string second parameter must be a constant string")

	// Bad data type
	exp, _ = parser.ParseExpr(`string_agg(t1.fieldFloat, ",")`)
	eCtx, err = NewPlainEvalCtxWithVarsAndInitializedAgg(AggFuncEnabled, &varValuesMap, AggStringAgg, exp.(*ast.CallExpr).Args)
	// TODO: can we check expression type before Eval?
	_, err = eCtx.Eval(exp)
	assert.Contains(t, err.Error(), "unsupported type float64")
}

func TestSum(t *testing.T) {
	varValuesMap := getTestValuesMap()

	var exp ast.Expr
	var eCtx EvalCtx
	var result interface{}

	// Sum float
	exp, _ = parser.ParseExpr("5 + sum(t1.fieldFloat)")
	eCtx = NewPlainEvalCtxWithVars(AggFuncEnabled, &varValuesMap)
	varValuesMap["t1"]["fieldFloat"] = 2.1
	result, _ = eCtx.Eval(exp)
	assert.Equal(t, 5+2.1, result)
	result, _ = eCtx.Eval(exp)
	assert.Equal(t, 5+4.2, result)

	// Sum int
	exp, _ = parser.ParseExpr("5 + sum(t1.fieldInt)")
	eCtx = NewPlainEvalCtxWithVars(AggFuncEnabled, &varValuesMap)
	varValuesMap["t1"]["fieldInt"] = 1
	result, _ = eCtx.Eval(exp)
	assert.Equal(t, int64(6), result)
	result, _ = eCtx.Eval(exp)
	assert.Equal(t, int64(7), result)

	// Sum dec
	exp, _ = parser.ParseExpr("5 + sum(t1.fieldDec)")
	eCtx = NewPlainEvalCtxWithVars(AggFuncEnabled, &varValuesMap)
	varValuesMap["t1"]["fieldDec"] = decimal.NewFromInt(1)
	result, _ = eCtx.Eval(exp)
	assert.Equal(t, decimal.New(600, -2), result)
	result, _ = eCtx.Eval(exp)
	assert.Equal(t, decimal.New(700, -2), result)

	// Sum int empty
	exp, _ = parser.ParseExpr("sum(t1.fieldInt)")
	eCtx = NewPlainEvalCtx(AggFuncEnabled)
	assert.Equal(t, int64(0), eCtx.Sum.Int)

	// Sum float empty
	exp, _ = parser.ParseExpr("sum(t1.fieldFloat)")
	eCtx = NewPlainEvalCtx(AggFuncEnabled)
	assert.Equal(t, float64(0), eCtx.Sum.Float)

	// Sum dec empty
	exp, _ = parser.ParseExpr("sum(t1.fieldDec)")
	eCtx = NewPlainEvalCtx(AggFuncEnabled)
	assert.Equal(t, defaultDecimal(), eCtx.Sum.Dec)
}

func TestAvg(t *testing.T) {
	varValuesMap := getTestValuesMap()

	var exp ast.Expr
	var eCtx EvalCtx
	var result interface{}

	// Avg int
	exp, _ = parser.ParseExpr("avg(t1.fieldInt)")
	eCtx = NewPlainEvalCtxWithVars(AggFuncEnabled, &varValuesMap)

	varValuesMap["t1"]["fieldInt"] = 1
	eCtx.Eval(exp)
	eCtx.Eval(exp)
	varValuesMap["t1"]["fieldInt"] = 2
	result, _ = eCtx.Eval(exp)
	assert.Equal(t, int64(1), result)

	// Avg float
	exp, _ = parser.ParseExpr("avg(t1.fieldFloat)")
	eCtx = NewPlainEvalCtxWithVars(AggFuncEnabled, &varValuesMap)
	varValuesMap["t1"]["fieldFloat"] = float64(1)
	eCtx.Eval(exp)
	eCtx.Eval(exp)
	varValuesMap["t1"]["fieldFloat"] = float64(2)
	result, _ = eCtx.Eval(exp)
	assert.Equal(t, float64(1.3333333333333333), result)

	// Avg dec
	exp, _ = parser.ParseExpr("avg(t1.fieldDec)")
	eCtx = NewPlainEvalCtxWithVars(AggFuncEnabled, &varValuesMap)
	varValuesMap["t1"]["fieldDec"] = decimal.NewFromInt(1)
	eCtx.Eval(exp)
	eCtx.Eval(exp)
	varValuesMap["t1"]["fieldDec"] = decimal.NewFromInt(2)
	result, _ = eCtx.Eval(exp)
	assert.Equal(t, decimal.NewFromFloat32(1.33), result)

	// Avg int empty
	exp, _ = parser.ParseExpr("avg(t1.fieldInt)")
	eCtx = NewPlainEvalCtx(AggFuncEnabled)
	assert.Equal(t, int64(0), eCtx.Avg.Int)

	// Avg float empty
	exp, _ = parser.ParseExpr("avg(t1.fieldFloat)")
	eCtx = NewPlainEvalCtx(AggFuncEnabled)
	assert.Equal(t, float64(0), eCtx.Avg.Float)

	// Avg dec empty
	exp, _ = parser.ParseExpr("avg(t1.fieldDec)")
	eCtx = NewPlainEvalCtx(AggFuncEnabled)
	assert.Equal(t, defaultDecimal(), eCtx.Avg.Dec)
}

func TestMin(t *testing.T) {
	varValuesMap := getTestValuesMap()

	var exp ast.Expr
	var eCtx EvalCtx
	var result interface{}

	// Min float
	eCtx = NewPlainEvalCtxWithVars(AggFuncEnabled, &varValuesMap)
	varValuesMap["t1"]["fieldFloat"] = 1.0
	exp, _ = parser.ParseExpr("min(t1.fieldFloat)")
	result, _ = eCtx.Eval(exp)
	assert.Equal(t, 1.0, result)
	varValuesMap["t1"]["fieldFloat"] = 2.0
	result, _ = eCtx.Eval(exp)
	assert.Equal(t, 1.0, result)

	// Min int
	eCtx = NewPlainEvalCtxWithVars(AggFuncEnabled, &varValuesMap)
	varValuesMap["t1"]["fieldInt"] = 1
	exp, _ = parser.ParseExpr("min(t1.fieldInt)")
	result, _ = eCtx.Eval(exp)
	assert.Equal(t, int64(1), result)
	varValuesMap["t1"]["fieldInt"] = 2
	result, _ = eCtx.Eval(exp)
	assert.Equal(t, int64(1), result)

	// Min dec
	eCtx = NewPlainEvalCtxWithVars(AggFuncEnabled, &varValuesMap)
	varValuesMap["t1"]["fieldDec"] = decimal.NewFromInt(1)
	exp, _ = parser.ParseExpr("min(t1.fieldDec)")
	result, _ = eCtx.Eval(exp)
	assert.Equal(t, decimal.NewFromInt(1), result)
	varValuesMap["t1"]["fieldDec"] = decimal.NewFromInt(2)
	result, _ = eCtx.Eval(exp)
	assert.Equal(t, decimal.NewFromInt(1), result)

	// Min str
	eCtx = NewPlainEvalCtxWithVars(AggFuncEnabled, &varValuesMap)
	varValuesMap["t1"]["fieldStr"] = "a"
	exp, _ = parser.ParseExpr("min(t1.fieldStr)")
	result, _ = eCtx.Eval(exp)
	assert.Equal(t, "a", result)
	varValuesMap["t1"]["fieldStr"] = "b"
	result, _ = eCtx.Eval(exp)
	assert.Equal(t, "a", result)

	// Empty int
	exp, _ = parser.ParseExpr("min(t1.fieldInt)")
	eCtx = NewPlainEvalCtx(AggFuncEnabled)
	assert.Equal(t, maxSupportedInt, eCtx.Min.Int)

	// Empty float
	exp, _ = parser.ParseExpr("min(t1.fieldFloat)")
	eCtx = NewPlainEvalCtx(AggFuncEnabled)
	assert.Equal(t, maxSupportedFloat, eCtx.Min.Float)

	// Empty dec
	exp, _ = parser.ParseExpr("min(t1.fieldDec)")
	eCtx = NewPlainEvalCtx(AggFuncEnabled)
	assert.Equal(t, maxSupportedDecimal(), eCtx.Min.Dec)

	// Empty str
	exp, _ = parser.ParseExpr("min(t1.fieldString)")
	eCtx = NewPlainEvalCtx(AggFuncEnabled)
	assert.Equal(t, "", eCtx.Min.Str)
}

func TestMax(t *testing.T) {
	varValuesMap := getTestValuesMap()

	var exp ast.Expr
	var eCtx EvalCtx
	var result interface{}

	// Max float
	eCtx = NewPlainEvalCtxWithVars(AggFuncEnabled, &varValuesMap)
	varValuesMap["t1"]["fieldFloat"] = 10.0
	exp, _ = parser.ParseExpr("max(t1.fieldFloat)")
	result, _ = eCtx.Eval(exp)
	assert.Equal(t, 10.0, result)
	varValuesMap["t1"]["fieldFloat"] = 2.0
	result, _ = eCtx.Eval(exp)
	assert.Equal(t, 10.0, result)

	// Max int
	eCtx = NewPlainEvalCtxWithVars(AggFuncEnabled, &varValuesMap)
	varValuesMap["t1"]["fieldInt"] = 1
	exp, _ = parser.ParseExpr("max(t1.fieldInt)")
	result, _ = eCtx.Eval(exp)
	assert.Equal(t, int64(1), result)
	varValuesMap["t1"]["fieldInt"] = 2
	result, _ = eCtx.Eval(exp)
	assert.Equal(t, int64(2), result)

	// Max dec
	eCtx = NewPlainEvalCtxWithVars(AggFuncEnabled, &varValuesMap)
	varValuesMap["t1"]["fieldDec"] = decimal.NewFromInt(1)
	exp, _ = parser.ParseExpr("max(t1.fieldDec)")
	result, _ = eCtx.Eval(exp)
	assert.Equal(t, decimal.NewFromInt(1), result)
	varValuesMap["t1"]["fieldDec"] = decimal.NewFromInt(2)
	result, _ = eCtx.Eval(exp)
	assert.Equal(t, decimal.NewFromInt(2), result)

	// Max str
	eCtx = NewPlainEvalCtxWithVars(AggFuncEnabled, &varValuesMap)
	varValuesMap["t1"]["fieldStr"] = "a"
	exp, _ = parser.ParseExpr("max(t1.fieldStr)")
	result, _ = eCtx.Eval(exp)
	assert.Equal(t, "a", result)
	varValuesMap["t1"]["fieldStr"] = "b"
	result, _ = eCtx.Eval(exp)
	assert.Equal(t, "b", result)

	// Empty int
	exp, _ = parser.ParseExpr("max(t1.fieldInt)")
	eCtx = NewPlainEvalCtx(AggFuncEnabled)
	assert.Equal(t, minSupportedInt, eCtx.Max.Int)

	// Empty float
	exp, _ = parser.ParseExpr("max(t1.fieldFloat)")
	eCtx = NewPlainEvalCtx(AggFuncEnabled)
	assert.Equal(t, minSupportedFloat, eCtx.Max.Float)

	// Empty dec
	exp, _ = parser.ParseExpr("max(t1.fieldDec)")
	eCtx = NewPlainEvalCtx(AggFuncEnabled)
	assert.Equal(t, minSupportedDecimal(), eCtx.Max.Dec)

	// Empty str
	exp, _ = parser.ParseExpr("max(t1.fieldString)")
	eCtx = NewPlainEvalCtx(AggFuncEnabled)
	assert.Equal(t, "", eCtx.Max.Str)
}

func TestCount(t *testing.T) {

	varValuesMap := getTestValuesMap()

	var exp ast.Expr
	var eCtx EvalCtx
	var result interface{}

	eCtx = NewPlainEvalCtxWithVars(AggFuncEnabled, &varValuesMap)
	exp, _ = parser.ParseExpr("count()")
	result, _ = eCtx.Eval(exp)
	assert.Equal(t, int64(1), result)
	result, _ = eCtx.Eval(exp)
	assert.Equal(t, int64(2), result)

	// Empty
	exp, _ = parser.ParseExpr("count()")
	eCtx = NewPlainEvalCtx(AggFuncEnabled)
	assert.Equal(t, int64(0), eCtx.Count)
}

func TestNoVars(t *testing.T) {

	var exp ast.Expr
	var eCtx EvalCtx
	var result interface{}

	eCtx = NewPlainEvalCtx(AggFuncEnabled)
	exp, _ = parser.ParseExpr("sum(5)")
	result, _ = eCtx.Eval(exp)
	assert.Equal(t, int64(5), result)
	result, _ = eCtx.Eval(exp)
	assert.Equal(t, int64(5+5), result)
}
