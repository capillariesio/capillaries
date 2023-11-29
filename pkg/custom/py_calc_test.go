package custom

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/capillariesio/capillaries/pkg/eval"
	"github.com/capillariesio/capillaries/pkg/proc"
	"github.com/capillariesio/capillaries/pkg/sc"
	"github.com/shopspring/decimal"
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

const script string = `
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
					"field_float1": {
						"expression": "float(r.col_field_int)",
						"type": "float"
					},
					"field_decimal1": {
						"expression": "decimal2(r.col_field_int)",
						"type": "decimal2"
					},
					"field_string1": {
						"expression": "string(r.col_field_int)",
						"type": "string"
					},
					"field_dt1": {
						"expression": "time.Date(2000, time.January, 1, 0, 0, 0, 0, time.FixedZone(\"\", -7200))",
						"type": "datetime"
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
						"expression": "increase_by_ten_percent(increase_by_ten_percent(r.field_int1))",
						"type": "int"
					},
					"taxed_field_float1": {
						"expression": "increase_by_ten_percent(r.field_float1)",
						"type": "float"
					},
					"taxed_field_string1": {
						"expression": "str(increase_by_ten_percent(float(r.field_string1)))",
						"type": "string"
					},
					"taxed_field_decimal1": {
						"expression": "increase_by_ten_percent(r.field_decimal1)",
						"type": "decimal2"
					},
					"taxed_field_bool1": {
						"expression": "bool(r.field_int1)",
						"type": "bool"
					},
					"taxed_field_dt1": {
						"expression": "r.field_dt1",
						"type": "datetime"
					}
				}
			},
			"w": {
				"name": "taxed_table1",
				"having": "w.taxed_field_decimal > 10",
				"fields": {
					"field_int1": {
						"expression": "p.taxed_field_int1",
						"type": "int"
					},
					"field_float1": {
						"expression": "p.taxed_field_float1",
						"type": "float"
					},
					"field_string1": {
						"expression": "p.taxed_field_string1",
						"type": "string"
					},
					"taxed_field_decimal": {
						"expression": "decimal2(p.taxed_field_float1)",
						"type": "decimal2"
					},
					"taxed_field_bool": {
						"expression": "p.taxed_field_bool1",
						"type": "bool"
					},
					"taxed_field_dt": {
						"expression": "p.taxed_field_dt1",
						"type": "datetime"
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

const envSettings string = `
{
	"python_interpreter_path":"/some/bad/python/path",
	"python_interpreter_params":["-u", "-"]
}`

func TestPyCalcDef(t *testing.T) {
	var err error
	var codeBase string

	newScript := &sc.ScriptDef{}
	if err = newScript.Deserialize([]byte(script), &PyCalcTestTestProcessorDefFactory{}, map[string]json.RawMessage{"py_calc": []byte(envSettings)}, "", nil); err != nil {
		t.Error(err)
	}

	// Initializing rowset is tedious and error-prone
	rs := proc.NewRowsetFromFieldRefs(sc.FieldRefs{
		{TableName: "r", FieldName: "field_int1", FieldType: sc.FieldTypeInt},
		{TableName: "r", FieldName: "field_float1", FieldType: sc.FieldTypeFloat},
		{TableName: "r", FieldName: "field_decimal1", FieldType: sc.FieldTypeDecimal2},
		{TableName: "r", FieldName: "field_string1", FieldType: sc.FieldTypeString},
		{TableName: "r", FieldName: "field_bool1", FieldType: sc.FieldTypeBool},
		{TableName: "r", FieldName: "field_dt1", FieldType: sc.FieldTypeDateTime},
	})
	rs.InitRows(1)

	i := int64(235)
	(*rs.Rows[0])[0] = &i
	f := float64(236)
	(*rs.Rows[0])[1] = &f
	d := decimal.NewFromFloat(237)
	(*rs.Rows[0])[2] = &d
	s := "238"
	(*rs.Rows[0])[3] = &s
	b := true
	(*rs.Rows[0])[4] = &b
	dt := time.Date(2001, 2, 2, 0, 0, 0, 0, time.FixedZone("", -7200))
	(*rs.Rows[0])[5] = &dt
	rs.RowCount++

	pyCalcProcDef := newScript.ScriptNodes["tax_table1"].CustomProcessor.(sc.CustomProcessorDef).(*PyCalcProcessorDef)
	codeBase, err = pyCalcProcDef.buildPythonCodebaseFromRowset(rs)
	assert.Equal(t, nil, err)
	assert.Contains(t, codeBase, "r_field_int1 = 235")
	assert.Contains(t, codeBase, "r_field_float1 = 236.000000")
	assert.Contains(t, codeBase, "r_field_decimal1 = 237")
	assert.Contains(t, codeBase, "r_field_string1 = '238'")
	assert.Contains(t, codeBase, "r_field_bool1 = TRUE")
	assert.Contains(t, codeBase, "r_field_dt1 = \"2001-02-02T00:00:00.000-02:00\"") // Capillaries official PythonDatetimeFormat
	assert.Contains(t, codeBase, "p_taxed_field_int1 = increase_by_ten_percent(increase_by_ten_percent(r_field_int1))")
	assert.Contains(t, codeBase, "p_taxed_field_float1 = increase_by_ten_percent(r_field_float1)")
	assert.Contains(t, codeBase, "p_taxed_field_decimal1 = increase_by_ten_percent(r_field_decimal1)")
	assert.Contains(t, codeBase, "p_taxed_field_string1 = str(increase_by_ten_percent(float(r_field_string1)))")
	assert.Contains(t, codeBase, "p_taxed_field_bool1 = bool(r_field_int1)")
	assert.Contains(t, codeBase, "p_taxed_field_dt1 = r_field_dt1")

	// Interpreter errors

	_, err = pyCalcProcDef.analyseExecError(codeBase, "", "", fmt.Errorf("file not found"))
	assert.Equal(t, "interpreter binary not found: /some/bad/python/path", err.Error())

	_, err = pyCalcProcDef.analyseExecError(codeBase, "", "rawErrors", fmt.Errorf("exit status"))
	assert.Equal(t, "interpreter returned an error (probably syntax), see log for details: rawErrors", err.Error())

	_, err = pyCalcProcDef.analyseExecError(codeBase, "", "unknown raw errors", fmt.Errorf("unexpected error"))
	assert.Equal(t, "unexpected calculation errors: unknown raw errors", err.Error())

	// Interpreter ok, see results

	var results []*eval.VarValuesMap
	flushVarsArray := func(varsArray []*eval.VarValuesMap, varsArrayCount int) error {
		results = varsArray
		return nil
	}

	// Some error was caught by Python try/catch, report it

	err = pyCalcProcDef.analyseExecSuccess(codeBase, "", "", pyCalcProcDef.GetFieldRefs(), rs, flushVarsArray)
	assert.Equal(t, "0: unexpected, cannot find calculation start marker --FMINIT:0;", err.Error())

	err = pyCalcProcDef.analyseExecSuccess(codeBase, "--FMINIT:0", "", pyCalcProcDef.GetFieldRefs(), rs, flushVarsArray)
	assert.Equal(t, "0: unexpected, cannot find calculation end marker --FMEND:0;", err.Error())

	err = pyCalcProcDef.analyseExecSuccess(codeBase, "--FMEND:0\n--FMINIT:0", "", pyCalcProcDef.GetFieldRefs(), rs, flushVarsArray)
	assert.Equal(t, "0: unexpected, end marker --FMEND:0(10) is earlier than start marker --FMEND:0(0);", err.Error())

	err = pyCalcProcDef.analyseExecSuccess(codeBase, "--FMINIT:0\n--FMEND:0", "", pyCalcProcDef.GetFieldRefs(), rs, flushVarsArray)
	assert.Equal(t, "0:cannot calculate data points;--FMINIT:0; \nUnexpected error, cannot find error line number in raw error output --FMINIT:0\n", err.Error())

	rawOutput :=
		`
--FMINIT:0
Traceback (most recent call last):
  File "<stdin>", line 1, in <module>  
    s = Something()
NameError: name 'Something' is not defined
--FMEND:0
`
	err = pyCalcProcDef.analyseExecSuccess(codeBase, rawOutput, "", pyCalcProcDef.GetFieldRefs(), rs, flushVarsArray)
	assert.Contains(t, err.Error(), "0:cannot calculate data points;NameError: name 'Something' is not defined; \nSource code lines close to the error location (line 1):\n000001    \n000002    import traceback")

	rawOutput =
		`
--FMINIT:0
Traceback (most recent call last):
  File "some_invalid_file_path", line 1, in <module>  
    s = Something()
NameError: name 'Something' is not defined
--FMEND:0
`
	err = pyCalcProcDef.analyseExecSuccess(codeBase, rawOutput, "", pyCalcProcDef.GetFieldRefs(), rs, flushVarsArray)
	assert.Contains(t, err.Error(), "0:cannot calculate data points;NameError: name 'Something' is not defined; \nUnexpected error, cannot find error line number in raw error output")

	// No error from Python try/catch, get the results

	rawOutput =
		`
--FMINIT:0
--FMOK:0
bla
--FMEND:0
`
	err = pyCalcProcDef.analyseExecSuccess(codeBase, rawOutput, "", pyCalcProcDef.GetFieldRefs(), rs, flushVarsArray)
	assert.Contains(t, err.Error(), "0:unexpected error, cannot deserialize results, invalid character 'b' looking for beginning of value, '\nbla\n'")

	rawOutput =
		`
--FMINIT:0
--FMOK:0
{"taxed_field_float1":2.2,"taxed_field_string1":"aaa","taxed_field_decimal1":3.3,"taxed_field_bool1":true,"taxed_field_int1":1}
--FMEND:0
`
	// {"taxed_field_int1":1,"taxed_field_float1":2.2,"taxed_field_decimal1": 3.3,"taxed_field_string1":"aaa","taxed_field_bool1:true,"taxed_field_dt1":\"2001-02-02T00:00:00.000-02:00\"}

	err = pyCalcProcDef.analyseExecSuccess(codeBase, rawOutput, "", pyCalcProcDef.GetFieldRefs(), rs, flushVarsArray)
	assert.Contains(t, err.Error(), "cannot find result for row 0, field taxed_field_dt1;")

	rawOutput =
		`
--FMINIT:0
--FMOK:0
{"taxed_field_float1":2.2,"taxed_field_string1":"aaa","taxed_field_decimal1":3.3,"taxed_field_bool1":true,"taxed_field_int1":1,"taxed_field_dt1":"2003-02-02T00:00:00.000-02:00"}
--FMEND:0
`
	err = pyCalcProcDef.analyseExecSuccess(codeBase, rawOutput, "", pyCalcProcDef.GetFieldRefs(), rs, flushVarsArray)
	assert.Equal(t, nil, err)
	flushedRow := *results[0]
	// Reader fields must be present in the result
	assert.Equal(t, int64(235), flushedRow["r"]["field_int1"])
	assert.Equal(t, float64(236.0), flushedRow["r"]["field_float1"])
	assert.Equal(t, decimal.NewFromFloat(237), flushedRow["r"]["field_decimal1"])
	assert.Equal(t, "238", flushedRow["r"]["field_string1"])
	assert.Equal(t, true, flushedRow["r"]["field_bool1"])
	assert.Equal(t, time.Date(2001, 2, 2, 0, 0, 0, 0, time.FixedZone("", -7200)), flushedRow["r"]["field_dt1"])
	// p field must be there too
	assert.Equal(t, int64(1), flushedRow["p"]["taxed_field_int1"])
	assert.Equal(t, 2.2, flushedRow["p"]["taxed_field_float1"])
	assert.True(t, decimal.NewFromFloat(3.3).Equal(flushedRow["p"]["taxed_field_decimal1"].(decimal.Decimal)))
	assert.Equal(t, "aaa", flushedRow["p"]["taxed_field_string1"])
	assert.Equal(t, true, flushedRow["p"]["taxed_field_bool1"])
	assert.Equal(t, time.Date(2003, 2, 2, 0, 0, 0, 0, time.FixedZone("", -7200)), flushedRow["p"]["taxed_field_dt1"])

	// Bad script

	err = newScript.Deserialize(
		[]byte(strings.Replace(script, `"having": "w.taxed_field_decimal > 10"`, `"having": "p.taxed_field_int1 > 10"`, 1)),
		&PyCalcTestTestProcessorDefFactory{}, map[string]json.RawMessage{"py_calc": []byte(envSettings)}, "", nil)
	assert.Contains(t, err.Error(), "prohibited field p.taxed_field_int1")

	err = newScript.Deserialize(
		[]byte(strings.Replace(script, `increase_by_ten_percent(r.field_int1)`, `bad_func(r.field_int1)`, 1)),
		&PyCalcTestTestProcessorDefFactory{}, map[string]json.RawMessage{"py_calc": []byte(envSettings)}, "", nil)
	assert.Contains(t, err.Error(), "function def 'bad_func(arg)' not found in Python file")
}
