package eval

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVarValuesMapUtils(t *testing.T) {
	varValuesMap := VarValuesMap{"some_table": map[string]interface{}{"some_field": 1}}
	assert.Equal(t, "[some_table ]", varValuesMap.Tables())
	assert.Equal(t, "[some_table.some_field ]", varValuesMap.Names())
}
