package eval_capi

import (
	"fmt"
	"go/parser"
	"math"
	"testing"

	"github.com/capillariesio/capillaries/pkg/eval"
	"github.com/stretchr/testify/assert"
)

func assertEqual(t *testing.T, expString string, expectedResult any, varValuesMap eval.VarValuesMap) {
	exp, err1 := parser.ParseExpr(expString)
	if err1 != nil {
		t.Error(fmt.Errorf("%s: %s", expString, err1.Error()))
		return
	}
	eCtx := eval.NewPlainEvalCtxWithVars(eval.AggFuncDisabled, CapillariesEvalFunctions, CapillariesEvalConstants, varValuesMap)
	result, err2 := eCtx.Eval(exp)
	if err2 != nil {
		t.Error(fmt.Errorf("%s: %s", expString, err2.Error()))
		return
	}

	assert.Equal(t, expectedResult, result, fmt.Sprintf("Unmatched: %v = %v: %s ", expectedResult, result, expString))
}

func assertFloatNan(t *testing.T, expString string, varValuesMap eval.VarValuesMap) {
	exp, err1 := parser.ParseExpr(expString)
	if err1 != nil {
		t.Error(fmt.Errorf("%s: %s", expString, err1.Error()))
		return
	}
	eCtx := eval.NewPlainEvalCtxWithVars(eval.AggFuncDisabled, CapillariesEvalFunctions, CapillariesEvalConstants, varValuesMap)
	result, err2 := eCtx.Eval(exp)
	if err2 != nil {
		t.Error(fmt.Errorf("%s: %s", expString, err2.Error()))
		return
	}
	floatResult, ok := result.(float64)
	assert.True(t, ok)
	assert.True(t, math.IsNaN(floatResult))
}

func assertEvalError(t *testing.T, expString string, expectedErrorMsg string, varValuesMap eval.VarValuesMap) {
	exp, err1 := parser.ParseExpr(expString)
	if err1 != nil {
		assert.Contains(t, err1.Error(), expectedErrorMsg, fmt.Sprintf("Unmatched: %v = %v: %s ", expectedErrorMsg, err1.Error(), expString))
		return
	}
	eCtx := eval.NewPlainEvalCtxWithVars(eval.AggFuncDisabled, CapillariesEvalFunctions, CapillariesEvalConstants, varValuesMap)
	_, err2 := eCtx.Eval(exp)

	assert.Contains(t, err2.Error(), expectedErrorMsg, fmt.Sprintf("Unmatched: %v = %v: %s ", expectedErrorMsg, err2.Error(), expString))
}
