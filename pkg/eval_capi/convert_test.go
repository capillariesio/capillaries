package eval_capi

import (
	"fmt"
	"testing"
	"time"

	"github.com/capillariesio/capillaries/pkg/eval"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestConvertEval(t *testing.T) {
	varValuesMap := eval.VarValuesMap{
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

func TestConvert(t *testing.T) {
	var val any
	var err error
	var testTime = time.Date(1, 1, 1, 1, 1, 1, 1, time.UTC)

	val, _ = callString([]any{12.2})
	assert.Equal(t, "12.2", val)
	val, _ = callString([]any{true})
	assert.Equal(t, "true", val)
	val, _ = callString([]any{false})
	assert.Equal(t, "false", val)
	val, _ = callString([]any{testTime})
	assert.Equal(t, "0001-01-01 01:01:01.000000001 +0000 UTC", val)
	_, err = callString([]any{12.2, 13.0})
	assert.Equal(t, "cannot evaluate string(), requires 1 args, 2 supplied", err.Error())

	val, _ = callInt([]any{int64(12)})
	assert.Equal(t, int64(12), val)
	val, _ = callInt([]any{"12"})
	assert.Equal(t, int64(12), val)
	val, _ = callInt([]any{true})
	assert.Equal(t, int64(1), val)
	val, _ = callInt([]any{false})
	assert.Equal(t, int64(0), val)
	_, err = callInt([]any{testTime})
	assert.Equal(t, `unsupported arg type for int(0001-01-01 01:01:01.000000001 +0000 UTC):time.Time`, err.Error())
	_, err = callInt([]any{"12.2"})
	assert.Equal(t, `cannot eval int(12.2):strconv.ParseInt: parsing "12.2": invalid syntax`, err.Error())
	_, err = callInt([]any{"12.0", "13.0"})
	assert.Equal(t, "cannot evaluate int(), requires 1 args, 2 supplied", err.Error())

	val, _ = callDecimal2([]any{int64(12)})
	assert.Equal(t, decimal.NewFromInt(12), val)
	val, _ = callDecimal2([]any{"12"})
	assert.Equal(t, decimal.NewFromInt(12), val)
	val, _ = callDecimal2([]any{true})
	assert.Equal(t, decimal.NewFromInt(1), val)
	val, _ = callDecimal2([]any{false})
	assert.Equal(t, decimal.NewFromInt(0), val)
	_, err = callDecimal2([]any{testTime})
	assert.Equal(t, `unsupported arg type for decimal2(0001-01-01 01:01:01.000000001 +0000 UTC):time.Time`, err.Error())
	_, err = callDecimal2([]any{"somestring"})
	assert.Equal(t, "cannot eval decimal2(somestring):can't convert somestring to decimal: exponent is not numeric", err.Error())
	_, err = callDecimal2([]any{"12.0", "13.0"})
	assert.Equal(t, "cannot evaluate decimal2(), requires 1 args, 2 supplied", err.Error())

	val, _ = callFloat([]any{int64(12)})
	assert.Equal(t, float64(12), val)
	val, _ = callFloat([]any{"12.1"})
	assert.Equal(t, float64(12.1), val)
	val, _ = callFloat([]any{decimal.NewFromFloat(12.2)})
	assert.Equal(t, float64(12.2), val)
	val, _ = callFloat([]any{true})
	assert.Equal(t, float64(1), val)
	val, _ = callFloat([]any{false})
	assert.Equal(t, float64(0), val)
	_, err = callFloat([]any{testTime})
	assert.Equal(t, `unsupported arg type for float(0001-01-01 01:01:01.000000001 +0000 UTC):time.Time`, err.Error())
	_, err = callFloat([]any{"somestring"})
	assert.Equal(t, `cannot eval float(somestring):strconv.ParseFloat: parsing "somestring": invalid syntax`, err.Error())
	_, err = callFloat([]any{"12.0", "13.0"})
	assert.Equal(t, "cannot evaluate float(), requires 1 args, 2 supplied", err.Error())
}
