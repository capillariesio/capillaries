package sc

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestJsonUnmarshal(t *testing.T) {
	var o map[string]any
	err := JsonOrYamlUnmarshal(ScriptJson, []byte(`{"a": {"a1":1}, "b": [2]}`), &o)
	assert.Nil(t, err)
	assert.Equal(t, float64(1), o["a"].(map[string]any)["a1"])
	assert.Equal(t, float64(2), o["b"].([]any)[0])
}

func TestYamlUnmarshal(t *testing.T) {
	yaml := `
a:
  a1: 1
b:
  - 2`
	var o map[string]any
	err := JsonOrYamlUnmarshal(ScriptYaml, []byte(yaml), &o)
	assert.Nil(t, err)
	assert.Equal(t, float64(1), o["a"].(map[string]any)["a1"])
	assert.Equal(t, float64(2), o["b"].([]any)[0])
}
