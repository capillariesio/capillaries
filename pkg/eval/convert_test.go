package eval

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestConvert(t *testing.T) {
	var val interface{}
	var err error
	var testTime = time.Date(1, 1, 1, 1, 1, 1, 1, time.UTC)

	val, _ = callString([]interface{}{12.2})
	assert.Equal(t, "12.2", val)
	val, _ = callString([]interface{}{true})
	assert.Equal(t, "true", val)
	val, _ = callString([]interface{}{false})
	assert.Equal(t, "false", val)
	val, _ = callString([]interface{}{testTime})
	assert.Equal(t, "0001-01-01 01:01:01.000000001 +0000 UTC", val)
	_, err = callString([]interface{}{12.2, 13.0})
	assert.Equal(t, "cannot evaluate string(), requires 1 args, 2 supplied", err.Error())

	val, _ = callInt([]interface{}{int64(12)})
	assert.Equal(t, int64(12), val)
	val, _ = callInt([]interface{}{"12"})
	assert.Equal(t, int64(12), val)
	val, _ = callInt([]interface{}{true})
	assert.Equal(t, int64(1), val)
	val, _ = callInt([]interface{}{false})
	assert.Equal(t, int64(0), val)
	_, err = callInt([]interface{}{testTime})
	assert.Equal(t, `unsupported arg type for int(0001-01-01 01:01:01.000000001 +0000 UTC):time.Time`, err.Error())
	_, err = callInt([]interface{}{"12.2"})
	assert.Equal(t, `cannot eval int(12.2):strconv.ParseInt: parsing "12.2": invalid syntax`, err.Error())
	_, err = callInt([]interface{}{"12.0", "13.0"})
	assert.Equal(t, "cannot evaluate int(), requires 1 args, 2 supplied", err.Error())

	val, _ = callDecimal2([]interface{}{int64(12)})
	assert.Equal(t, decimal.NewFromInt(12), val)
	val, _ = callDecimal2([]interface{}{"12"})
	assert.Equal(t, decimal.NewFromInt(12), val)
	val, _ = callDecimal2([]interface{}{true})
	assert.Equal(t, decimal.NewFromInt(1), val)
	val, _ = callDecimal2([]interface{}{false})
	assert.Equal(t, decimal.NewFromInt(0), val)
	_, err = callDecimal2([]interface{}{testTime})
	assert.Equal(t, `unsupported arg type for decimal2(0001-01-01 01:01:01.000000001 +0000 UTC):time.Time`, err.Error())
	_, err = callDecimal2([]interface{}{"somestring"})
	assert.Equal(t, "cannot eval decimal2(somestring):can't convert somestring to decimal: exponent is not numeric", err.Error())
	_, err = callDecimal2([]interface{}{"12.0", "13.0"})
	assert.Equal(t, "cannot evaluate decimal2(), requires 1 args, 2 supplied", err.Error())

	val, _ = callFloat([]interface{}{int64(12)})
	assert.Equal(t, float64(12), val)
	val, _ = callFloat([]interface{}{"12.1"})
	assert.Equal(t, float64(12.1), val)
	val, _ = callFloat([]interface{}{decimal.NewFromFloat(12.2)})
	assert.Equal(t, float64(12.2), val)
	val, _ = callFloat([]interface{}{true})
	assert.Equal(t, float64(1), val)
	val, _ = callFloat([]interface{}{false})
	assert.Equal(t, float64(0), val)
	_, err = callFloat([]interface{}{testTime})
	assert.Equal(t, `unsupported arg type for float(0001-01-01 01:01:01.000000001 +0000 UTC):time.Time`, err.Error())
	_, err = callFloat([]interface{}{"somestring"})
	assert.Equal(t, `cannot eval float(somestring):strconv.ParseFloat: parsing "somestring": invalid syntax`, err.Error())
	_, err = callFloat([]interface{}{"12.0", "13.0"})
	assert.Equal(t, "cannot evaluate float(), requires 1 args, 2 supplied", err.Error())
}
