package eval

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestCastSingle(t *testing.T) {
	var val interface{}
	var err error

	val, _ = castNumberToStandardType(int(12))
	assert.Equal(t, int64(12), val)
	val, _ = castNumberToStandardType(int16(12))
	assert.Equal(t, int64(12), val)
	val, _ = castNumberToStandardType(int32(12))
	assert.Equal(t, int64(12), val)
	val, _ = castNumberToStandardType(int64(12))
	assert.Equal(t, int64(12), val)
	val, _ = castNumberToStandardType(float32(12))
	assert.Equal(t, float64(12), val)
	val, _ = castNumberToStandardType(float64(12))
	assert.Equal(t, float64(12), val)
	val, _ = castNumberToStandardType(decimal.NewFromInt(12))
	assert.Equal(t, decimal.NewFromInt(12), val)
	_, err = castNumberToStandardType("12")
	assert.Equal(t, "cannot cast 12(string) to standard number type, unsuported type", err.Error())

	val, _ = castToInt64(int(12))
	assert.Equal(t, int64(12), val)
	val, _ = castToInt64(int16(12))
	assert.Equal(t, int64(12), val)
	val, _ = castToInt64(int32(12))
	assert.Equal(t, int64(12), val)
	val, _ = castToInt64(int64(12))
	assert.Equal(t, int64(12), val)
	val, _ = castToInt64(float32(12))
	assert.Equal(t, int64(12), val)
	val, _ = castToInt64(float64(12))
	assert.Equal(t, int64(12), val)
	val, _ = castToInt64(decimal.NewFromInt(12))
	assert.Equal(t, int64(12), val)
	_, err = castToInt64("12")
	assert.Equal(t, "cannot cast 12(string) to int64, unsuported type", err.Error())
	_, err = castToInt64(decimal.NewFromFloat(12.2))
	assert.Equal(t, "cannot cast decimal '12.2' to int64, exact conversion impossible", err.Error())

	val, _ = castToFloat64(int(12))
	assert.Equal(t, float64(12.0), val)
	val, _ = castToFloat64(int16(12))
	assert.Equal(t, float64(12.0), val)
	val, _ = castToFloat64(int32(12))
	assert.Equal(t, float64(12.0), val)
	val, _ = castToFloat64(float64(12.0))
	assert.Equal(t, float64(12.0), val)
	val, _ = castToFloat64(float32(12))
	assert.Equal(t, float64(12.0), val)
	val, _ = castToFloat64(float64(12))
	assert.Equal(t, float64(12.0), val)
	val, _ = castToFloat64(decimal.NewFromInt(12))
	assert.Equal(t, float64(12.0), val)
	_, err = castToFloat64("12")
	assert.Equal(t, "cannot cast 12(string) to float64, unsuported type", err.Error())
	val, _ = castToFloat64(decimal.NewFromFloat(12.2))
	assert.Equal(t, float64(12.2), val)

	val, _ = castToDecimal2(int(12))
	assert.Equal(t, decimal.NewFromFloat(12.0), val)
	val, _ = castToDecimal2(int16(12))
	assert.Equal(t, decimal.NewFromFloat(12.0), val)
	val, _ = castToDecimal2(int32(12))
	assert.Equal(t, decimal.NewFromFloat(12.0), val)
	val, _ = castToDecimal2(float64(12.1))
	assert.Equal(t, decimal.NewFromFloat(12.1), val)
	val, _ = castToDecimal2(float32(12.1))
	assert.Equal(t, decimal.NewFromFloat(12.1), val)
	val, _ = castToDecimal2(float64(12.1))
	assert.Equal(t, decimal.NewFromFloat(12.1), val)
	val, _ = castToDecimal2(decimal.NewFromFloat(12.1))
	assert.Equal(t, decimal.NewFromFloat(12.1), val)
	_, err = castToDecimal2("12")
	assert.Equal(t, "cannot cast 12(string) to decimal2, unsuported type", err.Error())
}

func TestCastPair(t *testing.T) {
	var vLeft, vRight interface{}
	var err error

	vLeft, vRight, _ = castNumberPairToCommonType(float32(12), int(13))
	assert.Equal(t, float64(12), vLeft)
	assert.Equal(t, float64(13), vRight)

	vLeft, vRight, _ = castNumberPairToCommonType(decimal.NewFromInt(12), int16(13))
	assert.Equal(t, decimal.NewFromInt(12), vLeft)
	assert.Equal(t, decimal.NewFromInt(13), vRight)

	vLeft, vRight, _ = castNumberPairToCommonType(int32(12), int(13))
	assert.Equal(t, int64(12), vLeft)
	assert.Equal(t, int64(13), vRight)

	_, _, err = castNumberPairToCommonType("12", int(13))
	assert.Equal(t, "invalid left arg: cannot cast 12(string) to standard number type, unsuported type", err.Error())

	_, _, err = castNumberPairToCommonType(int(132), "13")
	assert.Equal(t, "invalid right arg: cannot cast 13(string) to standard number type, unsuported type", err.Error())
}
