package eval

import (
	"fmt"
	"go/ast"
	"go/parser"
	"testing"
	"time"

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
	var eCtx *EvalCtx

	exp, _ = parser.ParseExpr("avg(t1.fieldInt)")
	eCtx = newPlainEvalCtx(AggFuncEnabled)
	_, err = eCtx.Eval(exp)
	assert.Contains(t, err.Error(), "no variables supplied to the context")

	delete(varValuesMap["t1"], "fieldInt")
	exp, _ = parser.ParseExpr("avg(t1.fieldInt)")
	aggFuncEnabled, aggFuncType, aggFuncArgs := DetectRootAggFunc(exp)
	assert.Equal(t, aggFuncEnabled, aggFuncEnabled)
	eCtx, err = NewAggEvalCtx(aggFuncType, aggFuncArgs, nil, nil, varValuesMap)
	_, err = eCtx.Eval(exp)
	assert.Contains(t, err.Error(), "variable not supplied")
}

func validateExtraAgg(expression string) string {
	varValuesMap := getTestValuesMap()
	exp, _ := parser.ParseExpr(expression)
	_, aggFuncType, aggFuncArgs := DetectRootAggFunc(exp)
	constants := map[string]any{"true": true, "false": false}
	eCtx, err := NewAggEvalCtx(aggFuncType, aggFuncArgs, nil, constants, varValuesMap)
	_, err = eCtx.Eval(exp)
	return err.Error()
}

func TestExtraAgg(t *testing.T) {
	assert.Contains(t, validateExtraAgg("sum(min(t1.fieldInt))"), "extra sum() found besides already used min()")
	assert.Contains(t, validateExtraAgg("avg(min_if(t1.fieldInt,true))"), "extra avg() found besides already used min()")
	assert.Contains(t, validateExtraAgg("min(min(t1.fieldInt))"), "extra min() found besides already used min()")
	assert.Contains(t, validateExtraAgg("max_if(min(t1.fieldInt))"), "extra max_if() found besides already used min()")
	assert.Contains(t, validateExtraAgg("min(t1.fieldFloat)+count())"), "extra count() found besides already used min()")
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

func validateAggTwoStringValues(aggFuncType AggFuncType, expression string, v1 any, v2 any) (any, any) {
	varValuesMap := getTestValuesMap()
	exp, _ := parser.ParseExpr(expression)
	eCtx, _ := NewAggEvalCtx(aggFuncType, exp.(*ast.CallExpr).Args, nil, nil, varValuesMap)
	varValuesMap["t1"]["fieldStr"] = v1
	result1, _ := eCtx.Eval(exp)
	varValuesMap["t1"]["fieldStr"] = v2
	result2, _ := eCtx.Eval(exp)
	return result1, result2
}

func TestStringAgg(t *testing.T) {
	var r1, r2 any

	r1, r2 = validateAggTwoStringValues(AggStringAggIf, `string_agg_if(t1.fieldStr,"-",t1.fieldStr == "b")`, "a", "b")
	assert.Equal(t, "", r1)
	assert.Equal(t, "b", r2)

	r1, r2 = validateAggTwoStringValues(AggStringAgg, `string_agg(t1.fieldStr,"-")`, "a", "b")
	assert.Equal(t, "a", r1)
	assert.Equal(t, "a-b", r2)
}

func TestStringAggEdgeCases(t *testing.T) {

	varValuesMap := getTestValuesMap()
	var exp ast.Expr

	varValuesMap["t1"]["fieldStr"] = "a"

	// Empty str
	exp, _ = parser.ParseExpr(`string_agg(t1.fieldStr,",")`)
	eCtx, _ := NewAggEvalCtx(AggStringAgg, exp.(*ast.CallExpr).Args, nil, nil, varValuesMap)
	assert.Equal(t, "", eCtx.stringAggCollector.Sb.String())

	var err error

	// Bad number of args
	exp, _ = parser.ParseExpr(`string_agg(t1.fieldStr)`)
	_, err = NewAggEvalCtx(AggStringAgg, exp.(*ast.CallExpr).Args, nil, nil, varValuesMap)
	assert.Contains(t, err.Error(), "string_agg must have two parameters")

	exp, _ = parser.ParseExpr(`string_agg_if(t1.fieldStr)`)
	_, err = NewAggEvalCtx(AggStringAggIf, exp.(*ast.CallExpr).Args, nil, nil, varValuesMap)
	assert.Contains(t, err.Error(), "string_agg_if must have three parameters")

	// Bad separators
	exp, _ = parser.ParseExpr(`string_agg(t1.fieldStr, t2.someBadField)`)
	_, err = NewAggEvalCtx(AggStringAgg, exp.(*ast.CallExpr).Args, nil, nil, varValuesMap)
	assert.Contains(t, err.Error(), "string_agg/if second parameter must be a basic literal")

	exp, _ = parser.ParseExpr(`string_agg(t1.fieldStr, 123)`)
	_, err = NewAggEvalCtx(AggStringAgg, exp.(*ast.CallExpr).Args, nil, nil, varValuesMap)
	assert.Contains(t, err.Error(), "string_agg/if second parameter must be a constant string")
}

func validateAggTwoValues(expression string, v1 any, v2 any) (any, any) {
	varValuesMap := getTestValuesMap()
	exp, _ := parser.ParseExpr(expression)
	_, aggFuncType, aggFuncArgs := DetectRootAggFunc(exp)
	eCtx, _ := NewAggEvalCtx(aggFuncType, aggFuncArgs, nil, nil, varValuesMap)
	varValuesMap["t1"]["fieldInt"] = v1
	result1, _ := eCtx.Eval(exp)
	varValuesMap["t1"]["fieldInt"] = v2
	result2, _ := eCtx.Eval(exp)
	return result1, result2
}

func validateAggThreeDecValues(expression string, v1 decimal.Decimal, v2 decimal.Decimal, v3 decimal.Decimal) (any, any, any) {
	varValuesMap := getTestValuesMap()
	exp, _ := parser.ParseExpr(expression)
	_, aggFuncType, aggFuncArgs := DetectRootAggFunc(exp)
	eCtx, _ := NewAggEvalCtx(aggFuncType, aggFuncArgs, nil, nil, varValuesMap)
	varValuesMap["t1"]["fieldDec"] = v1
	result1, _ := eCtx.Eval(exp)
	varValuesMap["t1"]["fieldDec"] = v2
	result2, _ := eCtx.Eval(exp)
	varValuesMap["t1"]["fieldDec"] = v3
	result3, _ := eCtx.Eval(exp)
	return result1, result2, result3
}

func TestSum(t *testing.T) {
	var r1, r2 any

	r1, r2 = validateAggTwoValues("sum_if(t1.fieldInt, t1.fieldInt == 2)", 1, 2)
	assert.Equal(t, int64(0), r1)
	assert.Equal(t, int64(2), r2)

	r1, r2 = validateAggTwoValues("sum(t1.fieldInt)", 1, 2)
	assert.Equal(t, int64(1), r1)
	assert.Equal(t, int64(3), r2)

	r1, r2 = validateAggTwoValues("sum_if(t1.fieldInt, t1.fieldInt == 2.0)", 1.0, 2.0)
	assert.Equal(t, 0.0, r1)
	assert.Equal(t, 2.0, r2)

	r1, r2 = validateAggTwoValues("sum(t1.fieldInt)", 1.0, 2.0)
	assert.Equal(t, 1.0, r1)
	assert.Equal(t, 3.0, r2)

	d1 := decimal.NewFromInt(1)
	d2 := decimal.NewFromInt(2)

	r1, r2 = validateAggTwoValues("sum_if(t1.fieldInt, t1.fieldInt == 2)", d1, d2)
	assert.Equal(t, decimal.NewFromInt(0), r1)
	assert.Equal(t, decimal.NewFromInt(2), r2)

	r1, r2 = validateAggTwoValues("sum(t1.fieldInt)", d1, d2)
	assert.Equal(t, d1, r1)
	assert.Equal(t, decimal.NewFromFloat(3), r2)
}

func TestAvg(t *testing.T) {
	var r1, r2, r3 any

	r1, r2 = validateAggTwoValues("avg_if(t1.fieldInt, t1.fieldInt == 2)", 1, 2)
	assert.Equal(t, int64(0), r1)
	assert.Equal(t, int64(2), r2)

	r1, r2 = validateAggTwoValues("avg(t1.fieldInt)", 1, 2)
	assert.Equal(t, int64(1), r1)
	assert.Equal(t, int64(1), r2)

	r1, r2 = validateAggTwoValues("avg_if(t1.fieldInt, t1.fieldInt == 2.0)", 1.0, 2.0)
	assert.Equal(t, 0.0, r1)
	assert.Equal(t, 2.0, r2)

	r1, r2 = validateAggTwoValues("avg(t1.fieldInt)", 1.0, 2.0)
	assert.Equal(t, 1.0, r1)
	assert.Equal(t, 1.5, r2)

	// Test decimals (re-use fieldInt field name sometimes)
	d1 := decimal.NewFromInt(1)
	d2 := decimal.NewFromInt(2)
	d3 := decimal.NewFromInt(1)

	r1, r2 = validateAggTwoValues("avg_if(t1.fieldInt, t1.fieldInt == 2)", d1, d2)
	assert.True(t, decimal.NewFromInt(0).Equal(r1.(decimal.Decimal)))
	assert.True(t, decimal.NewFromFloat(2).Div(decimal.NewFromInt(1)).Round(2).Equal(r2.(decimal.Decimal)))

	r1, r2 = validateAggTwoValues("avg(t1.fieldInt)", d1, d2)
	assert.True(t, d1.Div(decimal.NewFromInt(1)).Round(2).Equal(r1.(decimal.Decimal)))
	assert.True(t, decimal.NewFromFloat(3).Div(decimal.NewFromInt(2)).Round(2).Equal(r2.(decimal.Decimal)))

	r1, r2, r3 = validateAggThreeDecValues("avg_if(t1.fieldDec, t1.fieldDec == 1)", d1, d2, d3)
	assert.True(t, decimal.NewFromInt(1).Equal(r1.(decimal.Decimal)))
	assert.True(t, decimal.NewFromFloat(2).Div(decimal.NewFromInt(2)).Equal(r2.(decimal.Decimal)))
	assert.True(t, decimal.NewFromFloat(2).Div(decimal.NewFromInt(2)).Equal(r3.(decimal.Decimal)))

	r1, r2, r3 = validateAggThreeDecValues("avg(t1.fieldDec)", d1, d2, d3)
	assert.True(t, d1.Div(decimal.NewFromInt(1)).Round(2).Equal(r1.(decimal.Decimal)))
	assert.True(t, decimal.NewFromFloat(3).Div(decimal.NewFromInt(2)).Equal(r2.(decimal.Decimal)))
	assert.True(t, decimal.NewFromFloat(4).Div(decimal.NewFromInt(3)).Equal(r3.(decimal.Decimal)))
}

func TestMin(t *testing.T) {
	var r1, r2 any

	r1, r2 = validateAggTwoValues("min_if(t1.fieldInt, t1.fieldInt == 2.0)", 1.0, 2.0)
	assert.Equal(t, maxSupportedFloat, r1)
	assert.Equal(t, 2.0, r2)

	r1, r2 = validateAggTwoValues("min(t1.fieldInt)", 1.0, 2.0)
	assert.Equal(t, 1.0, r1)
	assert.Equal(t, 1.0, r2)

	r1, r2 = validateAggTwoValues("min_if(t1.fieldInt, t1.fieldInt == 2.0)", 1, 2)
	assert.Equal(t, maxSupportedInt, r1)
	assert.Equal(t, int64(2), r2)

	r1, r2 = validateAggTwoValues("min(t1.fieldInt)", 1, 2)
	assert.Equal(t, int64(1), r1)
	assert.Equal(t, int64(1), r2)

	d1 := decimal.NewFromInt(1)
	d2 := decimal.NewFromInt(2)

	r1, r2 = validateAggTwoValues("min_if(t1.fieldInt, t1.fieldInt == 2)", d1, d2)
	assert.Equal(t, maxSupportedDecimal(), r1)
	assert.Equal(t, d2, r2)

	r1, r2 = validateAggTwoValues("min(t1.fieldInt)", d1, d2)
	assert.Equal(t, d1, r1)
	assert.Equal(t, d1, r2)

	r1, r2 = validateAggTwoValues(`min_if(t1.fieldInt, t1.fieldInt == "b")`, "a", "b")
	assert.Equal(t, "", r1)
	assert.Equal(t, "b", r2)

	r1, r2 = validateAggTwoValues(`min(t1.fieldInt)`, "a", "b")
	assert.Equal(t, "a", r1)
	assert.Equal(t, "a", r2)
}

func TestMax(t *testing.T) {
	var r1, r2 any

	r1, r2 = validateAggTwoValues("max_if(t1.fieldInt, t1.fieldInt == 2.0)", 10.0, 2.0)
	assert.Equal(t, minSupportedFloat, r1)
	assert.Equal(t, 2.0, r2)

	r1, r2 = validateAggTwoValues("max(t1.fieldInt)", 10.0, 2.0)
	assert.Equal(t, 10.0, r1)
	assert.Equal(t, 10.0, r2)

	r1, r2 = validateAggTwoValues("max_if(t1.fieldInt,t1.fieldInt == 1)", 2, 1)
	assert.Equal(t, minSupportedInt, r1)
	assert.Equal(t, int64(1), r2)

	r1, r2 = validateAggTwoValues("max(t1.fieldInt)", 1, 2)
	assert.Equal(t, int64(1), r1)
	assert.Equal(t, int64(2), r2)

	d1 := decimal.NewFromInt(1)
	d2 := decimal.NewFromInt(2)

	r1, r2 = validateAggTwoValues("max_if(t1.fieldInt, t1.fieldInt == 1)", d1, d2)
	assert.Equal(t, d1, r1)
	assert.Equal(t, d1, r2)

	r1, r2 = validateAggTwoValues("max(t1.fieldInt)", d1, d2)
	assert.Equal(t, d1, r1)
	assert.Equal(t, d2, r2)

	r1, r2 = validateAggTwoValues(`max_if(t1.fieldInt, t1.fieldInt == "a")`, "a", "b")
	assert.Equal(t, "a", r1)
	assert.Equal(t, "a", r2)

	r1, r2 = validateAggTwoValues(`max(t1.fieldInt)`, "a", "b")
	assert.Equal(t, "a", r1)
	assert.Equal(t, "b", r2)
}

func TestCount(t *testing.T) {

	varValuesMap := getTestValuesMap()

	var exp ast.Expr
	var eCtx *EvalCtx
	var result any

	// count_if

	exp, _ = parser.ParseExpr("count_if(t1.fieldInt == 2)")
	_, aggFuncType, aggFuncArgs := DetectRootAggFunc(exp)
	eCtx, _ = NewAggEvalCtx(aggFuncType, aggFuncArgs, nil, nil, varValuesMap)
	varValuesMap["t1"]["fieldInt"] = 1
	result, _ = eCtx.Eval(exp)
	assert.Equal(t, int64(0), result)
	varValuesMap["t1"]["fieldInt"] = 2
	result, _ = eCtx.Eval(exp)
	assert.Equal(t, int64(1), result)

	// count
	_, aggFuncType, aggFuncArgs = DetectRootAggFunc(exp)
	eCtx, _ = NewAggEvalCtx(aggFuncType, aggFuncArgs, nil, nil, varValuesMap)
	exp, _ = parser.ParseExpr("count()")
	result, _ = eCtx.Eval(exp)
	assert.Equal(t, int64(1), result)
	result, _ = eCtx.Eval(exp)
	assert.Equal(t, int64(2), result)
}

func TestNoAggRows(t *testing.T) {
	var exp ast.Expr
	var eCtx *EvalCtx
	var aggFuncType AggFuncType
	var aggFuncArgs []ast.Expr

	varValuesMap := getTestValuesMap()
	varValuesMap["t1"]["fieldInt"] = 0

	// if

	exp, _ = parser.ParseExpr("count_if(t1.fieldInt > 0)")
	_, aggFuncType, aggFuncArgs = DetectRootAggFunc(exp)
	eCtx, _ = NewAggEvalCtx(aggFuncType, aggFuncArgs, nil, nil, varValuesMap)
	_, _ = eCtx.Eval(exp)
	assert.Equal(t, int64(0), eCtx.GetValue())
	assert.Equal(t, int64(0), eCtx.GetSafeValue(int64(35)))
	assert.True(t, eCtx.IsAggFuncEnabled())

	exp, _ = parser.ParseExpr("sum_if(t1.fieldInt > 0)")
	_, aggFuncType, aggFuncArgs = DetectRootAggFunc(exp)
	eCtx, _ = NewAggEvalCtx(aggFuncType, aggFuncArgs, nil, nil, varValuesMap)
	_, _ = eCtx.Eval(exp)
	assert.Equal(t, int64(0), eCtx.GetValue())
	assert.Equal(t, int64(0), eCtx.GetSafeValue(int64(35)))

	exp, _ = parser.ParseExpr("avg_if(t1.fieldInt > 0)")
	_, aggFuncType, aggFuncArgs = DetectRootAggFunc(exp)
	eCtx, _ = NewAggEvalCtx(aggFuncType, aggFuncArgs, nil, nil, varValuesMap)
	_, _ = eCtx.Eval(exp)
	assert.Equal(t, int64(0), eCtx.GetValue())
	assert.Equal(t, int64(0), eCtx.GetSafeValue(int64(35)))

	exp, _ = parser.ParseExpr("min_if(t1.fieldInt > 0)")
	_, aggFuncType, aggFuncArgs = DetectRootAggFunc(exp)
	eCtx, _ = NewAggEvalCtx(aggFuncType, aggFuncArgs, nil, nil, varValuesMap)
	_, _ = eCtx.Eval(exp)
	assert.Equal(t, nil, eCtx.GetValue())
	assert.Equal(t, int64(35), eCtx.GetSafeValue(int64(35)))

	exp, _ = parser.ParseExpr("max_if(t1.fieldInt > 0)")
	_, aggFuncType, aggFuncArgs = DetectRootAggFunc(exp)
	eCtx, _ = NewAggEvalCtx(aggFuncType, aggFuncArgs, nil, nil, varValuesMap)
	_, _ = eCtx.Eval(exp)
	assert.Equal(t, nil, eCtx.GetValue())
	assert.Equal(t, int64(35), eCtx.GetSafeValue(int64(35)))

	// No if, not a single row eval

	exp, _ = parser.ParseExpr("count()")
	_, aggFuncType, aggFuncArgs = DetectRootAggFunc(exp)
	eCtx, _ = NewAggEvalCtx(aggFuncType, aggFuncArgs, nil, nil, varValuesMap)
	assert.Equal(t, int64(0), eCtx.GetValue())
	assert.Equal(t, int64(0), eCtx.GetSafeValue(int64(35)))

	exp, _ = parser.ParseExpr("sum(t1.fieldInt)")
	_, aggFuncType, aggFuncArgs = DetectRootAggFunc(exp)
	eCtx, _ = NewAggEvalCtx(aggFuncType, aggFuncArgs, nil, nil, varValuesMap)
	assert.Equal(t, int64(0), eCtx.GetValue())
	assert.Equal(t, int64(0), eCtx.GetSafeValue(int64(35)))

	exp, _ = parser.ParseExpr("avg(t1.fieldInt)")
	_, aggFuncType, aggFuncArgs = DetectRootAggFunc(exp)
	eCtx, _ = NewAggEvalCtx(aggFuncType, aggFuncArgs, nil, nil, varValuesMap)
	assert.Equal(t, int64(0), eCtx.GetValue())
	assert.Equal(t, int64(0), eCtx.GetSafeValue(int64(35)))

	exp, _ = parser.ParseExpr("min(t1.fieldInt)")
	_, aggFuncType, aggFuncArgs = DetectRootAggFunc(exp)
	eCtx, _ = NewAggEvalCtx(aggFuncType, aggFuncArgs, nil, nil, varValuesMap)
	assert.Equal(t, nil, eCtx.GetValue())
	assert.Equal(t, int64(35), eCtx.GetSafeValue(int64(35)))

	exp, _ = parser.ParseExpr("max(t1.fieldInt)")
	_, aggFuncType, aggFuncArgs = DetectRootAggFunc(exp)
	eCtx, _ = NewAggEvalCtx(aggFuncType, aggFuncArgs, nil, nil, varValuesMap)
	assert.Equal(t, nil, eCtx.GetValue())
	assert.Equal(t, int64(35), eCtx.GetSafeValue(int64(35)))
}

func TestNoVars(t *testing.T) {
	var exp ast.Expr
	var eCtx *EvalCtx
	var result any

	_, aggFuncType, aggFuncArgs := DetectRootAggFunc(exp)
	eCtx, _ = NewAggEvalCtx(aggFuncType, aggFuncArgs, nil, nil, nil)
	exp, _ = parser.ParseExpr("sum(5)")
	result, _ = eCtx.Eval(exp)
	assert.Equal(t, int64(5), result)
	result, _ = eCtx.Eval(exp)
	assert.Equal(t, int64(5+5), result)
}

func validateArgs(expression string) string {
	varValuesMap := getTestValuesMap()
	exp, _ := parser.ParseExpr(expression)
	_, aggFuncType, aggFuncArgs := DetectRootAggFunc(exp)
	eCtx, err := NewAggEvalCtx(aggFuncType, aggFuncArgs, nil, nil, varValuesMap)
	if err != nil {
		return err.Error()
	}
	_, err = eCtx.Eval(exp)
	return err.Error()
}

func TestBadArgs(t *testing.T) {
	// string_agg/string_agg_if throw special errors because of the additional check performed in getAggStringSeparator
	assert.Contains(t, validateArgs(`string_agg(t1.fieldStr, "-", 123)`), "string_agg must have two parameters")
	assert.Contains(t, validateArgs(`string_agg_if(t1.fieldStr, "-")`), "string_agg_if must have three parameters")
	assert.Contains(t, validateArgs(`count(t1.fieldInt, "-", 123)`), "requires 0 args, 3 supplied")
	assert.Contains(t, validateArgs(`count_if(t1.fieldInt, "-", 123)`), "requires 1 args, 3 supplied")
	assert.Contains(t, validateArgs(`sum(t1.fieldInt, "-", 123)`), "requires 1 args, 3 supplied")
	assert.Contains(t, validateArgs(`sum_if(t1.fieldInt, "-", 123)`), "requires 2 args, 3 supplied")
	assert.Contains(t, validateArgs(`avg(t1.fieldInt, "-", 123)`), "requires 1 args, 3 supplied")
	assert.Contains(t, validateArgs(`avg_if(t1.fieldInt, "-", 123)`), "requires 2 args, 3 supplied")
	assert.Contains(t, validateArgs(`min(t1.fieldInt, "-", 123)`), "requires 1 args, 3 supplied")
	assert.Contains(t, validateArgs(`min_if(t1.fieldInt, "-", 123)`), "requires 2 args, 3 supplied")
	assert.Contains(t, validateArgs(`max(t1.fieldInt, "-", 123)`), "requires 1 args, 3 supplied")
	assert.Contains(t, validateArgs(`max_if(t1.fieldInt, "-", 123)`), "requires 2 args, 3 supplied")

	expected := "unexpected argument 123 of unsupported type int64"
	assert.Contains(t, validateArgs(`string_agg_if(t1.fieldStr, "-", 123)`), expected)
	assert.Contains(t, validateArgs(`count_if(123)`), expected)
	assert.Contains(t, validateArgs(`sum_if(t1.fieldInt, 123)`), expected)
	assert.Contains(t, validateArgs(`avg_if(t1.fieldInt, 123)`), expected)
	assert.Contains(t, validateArgs(`min_if(t1.fieldInt, 123)`), expected)
	assert.Contains(t, validateArgs(`max_if(t1.fieldInt, 123)`), expected)
}

func validateDisabledAggCtx(expression string) string {
	varValuesMap := getTestValuesMap()
	exp, _ := parser.ParseExpr(expression)
	constants := map[string]any{"true": true, "false": false}
	badCtx := NewPlainEvalCtx(nil, constants, varValuesMap)
	_, err := badCtx.Eval(exp)
	return err.Error()
}

func TestDisabledAggCtx(t *testing.T) {
	expected := "context aggregate not enabled"
	assert.Contains(t, validateDisabledAggCtx(`string_agg(t1.fieldStr)`), expected)
	assert.Contains(t, validateDisabledAggCtx(`string_agg_if(t1.fieldStr,true)`), expected)
	assert.Contains(t, validateDisabledAggCtx(`count()`), expected)
	assert.Contains(t, validateDisabledAggCtx(`count_if(true)`), expected)
	assert.Contains(t, validateDisabledAggCtx(`sum(t1.fieldInt)`), expected)
	assert.Contains(t, validateDisabledAggCtx(`sum_if(t1.fieldInt, true)`), expected)
	assert.Contains(t, validateDisabledAggCtx(`avg(t1.fieldInt)`), expected)
	assert.Contains(t, validateDisabledAggCtx(`avg_if(t1.fieldInt, true)`), expected)
	assert.Contains(t, validateDisabledAggCtx(`min(t1.fieldInt)`), expected)
	assert.Contains(t, validateDisabledAggCtx(`min_if(t1.fieldInt, true)`), expected)
	assert.Contains(t, validateDisabledAggCtx(`max(t1.fieldInt)`), expected)
	assert.Contains(t, validateDisabledAggCtx(`max_if(t1.fieldInt, true)`), expected)
}

func validateUnsupportedType(expression string, v any) string {
	varValuesMap := getTestValuesMap()
	exp, _ := parser.ParseExpr(expression)
	_, aggFuncType, aggFuncArgs := DetectRootAggFunc(exp)
	eCtx, _ := NewAggEvalCtx(aggFuncType, aggFuncArgs, nil, nil, varValuesMap)
	varValuesMap["t1"]["fieldInt"] = v
	_, err := eCtx.Eval(exp)
	return err.Error()
}

func TestUnsupportedTypes(t *testing.T) {
	dt := time.Date(2000, 1, 1, 0, 0, 0, 0, time.FixedZone("", -7200))
	assert.Contains(t, validateUnsupportedType(`string_agg(t1.fieldInt, ",")`, int64(1)), "unsupported type int64")
	assert.Contains(t, validateUnsupportedType(`string_agg(t1.fieldInt, ",")`, float64(1.0)), "unsupported type float64")
	assert.Contains(t, validateUnsupportedType(`string_agg(t1.fieldInt, ",")`, true), "unsupported type bool")
	assert.Contains(t, validateUnsupportedType(`string_agg(t1.fieldInt, ",")`, dt), "unsupported type time.Time")
	assert.Contains(t, validateUnsupportedType(`string_agg(t1.fieldInt, ",")`, decimal.NewFromInt(1)), "unsupported type decimal.Decimal")

	expected := "to standard number type, unsuported type"

	assert.Contains(t, validateUnsupportedType(`sum(t1.fieldInt)`, true), expected)
	assert.Contains(t, validateUnsupportedType(`sum(t1.fieldInt)`, dt), expected)
	assert.Contains(t, validateUnsupportedType(`sum(t1.fieldInt)`, "a"), expected)

	assert.Contains(t, validateUnsupportedType(`avg(t1.fieldInt)`, true), expected)
	assert.Contains(t, validateUnsupportedType(`avg(t1.fieldInt)`, dt), expected)
	assert.Contains(t, validateUnsupportedType(`avg(t1.fieldInt)`, "a"), expected)

	assert.Contains(t, validateUnsupportedType(`min(t1.fieldInt)`, true), expected)
	assert.Contains(t, validateUnsupportedType(`min(t1.fieldInt)`, dt), expected)

	assert.Contains(t, validateUnsupportedType(`max(t1.fieldInt)`, true), expected)
	assert.Contains(t, validateUnsupportedType(`max(t1.fieldInt)`, dt), expected)
}

func validateFieldTypeChange(expression string, v1 any, v2 any) string {
	varValuesMap := getTestValuesMap()
	exp, _ := parser.ParseExpr(expression)
	_, aggFuncType, aggFuncArgs := DetectRootAggFunc(exp)
	eCtx, _ := NewAggEvalCtx(aggFuncType, aggFuncArgs, nil, nil, varValuesMap)
	varValuesMap["t1"]["fieldInt"] = v1
	_, err := eCtx.Eval(exp)
	if err != nil {
		return err.Error()
	}
	varValuesMap["t1"]["fieldInt"] = v2
	_, err = eCtx.Eval(exp)
	return err.Error()
}

func TestFieldTypeChange(t *testing.T) {
	i := int64(1)
	f := float64(1.0)
	d := decimal.NewFromInt(1)
	s := "a"

	assert.Contains(t, validateFieldTypeChange(`sum(t1.fieldInt)`, i, f), "started with type int, now got float value")
	assert.Contains(t, validateFieldTypeChange(`sum(t1.fieldInt)`, i, d), "started with type int, now got decimal value")
	assert.Contains(t, validateFieldTypeChange(`sum(t1.fieldInt)`, f, i), "started with type float, now got int value")
	assert.Contains(t, validateFieldTypeChange(`sum(t1.fieldInt)`, f, d), "started with type float, now got decimal value")
	assert.Contains(t, validateFieldTypeChange(`sum(t1.fieldInt)`, d, i), "started with type decimal, now got int value")
	assert.Contains(t, validateFieldTypeChange(`sum(t1.fieldInt)`, d, f), "started with type decimal, now got float value")

	assert.Contains(t, validateFieldTypeChange(`avg(t1.fieldInt)`, i, f), "started with type int, now got float value")
	assert.Contains(t, validateFieldTypeChange(`avg(t1.fieldInt)`, i, d), "started with type int, now got decimal value")
	assert.Contains(t, validateFieldTypeChange(`avg(t1.fieldInt)`, f, i), "started with type float, now got int value")
	assert.Contains(t, validateFieldTypeChange(`avg(t1.fieldInt)`, f, d), "started with type float, now got decimal value")
	assert.Contains(t, validateFieldTypeChange(`avg(t1.fieldInt)`, d, i), "started with type decimal, now got int value")
	assert.Contains(t, validateFieldTypeChange(`avg(t1.fieldInt)`, d, f), "started with type decimal, now got float value")

	assert.Contains(t, validateFieldTypeChange(`min(t1.fieldInt)`, i, f), "started with type int, now got float value")
	assert.Contains(t, validateFieldTypeChange(`min(t1.fieldInt)`, i, d), "started with type int, now got decimal value")
	assert.Contains(t, validateFieldTypeChange(`min(t1.fieldInt)`, i, s), "started with type int, now got string value")
	assert.Contains(t, validateFieldTypeChange(`min(t1.fieldInt)`, f, i), "started with type float, now got int value")
	assert.Contains(t, validateFieldTypeChange(`min(t1.fieldInt)`, f, d), "started with type float, now got decimal value")
	assert.Contains(t, validateFieldTypeChange(`min(t1.fieldInt)`, f, s), "started with type float, now got string value")
	assert.Contains(t, validateFieldTypeChange(`min(t1.fieldInt)`, d, i), "started with type decimal, now got int value")
	assert.Contains(t, validateFieldTypeChange(`min(t1.fieldInt)`, d, f), "started with type decimal, now got float value")
	assert.Contains(t, validateFieldTypeChange(`min(t1.fieldInt)`, d, s), "started with type decimal, now got string value")
	assert.Contains(t, validateFieldTypeChange(`min(t1.fieldInt)`, s, i), "started with type string, now got int value")
	assert.Contains(t, validateFieldTypeChange(`min(t1.fieldInt)`, s, f), "started with type string, now got float value")
	assert.Contains(t, validateFieldTypeChange(`min(t1.fieldInt)`, s, d), "started with type string, now got decimal value")

	assert.Contains(t, validateFieldTypeChange(`max(t1.fieldInt)`, i, f), "started with type int, now got float value")
	assert.Contains(t, validateFieldTypeChange(`max(t1.fieldInt)`, i, d), "started with type int, now got decimal value")
	assert.Contains(t, validateFieldTypeChange(`max(t1.fieldInt)`, i, s), "started with type int, now got string value")
	assert.Contains(t, validateFieldTypeChange(`max(t1.fieldInt)`, f, i), "started with type float, now got int value")
	assert.Contains(t, validateFieldTypeChange(`max(t1.fieldInt)`, f, d), "started with type float, now got decimal value")
	assert.Contains(t, validateFieldTypeChange(`max(t1.fieldInt)`, f, s), "started with type float, now got string value")
	assert.Contains(t, validateFieldTypeChange(`max(t1.fieldInt)`, d, i), "started with type decimal, now got int value")
	assert.Contains(t, validateFieldTypeChange(`max(t1.fieldInt)`, d, f), "started with type decimal, now got float value")
	assert.Contains(t, validateFieldTypeChange(`max(t1.fieldInt)`, d, s), "started with type decimal, now got string value")
	assert.Contains(t, validateFieldTypeChange(`max(t1.fieldInt)`, s, i), "started with type string, now got int value")
	assert.Contains(t, validateFieldTypeChange(`max(t1.fieldInt)`, s, f), "started with type string, now got float value")
	assert.Contains(t, validateFieldTypeChange(`max(t1.fieldInt)`, s, d), "started with type string, now got decimal value")
}

func TestBlanks(t *testing.T) {
	eCtx := newPlainEvalCtx(AggFuncEnabled)
	assert.Equal(t, int64(0), eCtx.sumCollector.Int)
	assert.Equal(t, float64(0), eCtx.sumCollector.Float)
	assert.Equal(t, defaultDecimal(), eCtx.sumCollector.Dec)

	assert.Equal(t, int64(0), eCtx.avgCollector.Int)
	assert.Equal(t, float64(0), eCtx.avgCollector.Float)
	assert.Equal(t, defaultDecimal(), eCtx.avgCollector.Dec)

	assert.Equal(t, maxSupportedInt, eCtx.minCollector.Int)
	assert.Equal(t, maxSupportedFloat, eCtx.minCollector.Float)
	assert.Equal(t, maxSupportedDecimal(), eCtx.minCollector.Dec)
	assert.Equal(t, "", eCtx.minCollector.Str)

	assert.Equal(t, minSupportedInt, eCtx.maxCollector.Int)
	assert.Equal(t, minSupportedFloat, eCtx.maxCollector.Float)
	assert.Equal(t, minSupportedDecimal(), eCtx.maxCollector.Dec)
	assert.Equal(t, "", eCtx.maxCollector.Str)
}

// This is a demonstration of the importance of using precision on every step of the agg calculation,
// instead of just rounding the final result.
func TestAggPrecision(t *testing.T) {
	varValuesMap := VarValuesMap{
		"": map[string]any{},
	}
	exp, _ := parser.ParseExpr("avg(price*tax)")
	var eCtx *EvalCtx

	// 1. This is the case when only final rounding (no intermediate rounding) works:
	// round(1.333.., 2) == 1.33

	// Tax 2.00, no precision
	// avg(0.50*2.00, 1.00*2.00, 0.50*2.00) = 1.333...
	varValuesMap[""]["tax"] = decimal.NewFromFloat(2.00)
	eCtx, _ = NewAggEvalCtx(AggAvg, exp.(*ast.CallExpr).Args, nil, nil, nil)
	varValuesMap[""]["price"] = decimal.NewFromFloat(0.50)
	eCtx.SetVars(varValuesMap)
	_, _ = eCtx.Eval(exp)
	varValuesMap[""]["price"] = decimal.NewFromFloat(1.00)
	eCtx.SetVars(varValuesMap)
	_, _ = eCtx.Eval(exp)
	varValuesMap[""]["price"] = decimal.NewFromFloat(0.50)
	eCtx.SetVars(varValuesMap)
	_, _ = eCtx.Eval(exp)
	assert.Equal(t, decimal.NewFromFloat(1.3333333333333333).String(), eCtx.GetValue().(decimal.Decimal).String())

	// Tax 2.00, precision 2
	// avg(0.50*2.00, 1.00*2.00, 0.50*2.00) = 1.33
	varValuesMap[""]["tax"] = decimal.NewFromFloat(2.00)
	eCtx, _ = NewAggEvalCtx(AggAvg, exp.(*ast.CallExpr).Args, nil, nil, nil)
	eCtx.SetRoundDec(2)
	varValuesMap[""]["price"] = decimal.NewFromFloat(0.50)
	eCtx.SetVars(varValuesMap)
	_, _ = eCtx.Eval(exp)
	varValuesMap[""]["price"] = decimal.NewFromFloat(1.00)
	eCtx.SetVars(varValuesMap)
	_, _ = eCtx.Eval(exp)
	varValuesMap[""]["price"] = decimal.NewFromFloat(0.50)
	eCtx.SetVars(varValuesMap)
	_, _ = eCtx.Eval(exp)
	assert.Equal(t, decimal.NewFromFloat(1.33).String(), eCtx.GetValue().(decimal.Decimal).String())

	// 2. This is the case when only final rounding (no intermediate rounding)
	// does not give the same result, rounding on every step is essential:
	// round(1.336, 2) != 1.33

	// Tax 2.009, no precision
	// avg(0.50*2.009, 1.00*2.009, 0.50*2.009) =
	// avg(1.0045, 2.009, 1.0045) = 4.018 / 3 = 1.336
	varValuesMap[""]["tax"] = decimal.NewFromFloat(2.004)
	eCtx, _ = NewAggEvalCtx(AggAvg, exp.(*ast.CallExpr).Args, nil, nil, nil)
	varValuesMap[""]["price"] = decimal.NewFromFloat(0.50)
	eCtx.SetVars(varValuesMap)
	_, _ = eCtx.Eval(exp)
	varValuesMap[""]["price"] = decimal.NewFromFloat(1.00)
	eCtx.SetVars(varValuesMap)
	_, _ = eCtx.Eval(exp)
	varValuesMap[""]["price"] = decimal.NewFromFloat(0.50)
	eCtx.SetVars(varValuesMap)
	_, _ = eCtx.Eval(exp)
	assert.Equal(t, decimal.NewFromFloat(1.336).String(), eCtx.GetValue().(decimal.Decimal).String())

	// Tax 2.009, precision 2
	// avg(round(0.50*2.009, 2), round(1.00*2.009, 2), round(0.50*2.009, 2)) =
	// avg(1.00, 2.00, 1.00) = 4.00 / 3 = 1.333...
	varValuesMap[""]["tax"] = decimal.NewFromFloat(2.004)
	eCtx, _ = NewAggEvalCtx(AggAvg, exp.(*ast.CallExpr).Args, nil, nil, nil)
	eCtx.SetRoundDec(2)
	varValuesMap[""]["price"] = decimal.NewFromFloat(0.50)
	eCtx.SetVars(varValuesMap)
	_, _ = eCtx.Eval(exp)
	varValuesMap[""]["price"] = decimal.NewFromFloat(1.00)
	eCtx.SetVars(varValuesMap)
	_, _ = eCtx.Eval(exp)
	varValuesMap[""]["price"] = decimal.NewFromFloat(0.50)
	eCtx.SetVars(varValuesMap)
	_, _ = eCtx.Eval(exp)
	assert.Equal(t, decimal.NewFromFloat(1.33).String(), eCtx.GetValue().(decimal.Decimal).String())
}
