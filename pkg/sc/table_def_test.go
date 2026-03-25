package sc

import (
	"testing"
	"time"

	"github.com/capillariesio/capillaries/pkg/eval_capi"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestTableDefCheckValueTypeFailures(t *testing.T) {
	err := CheckValueType(int64(1), eval_capi.FieldTypeBool)
	assert.Contains(t, err.Error(), "expected type bool, but got int64")

	err = CheckValueType(float64(1.1), eval_capi.FieldTypeBool)
	assert.Contains(t, err.Error(), "expected type bool, but got float64")

	err = CheckValueType(decimal.NewFromFloat(1.1), eval_capi.FieldTypeBool)
	assert.Contains(t, err.Error(), "expected type bool, but got decimal")

	err = CheckValueType(time.Now(), eval_capi.FieldTypeBool)
	assert.Contains(t, err.Error(), "expected type bool, but got datetime")

	err = CheckValueType("aaa", eval_capi.FieldTypeBool)
	assert.Contains(t, err.Error(), "expected type bool, but got string")

	err = CheckValueType(true, eval_capi.FieldTypeInt)
	assert.Contains(t, err.Error(), "expected type int, but got bool")

	err = CheckValueType([]string{"aaa"}, eval_capi.FieldTypeInt)
	assert.Contains(t, err.Error(), "expected type int, but got unexpected type []string")
}
