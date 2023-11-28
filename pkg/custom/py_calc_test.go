package custom

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/capillariesio/capillaries/pkg/sc"
	"github.com/stretchr/testify/assert"
)

type PyCalcTestTestProcessorDefFactory struct {
}

func (f *PyCalcTestTestProcessorDefFactory) Create(processorType string) (sc.CustomProcessorDef, bool) {
	switch processorType {
	case ProcessorPyCalcName:
		return &PyCalcProcessorDef{}, true
	default:
		return nil, false
	}
}

func TestPyCalcDef(t *testing.T) {
	script := `
	{
		"nodes": {
			"read_table1": {
				"type": "file_table",
				"r": {
					"urls": [
						"file1.csv"
					],
					"csv":{
					    "first_data_line_idx": 0
					},
					"columns": {
						"col_field_int": {
							"csv":{"col_idx": 0},
							"col_type": "int"
						},
						"col_field_string": {
							"csv":{"col_idx": 1},
							"col_type": "string"
						}
					}
				},
				"w": {
					"name": "table1",
					"having": "w.field_int1 > 1",
					"fields": {
						"field_int1": {
							"expression": "r.col_field_int",
							"type": "int"
						},
						"field_string1": {
							"expression": "r.col_field_string",
							"type": "string"
						}
					}
				}
			},
			"tax_table1": {
				"type": "table_custom_tfm_table",
				"custom_proc_type": "py_calc",
				"r": {
					"table": "table1"
				},
				"p": {
					"python_code_urls": [
						"../../test/data/cfg/py_calc_quicktest/py/calc_order_items_code.py"
					],
					"calculated_fields": {
						"taxed_field_int1": {
							"expression": "increase_by_ten_percent(r.field_int1)",
							"type": "int"
						}
					}
				},
				"w": {
					"name": "taxed_table1",
					"having": "w.taxed_field_decimal > 10",
					"fields": {
						"field_int1": {
							"expression": "r.field_int1",
							"type": "int"
						},
						"field_string1": {
							"expression": "r.field_string1",
							"type": "string"
						},
						"taxed_field_decimal": {
							"expression": "decimal2(p.taxed_field_int1)",
							"type": "decimal2"
						}
					}
				}
			},
			"file_taxed_table1": {
				"type": "table_file",
				"r": {
					"table": "taxed_table1"
				},
				"w": {
					"top": {
						"order": "taxed_field_int1(asc)"
					},
					"url_template": "taxed_table1.csv",
					"columns": [
						{
							"csv":{
								"header": "field_int1",
								"format": "%d"
							},
							"name": "field_int1",
							"expression": "r.field_int1",
							"type": "int"
						},
						{
							"csv":{
								"header": "field_string1",
								"format": "%s"
							},
							"name": "field_string1",
							"expression": "r.field_string1",
							"type": "string"
						},
						{
							"csv":{
								"header": "taxed_field_decimal",
								"format": "%s"
							},
							"name": "taxed_field_decimal",
							"expression": "r.taxed_field_decimal",
							"type": "decimal2"
						}
					]
				}
			}
		},
		"dependency_policies": {
			"current_active_first_stopped_nogo":` + sc.DefaultPolicyCheckerConf +
		`		
		}
	}`

	envSettings := `
	{
        "python_interpreter_path":"python",
        "python_interpreter_params":["-u", "-"]
    }`

	var err error

	newScript := &sc.ScriptDef{}
	if err = newScript.Deserialize([]byte(script), &PyCalcTestTestProcessorDefFactory{}, map[string]json.RawMessage{"py_calc": []byte(envSettings)}, "", nil); err != nil {
		t.Error(err)
	}

	err = newScript.Deserialize(
		[]byte(strings.Replace(script, `"having": "w.taxed_field_decimal > 10"`, `"having": "p.taxed_field_int1 > 10"`, 1)),
		&PyCalcTestTestProcessorDefFactory{}, map[string]json.RawMessage{"py_calc": []byte(envSettings)}, "", nil)
	assert.Contains(t, err.Error(), "prohibited field p.taxed_field_int1")

	err = newScript.Deserialize(
		[]byte(strings.Replace(script, `"expression": "increase_by_ten_percent(r.field_int1)"`, `"expression": "bad_func(r.field_int1)"`, 1)),
		&PyCalcTestTestProcessorDefFactory{}, map[string]json.RawMessage{"py_calc": []byte(envSettings)}, "", nil)
	assert.Contains(t, err.Error(), "function def 'bad_func(arg)' not found in Python file")
}
