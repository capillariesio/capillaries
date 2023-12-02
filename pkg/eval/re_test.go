package eval

import (
	"testing"
)

func TestReFunctions(t *testing.T) {
	var vars VarValuesMap
	var re string

	vars = VarValuesMap{"r": map[string]any{"product_spec": `{"k":"Ideal For","v":"Boys, Men, Girls, Women"}`}}
	re = "re.MatchString(`\"k\":\"Ideal For\",\"v\":\"[\\w ,]*Boys[\\w ,]*\"`, r.product_spec)"
	assertEqual(t, re, true, vars)

	vars = VarValuesMap{"r": map[string]any{"product_spec": `{"k":"Water Resistance Depth","v":"100 m"}`}}
	re = "re.MatchString(`\"k\":\"Water Resistance Depth\",\"v\":\"(100|200) m\"`, r.product_spec)"
	assertEqual(t, re, true, vars)

	vars = VarValuesMap{"r": map[string]any{"product_spec": `{"k":"Occasion","v":"Ethnic, Casual, Party, Formal"}`}}
	re = "re.MatchString(`\"k\":\"Occasion\",\"v\":\"[\\w ,]*(Casual|Festive)[\\w ,]*\"`, r.product_spec)"
	assertEqual(t, re, true, vars)

	vars = VarValuesMap{"r": map[string]any{"product_spec": `{"k":"Base Material","v":"Gold"},{"k":"Gemstone","v":"Diamond"}`, "retail_price": 101}}
	re = "re.MatchString(`\"k\":\"Base Material\",\"v\":\"Gold\"`, r.product_spec) && re.MatchString(`\"k\":\"Gemstone\",\"v\":\"Diamond\"`, r.product_spec) && r.retail_price > 100"
	assertEqual(t, re, true, vars)

	vars = VarValuesMap{"r": map[string]any{"product_spec": `{"k":"Base Material","v":"Gold"},{"k":"Gemstone","v":"Diamond"}`, "retail_price": 100}}
	re = "re.MatchString(`\"k\":\"Base Material\",\"v\":\"Gold\"`, r.product_spec) && re.MatchString(`\"k\":\"Gemstone\",\"v\":\"Diamond\"`, r.product_spec) && r.retail_price > 100"
	assertEqual(t, re, false, vars)

	assertEvalError(t, `re.MatchString("a")`, "cannot evaluate re.MatchString(), requires 2 args, 1 supplied", vars)
	assertEvalError(t, `re.MatchString("a",1)`, "cannot convert re.MatchString() args a and 1 to string", vars)

}
