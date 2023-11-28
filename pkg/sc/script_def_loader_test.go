package sc

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

const parameterizedScriptJson string = `
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
						"csv":{
							"col_idx": 0
						},
						"col_type": "int"
					},
					"col_field_string": {
						"csv":{
							"col_idx": 1
						},
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
        "custom_processor_node": {
            "type": "table_custom_tfm_table",
            "custom_proc_type": "some_test_custom_proc",
            "desc": "",
            "r": {
                "table": "{source_table_for_test_custom_processor}",
				"expected_batches_total": "{number_of_batches_for_test_custom_processor|number}"
            },
			"p": {
				"produced_fields": {
					"produced_field_int1": {
						"expression": "r.field_int1*2",
						"type": "int"
					}
				}
			},
			"w": {
                "name": "processed_table1",
                "fields": {
                    "field_int1": {
                        "expression": "p.produced_field_int1",
                        "type": "int"
                    },
					"field_string1": {
						"expression": "{constant_string_for_test_custom_processor}",
						"type": "string"
					}
                }
            }
        },
		"read_table2": {
			"type": "file_table",
			"r": {
				"urls": [
					"file2.tsv"
				],
				"csv":{
					"first_data_line_idx": 0
				},
				"columns": {
					"col_field_int": {
						"csv":{
							"col_idx": 0
						},
						"col_type": "int"
					},
					"col_field_string": {
						"csv":{
							"col_idx": 1
						},
						"col_type": "string"
					}
				}
			},
			"w": {
				"name": "table2",
				"fields": {
					"field_int2": {
						"expression": "r.col_field_int",
						"type": "int"
					},
					"field_string2": {
						"expression": "r.col_field_string",
						"type": "string"
					}
				},
				"indexes": {
					"idx_table2_string2": "unique(field_string2)"
				}
			}
		},
		"join_table1_table2": {
			"type": "table_lookup_table",
			"start_policy": "auto",
			"r": {
				"table": "table1",
				"expected_batches_total": 2
			},
			"l": {
				"index_name": "idx_table2_string2",
				"join_on": "r.field_string1",
				"filter": "l.field_int2 > 100",
				"group": "{join_table1_table2_group|bool}",
				"join_type": "left"
			},
			"w": {
				"name": "joined_table1_table2",
				"having": "w.total_value > 2",
				"fields": {
					"field_int1": {
						"expression": "r.field_int1",
						"type": "int"
					},
					"field_string1": {
						"expression": "r.field_string1",
						"type": "string"
					},
					"total_value": {
						"expression": "sum(l.field_int2)",
						"type": "int"
					},
					"item_count": {
						"expression": "count()",
						"type": "int"
					}
				}
			}
		}
	},
	"dependency_policies": {
		"current_active_first_stopped_nogo":` + DefaultPolicyCheckerConf +
	`		
	}
}`

const paramsJson string = `
{
	"source_table_for_test_custom_processor": "table1",
	"number_of_batches_for_test_custom_processor": 10,
    "constant_string_for_test_custom_processor": "\\\"aaa\\\n\\\"",
    "join_table1_table2_group": true
}
`

type SomeTestCustomProcessorDef struct {
	ProducedFields                map[string]*WriteTableFieldDef `json:"produced_fields"`
	UsedInTargetExpressionsFields FieldRefs
}

func (procDef *SomeTestCustomProcessorDef) GetFieldRefs() *FieldRefs {
	fieldRefs := make(FieldRefs, len(procDef.ProducedFields))
	i := 0
	for fieldName, fieldDef := range procDef.ProducedFields {
		fieldRefs[i] = FieldRef{
			TableName: CustomProcessorAlias,
			FieldName: fieldName,
			FieldType: fieldDef.Type}
		i += 1
	}
	return &fieldRefs
}

func (procDef *SomeTestCustomProcessorDef) Deserialize(raw json.RawMessage, customProcSettings json.RawMessage, caPath string, privateKeys map[string]string) error {
	var err error
	if err = json.Unmarshal(raw, procDef); err != nil {
		return fmt.Errorf("cannot unmarshal some_test_custom_processor def: %s", err.Error())
	}

	errors := make([]string, 0)

	// Produced fields (same as in PyCalcProcessorDef.Deserialize, we assume that this test processor also uses Python syntax for expressions)
	for _, fieldDef := range procDef.ProducedFields {
		if fieldDef.ParsedExpression, err = ParseRawRelaxedGolangExpressionStringAndHarvestFieldRefs(fieldDef.RawExpression, &fieldDef.UsedFields, FieldRefAllowUnknownIdents); err != nil {
			errors = append(errors, fmt.Sprintf("cannot parse field expression [%s]: [%s]", fieldDef.RawExpression, err.Error()))
		} else if !IsValidFieldType(fieldDef.Type) {
			errors = append(errors, fmt.Sprintf("invalid field type [%s]", fieldDef.Type))
		}
	}

	procDef.UsedInTargetExpressionsFields = GetFieldRefsUsedInAllTargetExpressions(procDef.ProducedFields)
	return nil
}

func (procDef *SomeTestCustomProcessorDef) GetUsedInTargetExpressionsFields() *FieldRefs {
	return &procDef.UsedInTargetExpressionsFields
}

type SomeTestCustomProcessorDefFactory struct {
}

func (f *SomeTestCustomProcessorDefFactory) Create(processorType string) (CustomProcessorDef, bool) {
	switch processorType {
	case "some_test_custom_proc":
		return &SomeTestCustomProcessorDef{}, true
	default:
		return nil, false
	}
}

func TestNewScriptFromFileBytes(t *testing.T) {
	// Test main script parsing function
	scriptDef, err, initProblem := NewScriptFromFileBytes("", nil,
		"someScriptUri", []byte(parameterizedScriptJson),
		"someScriptParamsUrl", []byte(paramsJson),
		&SomeTestCustomProcessorDefFactory{}, map[string]json.RawMessage{"some_test_custom_proc": []byte("{}")})
	assert.Equal(t, nil, err)
	assert.Equal(t, 4, len(scriptDef.ScriptNodes))
	assert.Equal(t, ScriptInitNoProblem, initProblem)

	// Verify template parameters were applied
	assert.Equal(t, "table1", scriptDef.ScriptNodes["custom_processor_node"].TableReader.TableName)
	assert.Equal(t, 10, scriptDef.ScriptNodes["custom_processor_node"].TableReader.ExpectedBatchesTotal)
	assert.Equal(t, "\"aaa\\n\"", scriptDef.ScriptNodes["custom_processor_node"].TableCreator.Fields["field_string1"].RawExpression)
	assert.Equal(t, true, scriptDef.ScriptNodes["join_table1_table2"].Lookup.IsGroup)

	// Tweak paramater name and make sure templating engine catches it
	scriptDef, err, initProblem = NewScriptFromFileBytes("", nil,
		"someScriptUri", []byte(strings.ReplaceAll(parameterizedScriptJson, "source_table_for_test_custom_processor", "some_bad_param")),
		"someScriptParamsUrl", []byte(paramsJson),
		nil, nil)
	assert.Contains(t, err.Error(), "unresolved parameter references", err.Error())

	// Bad-formed JSON
	scriptDef, err, initProblem = NewScriptFromFileBytes("", nil,
		"someScriptUri", []byte(strings.TrimSuffix(parameterizedScriptJson, "}")),
		"someScriptParamsUrl", []byte(paramsJson),
		nil, nil)
	assert.Contains(t, err.Error(), "unexpected end of JSON input", err.Error())

	// Invalid field in custom processor (Python) formula
	scriptDef, err, initProblem = NewScriptFromFileBytes("", nil,
		"someScriptUri", []byte(strings.ReplaceAll(parameterizedScriptJson, "r.field_int1*2", "r.bad_field")),
		"someScriptParamsUrl", []byte(paramsJson),
		&SomeTestCustomProcessorDefFactory{}, map[string]json.RawMessage{"some_test_custom_proc": []byte("{}")})
	assert.Contains(t, err.Error(), "field usage error in custom processor creator")

	// Invalid dependency policy
	scriptDef, err, initProblem = NewScriptFromFileBytes("", nil,
		"someScriptUri", []byte(strings.ReplaceAll(parameterizedScriptJson, "run_is_current(desc),node_start_ts(desc)", "some_bad_event_priority_order")),
		"someScriptParamsUrl", []byte(paramsJson),
		&SomeTestCustomProcessorDefFactory{}, map[string]json.RawMessage{"some_test_custom_proc": []byte("{}")})
	assert.Contains(t, err.Error(), "failed to deserialize dependency policy")

	// Run (tweaked) dependency policy checker with some vanilla values and see if it works
	scriptDef, err, initProblem = NewScriptFromFileBytes("", nil,
		"someScriptUri", []byte(strings.ReplaceAll(parameterizedScriptJson, "e.run_final_status == wfmodel.RunStart", "e.run_final_status == true")),
		"someScriptParamsUrl", []byte(paramsJson),
		&SomeTestCustomProcessorDefFactory{}, map[string]json.RawMessage{"some_test_custom_proc": []byte("{}")})
	assert.Contains(t, err.Error(), "failed to test dependency policy")
}