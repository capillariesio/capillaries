package eval

import (
	"fmt"
	"testing"
)

func TestStringsFunctions(t *testing.T) {
	varValuesMap := VarValuesMap{}
	assertEqual(t, fmt.Sprintf(`strings.ReplaceAll("abc","a","b")`), "bbc", varValuesMap)
}
