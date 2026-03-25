package sc

import (
	"testing"
	"time"

	"github.com/capillariesio/capillaries/pkg/evalcapi"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestTableDefCheckValueTypeFailures(t *testing.T) {
	err := CheckValueType(int64(1), evalcapi.FieldTypeBool)
	assert.Contains(t, err.Error(), "expected type bool, but got int64")

	err = CheckValueType(float64(1.1), evalcapi.FieldTypeBool)
	assert.Contains(t, err.Error(), "expected type bool, but got float64")

	err = CheckValueType(decimal.NewFromFloat(1.1), evalcapi.FieldTypeBool)
	assert.Contains(t, err.Error(), "expected type bool, but got decimal")

	err = CheckValueType(time.Now(), evalcapi.FieldTypeBool)
	assert.Contains(t, err.Error(), "expected type bool, but got datetime")

	err = CheckValueType("aaa", evalcapi.FieldTypeBool)
	assert.Contains(t, err.Error(), "expected type bool, but got string")

	err = CheckValueType(true, evalcapi.FieldTypeInt)
	assert.Contains(t, err.Error(), "expected type int, but got bool")

	err = CheckValueType([]string{"aaa"}, evalcapi.FieldTypeInt)
	assert.Contains(t, err.Error(), "expected type int, but got unexpected type []string")
}
