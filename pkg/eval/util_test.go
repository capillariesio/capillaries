package eval

import (
	"fmt"
	"go/parser"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

func assertEqual(t *testing.T, expString string, expectedResult any, varValuesMap VarValuesMap) {
	exp, err1 := parser.ParseExpr(expString)
	if err1 != nil {
		t.Error(fmt.Errorf("%s: %s", expString, err1.Error()))
		return
	}
	constants := map[string]any{"true": true, "false": false}
	eCtx := NewPlainEvalCtx(nil, constants, varValuesMap)
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
	eCtx := NewPlainEvalCtx(nil, nil, varValuesMap)
	result, err2 := eCtx.Eval(exp)
	if err2 != nil {
		t.Error(fmt.Errorf("%s: %s", expString, err2.Error()))
		return
	}
	floatResult, ok := result.(float64)
	assert.True(t, ok)
	assert.True(t, math.IsNaN(floatResult))
}

func assertEvalError(t *testing.T, expString string, expectedErrorMsg string, varValuesMap VarValuesMap) {
	exp, err1 := parser.ParseExpr(expString)
	if err1 != nil {
		assert.Contains(t, err1.Error(), expectedErrorMsg, fmt.Sprintf("Unmatched: %v = %v: %s ", expectedErrorMsg, err1.Error(), expString))
		return
	}
	constants := map[string]any{"true": true, "false": false}
	eCtx := NewPlainEvalCtx(nil, constants, varValuesMap)
	_, err2 := eCtx.Eval(exp)

	assert.Contains(t, err2.Error(), expectedErrorMsg, fmt.Sprintf("Unmatched: %v = %v: %s ", expectedErrorMsg, err2.Error(), expString))
}
