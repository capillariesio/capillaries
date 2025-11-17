package sc

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/capillariesio/capillaries/pkg/eval"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

const plainScriptJson string = `
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
				"group": true,
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
		},
		"distinct_table1": {
			"type": "distinct_table",
			"rerun_policy": "fail",
			"r": {
				"table": "table1",
				"expected_batches_total": 100
			},
			"w": {
				"name": "distinct_table1",
				"fields": {
					"field_int1": {
						"expression": "r.field_int1",
						"type": "int"
					},
					"field_string1": {
						"expression": "r.field_string1",
						"type": "string"
					}
				},
				"indexes": {
					"idx_distinct_table1_field_int1": "unique(field_int1)"
				}
			}
		},
		"file_totals": {
			"type": "table_file",
			"r": {
				"table": "joined_table1_table2"
			},
			"w": {
				"top": {
					"order": "field_int1(asc),item_count(asc)",
					"limit": 5000000
				},
				"having": "w.total_value > 3",
				"url_template": "file_totals.csv",
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
							"header": "total_value",
							"format": "%s"
						},
						"name": "total_value",
						"expression": "decimal2(r.total_value)",
						"type": "decimal2"
					},
					{
						"csv":{
							"header": "item_count",
							"format": "%d"
						},
						"name": "item_count",
						"expression": "r.item_count",
						"type": "int"
					}
				]
			}
		}
	},
	"dependency_policies": {
		"current_active_first_stopped_nogo":` + DefaultPolicyCheckerConfJson +
	`		
	}
}`

const trickyAffectedScriptJson string = `
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
				},
				"indexes": {
					"idx_table1_string1": "unique(field_string1)"
				}
			}
		},
		"distinct_table1": {
			"type": "distinct_table",
			"rerun_policy": "fail",
			"start_policy": "manual",
			"r": {
				"table": "table1",
				"expected_batches_total": 100
			},
			"w": {
				"name": "distinct_table1",
				"fields": {
					"field_int1": {
						"expression": "r.field_int1",
						"type": "int"
					},
					"field_string1": {
						"expression": "r.field_string1",
						"type": "string"
					}
				},
				"indexes": {
					"idx_distinct_table1_field_int1": "unique(field_int1)"
				}
			}
		},
		"join_table1_table1": {
			"type": "table_lookup_table",
			"start_policy": "auto",
			"r": {
				"table": "distinct_table1",
				"expected_batches_total": 2
			},
			"l": {
				"index_name": "idx_table1_string1",
				"join_on": "r.field_string1",
				"filter": "l.field_int1 > 100",
				"group": true,
				"join_type": "left"
			},
			"w": {
				"name": "joined_table1_table1",
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
						"expression": "sum(l.field_int1)",
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
		"current_active_first_stopped_nogo":` + DefaultPolicyCheckerConfJson +
	`		
	}
}`

const affectedPortfolioScriptJson string = `
{
    "nodes": {
        "1_read_accounts": {
            "type": "file_table",
            "desc": "Load accounts from csv",
            "r": {
                "urls": ["aaa"],
                "csv": {
                    "hdr_line_idx": 0,
                    "first_data_line_idx": 1
                },
                "columns": {
                    "col_account_id": {
                        "csv": {
                            "col_hdr": "account_id"
                        },
                        "col_type": "string"
                    },
                    "col_earliest_period_start": {
                        "csv": {
                            "col_hdr": "earliest_period_start"
                        },
                        "col_type": "string"
                    }
                }
            },
            "w": {
                "name": "accounts",
                "fields": {
                    "account_id": {
                        "expression": "r.col_account_id",
                        "type": "string"
                    },
                    "earliest_period_start": {
                        "expression": "r.col_earliest_period_start",
                        "type": "string"
                    }
                }
            }
        },
        "1_read_txns": {
            "type": "file_table",
            "desc": "Load txns from csv",
            "r": {
                "urls": ["aaa"],
                "csv": {
                    "hdr_line_idx": 0,
                    "first_data_line_idx": 1
                },
                "columns": {
                    "col_ts": {
                        "csv": {
                            "col_hdr": "ts"
                        },
                        "col_type": "string"
                    },
                    "col_account_id": {
                        "csv": {
                            "col_hdr": "account_id"
                        },
                        "col_type": "string"
                    },
                    "col_ticker": {
                        "csv": {
                            "col_hdr": "ticker"
                        },
                        "col_type": "string"
                    },
                    "col_qty": {
                        "csv": {
                            "col_hdr": "qty",
                            "col_format": "%d"
                        },
                        "col_type": "int"
                    },
                    "col_price": {
                        "csv": {
                            "col_hdr": "price",
                            "col_format": "%f"
                        },
                        "col_type": "float"
                    }
                }
            },
            "w": {
                "name": "txns",
                "fields": {
                    "account_id": {
                        "expression": "r.col_account_id",
                        "type": "string"
                    },
                    "ts": {
                        "expression": "r.col_ts",
                        "type": "string"
                    },
                    "txn_json": {
                        "expression": "r.col_account_id",
                        "type": "string"
                    }
                },
                "indexes": {
                    "idx_txns_account_id": "non_unique(account_id)"
                }
            }
        },
        "2_account_txns_outer": {
            "type": "table_lookup_table",
            "desc": "For each account, merge all txns into single json string",
            "start_policy": "manual",
            "r": {
                "table": "accounts",
                "expected_batches_total": 10
            },
            "l": {
                "index_name": "idx_txns_account_id",
                "join_on": "r.account_id",
                "group": true,
                "join_type": "left"
            },
            "w": {
                "name": "account_txns",
                "fields": {
                    "account_id": {
                        "expression": "r.account_id",
                        "type": "string"
                    },
                    "txns_json": {
                        "expression": "string_agg(l.txn_json,\",\")",
                        "type": "string"
                    }
                }
            }
        },
        "1_read_period_holdings": {
            "type": "file_table",
            "desc": "Load holdings from csv",
            "r": {
                "urls": ["aaa"],
                "csv": {
                    "hdr_line_idx": 0,
                    "first_data_line_idx": 1
                },
                "columns": {
                    "col_eod": {
                        "csv": {
                            "col_hdr": "d"
                        },
                        "col_type": "string"
                    },
                    "col_account_id": {
                        "csv": {
                            "col_hdr": "account_id"
                        },
                        "col_type": "string"
                    },
                    "col_ticker": {
                        "csv": {
                            "col_hdr": "ticker"
                        },
                        "col_type": "string"
                    },
                    "col_qty": {
                        "csv": {
                            "col_hdr": "qty",
                            "col_format": "%d"
                        },
                        "col_type": "int"
                    }
                }
            },
            "w": {
                "name": "period_holdings",
                "fields": {
                    "account_id": {
                        "expression": "r.col_account_id",
                        "type": "string"
                    },
                    "eod": {
                        "expression": "r.col_eod",
                        "type": "string"
                    },
                    "holding_json": {
                        "expression": "r.col_account_id",
                        "type": "string"
                    }
                },
                "indexes": {
                    "idx_period_holdings_account_id": "non_unique(account_id)"
                }
            }
        },
        "2_account_period_holdings_outer": {
            "type": "table_lookup_table",
            "desc": "For each account, merge all holdings into single json string",
            "start_policy": "manual",
            "r": {
                "table": "accounts",
                "expected_batches_total": 10
            },
            "l": {
                "index_name": "idx_period_holdings_account_id",
                "join_on": "r.account_id",
                "group": true,
                "join_type": "left"
            },
            "w": {
                "name": "account_period_holdings",
                "fields": {
                    "account_id": {
                        "expression": "r.account_id",
                        "type": "string"
                    },
                    "holdings_json": {
                        "expression": "string_agg(l.holding_json,\",\")",
                        "type": "string"
                    }
                },
                "indexes": {
                    "idx_account_period_holdings_account_id": "unique(account_id)"
                }
            }
        },
        "3_build_account_period_activity": {
            "type": "table_lookup_table",
            "desc": "For each account, merge holdings and txns",
            "start_policy": "manual",
            "r": {
                "table": "account_txns",
                "expected_batches_total": 10
            },
            "l": {
                "index_name": "idx_account_period_holdings_account_id",
                "join_on": "r.account_id",
                "group": false,
                "join_type": "left"
            },
            "w": {
                "name": "account_period_activity",
                "fields": {
                    "account_id": {
                        "expression": "r.account_id",
                        "type": "string"
                    },
                    "txns_json": {
                        "expression": " \"[\" + r.txns_json + \"]\" ",
                        "type": "string"
                    },
                    "holdings_json": {
                        "expression": " \"[\" + l.holdings_json + \"]\" ",
                        "type": "string"
                    }
                }
            }
        },
        "4_calc_account_period_perf": {
            "type": "table_table",
            "r": {
                "table": "account_period_activity",
                "expected_batches_total": 10
            },
            "w": {
                "name": "account_period_perf",
                "fields": {
                    "account_id": {
                        "expression": "r.account_id",
                        "type": "string"
                    },
                    "perf_json": {
                        "expression": "r.txns_json",
                        "type": "string"
                    }
                }
            }
        },
        "5_tag_by_period": {
            "type": "table_table",
            "start_policy": "manual",
            "r": {
                "table": "account_period_perf",
                "expected_batches_total": 10
            },
            "w": {
                "name": "account_period_perf_by_period",
                "fields": {
                    "period": {
                        "expression": "\"aaa\"",
                        "type": "string"
                    },
                    "account_id": {
                        "expression": "r.account_id",
                        "type": "string"
                    },
                    "perf_json": {
                        "expression": "r.perf_json",
                        "type": "string"
                    }
                }
            }
        },
        "5_tag_by_sector": {
            "type": "table_table",
            "r": {
                "table": "account_period_perf_by_period",
                "expected_batches_total": 10
            },
            "w": {
                "name": "account_period_perf_by_period_sector",
                "fields": {
                    "period": {
                        "expression": "r.period",
                        "type": "string"
                    },
                    "sector": {
                        "expression": "\"bbb\"",
                        "type": "string"
                    },
                    "account_id": {
                        "expression": "r.account_id",
                        "type": "string"
                    },
                    "perf_json": {
                        "expression": "r.perf_json",
                        "type": "string"
                    }
                }
            }
        }
	},
	"dependency_policies": {
		"current_active_first_stopped_nogo":` + DefaultPolicyCheckerConfJson +
	`		
	}
}`

func jsonToYamlToScriptDef(t *testing.T) *ScriptDef {
	var jsonDeserializedAsMap map[string]any
	err := json.Unmarshal([]byte(plainScriptJson), &jsonDeserializedAsMap)
	assert.Nil(t, err)

	scriptYamlBytes, err := yaml.Marshal(jsonDeserializedAsMap)
	assert.Nil(t, err)

	scriptDef := &ScriptDef{}
	err = scriptDef.Deserialize(scriptYamlBytes, ScriptYaml, nil, nil, "", nil)
	assert.Nil(t, err)

	return scriptDef
}

func testCreatorFieldRefs(t *testing.T, scriptDef *ScriptDef) {
	tableFieldRefs := scriptDef.ScriptNodes["read_table2"].TableCreator.GetFieldRefsWithAlias(CreatorAlias)
	var tableFieldRef *FieldRef
	tableFieldRef, _ = tableFieldRefs.FindByFieldName("field_int2")
	assert.Equal(t, CreatorAlias, tableFieldRef.TableName)
	assert.Equal(t, FieldTypeInt, tableFieldRef.FieldType)

	fileFieldRefs := scriptDef.ScriptNodes["file_totals"].FileCreator.getFieldRefs()
	var fileFieldRef *FieldRef
	fileFieldRef, _ = fileFieldRefs.FindByFieldName("total_value")
	assert.Equal(t, CreatorAlias, fileFieldRef.TableName)
	assert.Equal(t, FieldTypeDecimal2, fileFieldRef.FieldType)

	// Duplicate creator

	err := scriptDef.Deserialize(
		[]byte(strings.Replace(plainScriptJson, `"name": "table2"`, `"name": "table1"`, 1)), ScriptJson,
		nil, nil, "", nil)
	assert.Contains(t, err.Error(), "duplicate table name: table1")

	// Bad readertable name

	err = scriptDef.Deserialize(
		[]byte(strings.Replace(plainScriptJson, `"table": "table1"`, `"table": "bad_table_name"`, 1)), ScriptJson,
		nil, nil, "", nil)
	assert.Contains(t, err.Error(), "cannot find the node that creates table [bad_table_name]")
}

func TestCreatorFieldRefsJson(t *testing.T) {
	scriptDef := &ScriptDef{}
	assert.Nil(t, scriptDef.Deserialize([]byte(plainScriptJson), ScriptJson, nil, nil, "", nil))

	testCreatorFieldRefs(t, scriptDef)
}

func TestCreatorFieldRefsYaml(t *testing.T) {
	scriptDef := jsonToYamlToScriptDef(t)
	testCreatorFieldRefs(t, scriptDef)
}

func testCreatorCalculateHaving(t *testing.T, scriptDef *ScriptDef) {
	var isHaving bool
	// Table writer: calculate having
	var tableRecord map[string]any
	tableCreator := scriptDef.ScriptNodes["join_table1_table2"].TableCreator

	tableRecord = map[string]any{"total_value": 3}
	isHaving, _ = tableCreator.CheckTableRecordHavingCondition(tableRecord)
	assert.True(t, isHaving)

	tableRecord = map[string]any{"total_value": 2}
	isHaving, _ = tableCreator.CheckTableRecordHavingCondition(tableRecord)
	assert.False(t, isHaving)

	// File writer: calculate having
	var colVals []any
	fileCreator := scriptDef.ScriptNodes["file_totals"].FileCreator

	colVals = make([]any, 0)
	colVals = append(colVals, 0, "a", 4, 0)
	isHaving, _ = fileCreator.CheckFileRecordHavingCondition(colVals)
	assert.True(t, isHaving)

	colVals = make([]any, 0)
	colVals = append(colVals, 0, "a", 3, 0)
	isHaving, _ = fileCreator.CheckFileRecordHavingCondition(colVals)
	assert.False(t, isHaving)
}

func TestCreatorCalculateHavingJson(t *testing.T) {
	scriptDef := &ScriptDef{}
	assert.Nil(t, scriptDef.Deserialize([]byte(plainScriptJson), ScriptJson, nil, nil, "", nil))
	testCreatorCalculateHaving(t, scriptDef)
}

func TestCreatorCalculateHavingYaml(t *testing.T) {
	scriptDef := jsonToYamlToScriptDef(t)
	testCreatorCalculateHaving(t, scriptDef)
}

func testCreatorCalculateOutput(t *testing.T, scriptDef *ScriptDef) {
	var err error
	var vars eval.VarValuesMap

	// Table creator: calculate fields

	var fields map[string]any
	vars = eval.VarValuesMap{"r": {"field_int1": int64(1), "field_string1": "a"}}
	fields, _ = scriptDef.ScriptNodes["join_table1_table2"].TableCreator.CalculateTableRecordFromSrcVars(true, vars)
	if len(fields) == 4 {
		assert.Equal(t, int64(1), fields["field_int1"])
		assert.Equal(t, "a", fields["field_string1"])
		assert.Equal(t, int64(1), fields["total_value"])
		assert.Equal(t, int64(1), fields["item_count"])
	}

	// Table creator: bad field expression, tweak sum

	err = scriptDef.Deserialize(
		[]byte(strings.Replace(plainScriptJson, `sum(l.field_int2)`, `sum(l.field_int2`, 1)), ScriptJson,
		nil, nil, "", nil)
	assert.Contains(t, err.Error(), "cannot parse field expression [sum(l.field_int2]")

	// File creator: calculate columns

	var cols []any
	vars = eval.VarValuesMap{"r": {"field_int1": int64(1), "field_string1": "a", "total_value": decimal.NewFromInt(1), "item_count": int64(1)}}
	cols, _ = scriptDef.ScriptNodes["file_totals"].FileCreator.CalculateFileRecordFromSrcVars(vars)
	assert.Equal(t, 4, len(cols))
	if len(cols) == 4 {
		assert.Equal(t, int64(1), cols[0])
		assert.Equal(t, "a", cols[1])
		assert.Equal(t, decimal.NewFromInt(1), cols[2])
		assert.Equal(t, int64(1), cols[3])
	}

	// File creator: bad column expression, tweak decimal2()

	err = scriptDef.Deserialize(
		[]byte(strings.Replace(plainScriptJson, `decimal2(r.total_value)`, `decimal2(r.total_value`, 1)), ScriptJson,
		nil, nil, "", nil)
	assert.Contains(t, err.Error(), "[cannot parse column expression [decimal2(r.total_value]")
}

func TestCreatorCalculateOutputJson(t *testing.T) {
	scriptDef := &ScriptDef{}
	assert.Nil(t, scriptDef.Deserialize([]byte(plainScriptJson), ScriptJson, nil, nil, "", nil))
	testCreatorCalculateOutput(t, scriptDef)
}

func TestCreatorCalculateOutputYaml(t *testing.T) {
	scriptDef := jsonToYamlToScriptDef(t)
	testCreatorCalculateOutput(t, scriptDef)
}

func testLookup(t *testing.T, scriptDef *ScriptDef) {
	var err error
	var vars eval.VarValuesMap
	var isMatch bool

	// Invalid (writer) field in aggregate, tweak sum() arg

	err = scriptDef.Deserialize(
		[]byte(strings.Replace(plainScriptJson, `"expression": "sum(l.field_int2)"`, `"expression": "sum(w.field_int1)"`, 1)), ScriptJson,
		nil, nil, "", nil)
	assert.Contains(t, err.Error(), "invalid field(s) in target table field expression: [prohibited field w.field_int1]")

	// Filter calculation

	vars = eval.VarValuesMap{"l": {"field_int2": 101}}
	isMatch, _ = scriptDef.ScriptNodes["join_table1_table2"].Lookup.CheckFilterCondition(vars)
	assert.True(t, isMatch)

	vars = eval.VarValuesMap{"l": {"field_int2": 100}}
	isMatch, _ = scriptDef.ScriptNodes["join_table1_table2"].Lookup.CheckFilterCondition(vars)
	assert.False(t, isMatch)

	// bad index_name, tweak it

	err = scriptDef.Deserialize(
		[]byte(strings.Replace(plainScriptJson, `"index_name": "idx_table2_string2"`, `"index_name": "idx_table2_string2_bad"`, 1)), ScriptJson,
		nil, nil, "", nil)
	assert.Contains(t, err.Error(), "cannot find the node that creates index [idx_table2_string2_bad]")

	// bad join_on, tweak it

	err = scriptDef.Deserialize(
		[]byte(strings.Replace(plainScriptJson, `"join_on": "r.field_string1"`, `"join_on": ""`, 1)), ScriptJson,
		nil, nil, "", nil)
	assert.Contains(t, err.Error(), "expected a comma-separated list of <table_name>.<field_name>, got []")

	err = scriptDef.Deserialize(
		[]byte(strings.Replace(plainScriptJson, `"join_on": "r.field_string1"`, `"join_on": "bla"`, 1)), ScriptJson,
		nil, nil, "", nil)
	assert.Contains(t, err.Error(), "expected a comma-separated list of <table_name>.<field_name>, got [bla]")

	err = scriptDef.Deserialize(
		[]byte(strings.Replace(plainScriptJson, `"join_on": "r.field_string1"`, `"join_on": "bla.bla"`, 1)), ScriptJson,
		nil, nil, "", nil)
	assert.Contains(t, err.Error(), "source table name [bla] unknown, expected [r]")

	err = scriptDef.Deserialize(
		[]byte(strings.Replace(plainScriptJson, `"join_on": "r.field_string1"`, `"join_on": "r.field_string1_bad"`, 1)), ScriptJson,
		nil, nil, "", nil)
	assert.Contains(t, err.Error(), "source [r] does not produce field [field_string1_bad]")

	err = scriptDef.Deserialize(
		[]byte(strings.Replace(plainScriptJson, `"join_on": "r.field_string1"`, `"join_on": "r.field_int1"`, 1)), ScriptJson,
		nil, nil, "", nil)
	assert.Contains(t, err.Error(), "left-side field field_int1 has type int, while index field field_string2 has type string")

	// bad filter, tweak it

	err = scriptDef.Deserialize(
		[]byte(strings.Replace(plainScriptJson, `"filter": "l.field_int2 > 100"`, `"filter": "r.field_int2 > 100"`, 1)), ScriptJson,
		nil, nil, "", nil)
	assert.Contains(t, err.Error(), "invalid field in lookup filter [r.field_int2 > 100], only fields from the lookup table [table2](alias l) are allowed: [unknown field r.field_int2]")

	// bad join_type, tweak it

	err = scriptDef.Deserialize(
		[]byte(strings.Replace(plainScriptJson, `"join_type": "left"`, `"join_type": "left_bad"`, 1)), ScriptJson,
		nil, nil, "", nil)
	assert.Contains(t, err.Error(), "invalid join type, expected inner or left, left_bad is not supported")
}

func TestLookupJson(t *testing.T) {
	scriptDef := &ScriptDef{}
	assert.Nil(t, scriptDef.Deserialize([]byte(plainScriptJson), ScriptJson, nil, nil, "", nil))
	testLookup(t, scriptDef)
}

func TestLookupYaml(t *testing.T) {
	scriptDef := jsonToYamlToScriptDef(t)
	testLookup(t, scriptDef)
}

func testBadCreatorHaving(t *testing.T, scriptDef *ScriptDef) {
	// Bad expression, tweak having expression

	err := scriptDef.Deserialize(
		[]byte(strings.Replace(plainScriptJson, `"having": "w.total_value > 2"`, `"having": "w.total_value &> 2"`, 1)), ScriptJson,
		nil, nil, "", nil)
	assert.Contains(t, err.Error(), "cannot parse table creator 'having' condition [w.total_value &> 2]")

	err = scriptDef.Deserialize(
		[]byte(strings.Replace(plainScriptJson, `"having": "w.total_value > 3"`, `"having": "w.bad_field &> 3"`, 1)), ScriptJson,
		nil, nil, "", nil)
	assert.Contains(t, err.Error(), "cannot parse file creator 'having' condition [w.bad_field &> 3]")

	// Unknown field in having

	err = scriptDef.Deserialize(
		[]byte(strings.Replace(plainScriptJson, `"having": "w.total_value > 2"`, `"having": "w.bad_field > 2"`, 1)), ScriptJson,
		nil, nil, "", nil)
	assert.Contains(t, err.Error(), "invalid field in table creator 'having' condition: [unknown field w.bad_field]")

	err = scriptDef.Deserialize(
		[]byte(strings.Replace(plainScriptJson, `"having": "w.total_value > 3"`, `"having": "w.bad_field > 3"`, 1)), ScriptJson,
		nil, nil, "", nil)
	assert.Contains(t, err.Error(), "invalid field in file creator 'having' condition: [unknown field w.bad_field]]")

	// Prohibited reader field in having

	err = scriptDef.Deserialize(
		[]byte(strings.Replace(plainScriptJson, `"having": "w.total_value > 2"`, `"having": "r.field_int1 > 2"`, 1)), ScriptJson,
		nil, nil, "", nil)
	assert.Contains(t, err.Error(), "invalid field in table creator 'having' condition: [prohibited field r.field_int1]")

	err = scriptDef.Deserialize(
		[]byte(strings.Replace(plainScriptJson, `"having": "w.total_value > 3"`, `"having": "r.field_int1 > 3"`, 1)), ScriptJson,
		nil, nil, "", nil)
	assert.Contains(t, err.Error(), "invalid field in file creator 'having' condition: [prohibited field r.field_int1]")

	// Prohibited lookup field in table creator having

	err = scriptDef.Deserialize(
		[]byte(strings.Replace(plainScriptJson, `"having": "w.total_value > 2"`, `"having": "l.field_int2 > 2"`, 1)), ScriptJson,
		nil, nil, "", nil)
	assert.Contains(t, err.Error(), "invalid field in table creator 'having' condition: [prohibited field l.field_int2]")

	// Type mismatch in having

	err = scriptDef.Deserialize(
		[]byte(strings.Replace(plainScriptJson, `"having": "w.total_value > 2"`, `"having": "w.total_value == true"`, 1)), ScriptJson,
		nil, nil, "", nil)
	assert.Contains(t, err.Error(), "cannot evaluate table creator 'having' expression [w.total_value == true]: [cannot perform binary comp op, incompatible arg types '0(int64)' == 'true(bool)' ]")

	err = scriptDef.Deserialize(
		[]byte(strings.Replace(plainScriptJson, `"having": "w.total_value > 3"`, `"having": "w.total_value == true"`, 1)), ScriptJson,
		nil, nil, "", nil)
	assert.Contains(t, err.Error(), "cannot evaluate file creator 'having' expression [w.total_value == true]: [cannot perform binary comp op, incompatible arg types '0(decimal.Decimal)' == 'true(bool)' ]")
}

func TestBadCreatorHavingJson(t *testing.T) {
	scriptDef := &ScriptDef{}
	assert.Nil(t, scriptDef.Deserialize([]byte(plainScriptJson), ScriptJson, nil, nil, "", nil))
	testBadCreatorHaving(t, scriptDef)
}

func TestBadCreatorHavingYaml(t *testing.T) {
	scriptDef := jsonToYamlToScriptDef(t)
	testBadCreatorHaving(t, scriptDef)
}

func testTopLimit(t *testing.T, scriptDef *ScriptDef) {
	assert.Equal(t, "w", scriptDef.ScriptNodes["file_totals"].GetTargetName())

	// Tweak limit beyond allowed maximum

	err := scriptDef.Deserialize(
		[]byte(strings.Replace(plainScriptJson, `"limit": 5000000`, `"limit": 5000001`, 1)), ScriptJson,
		nil, nil, "", nil)
	assert.Contains(t, err.Error(), "top.limit cannot exceed 5000000")

	// Remove limit altogether

	assert.Nil(t, scriptDef.Deserialize(
		[]byte(strings.Replace(plainScriptJson, `"limit": 5000000`, `"some_bogus_setting": 5000000`, 1)), ScriptJson,
		nil, nil, "", nil))
	assert.Equal(t, 5000000, scriptDef.ScriptNodes["file_totals"].FileCreator.Top.Limit)
}

func TestTopLimitJson(t *testing.T) {
	scriptDef := &ScriptDef{}
	assert.Nil(t, scriptDef.Deserialize([]byte(plainScriptJson), ScriptJson, nil, nil, "", nil))
	testTopLimit(t, scriptDef)
}

func TestTopLimitYaml(t *testing.T) {
	scriptDef := jsonToYamlToScriptDef(t)
	testTopLimit(t, scriptDef)
}

func testBatchIntervalsCalculation(t *testing.T, scriptDef *ScriptDef) {
	var intervals [][]int64

	tableReaderNodeDef := scriptDef.ScriptNodes["join_table1_table2"]
	intervals, _ = tableReaderNodeDef.GetTokenIntervalsByNumberOfBatches()

	assert.Equal(t, 2, len(intervals))
	if len(intervals) == 2 {
		assert.Equal(t, int64(-9223372036854775808), intervals[0][0])
		assert.Equal(t, int64(-2), intervals[0][1])
		assert.Equal(t, int64(-1), intervals[1][0])
		assert.Equal(t, int64(9223372036854775807), intervals[1][1])
	}

	fileReaderNodeDef := scriptDef.ScriptNodes["read_table1"]
	intervals, _ = fileReaderNodeDef.GetTokenIntervalsByNumberOfBatches()

	assert.Equal(t, 1, len(intervals))
	if len(intervals) == 1 {
		assert.Equal(t, int64(0), intervals[0][0])
		assert.Equal(t, int64(0), intervals[0][1])
	}

	fileCreatorNodeDef := scriptDef.ScriptNodes["file_totals"]
	intervals, _ = fileCreatorNodeDef.GetTokenIntervalsByNumberOfBatches()

	assert.Equal(t, 1, len(intervals))
	if len(intervals) == 1 {
		assert.Equal(t, int64(-9223372036854775808), intervals[0][0])
		assert.Equal(t, int64(9223372036854775807), intervals[0][1])
	}
}

func TestBatchIntervalsCalculationJson(t *testing.T) {
	scriptDef := &ScriptDef{}
	assert.Nil(t, scriptDef.Deserialize([]byte(plainScriptJson), ScriptJson, nil, nil, "", nil))
	testBatchIntervalsCalculation(t, scriptDef)
}

func TestBatchIntervalsCalculationYaml(t *testing.T) {
	scriptDef := jsonToYamlToScriptDef(t)
	testBatchIntervalsCalculation(t, scriptDef)
}

func testUniqueIndexesFieldRefs(t *testing.T, scriptDef *ScriptDef) {
	fileReaderNodeDef := scriptDef.ScriptNodes["read_table2"]
	fieldRefs := fileReaderNodeDef.GetUniqueIndexesFieldRefs()
	assert.Equal(t, 1, len(*fieldRefs))
	if len(*fieldRefs) == 1 {
		assert.Equal(t, "table2", (*fieldRefs)[0].TableName)
		assert.Equal(t, "field_string2", (*fieldRefs)[0].FieldName)
		assert.Equal(t, FieldTypeString, (*fieldRefs)[0].FieldType)
	}
}

func TestUniqueIndexesFieldRefsJson(t *testing.T) {
	scriptDef := &ScriptDef{}
	assert.Nil(t, scriptDef.Deserialize([]byte(plainScriptJson), ScriptJson, nil, nil, "", nil))
	testUniqueIndexesFieldRefs(t, scriptDef)
}

func TestUniqueIndexesFieldRefsYaml(t *testing.T) {
	scriptDef := jsonToYamlToScriptDef(t)
	testUniqueIndexesFieldRefs(t, scriptDef)
}

func testAffectedNodes(t *testing.T, scriptDef *ScriptDef) {
	affectedNodes := scriptDef.GetAffectedNodes([]string{"read_table1"})
	assert.Equal(t, 4, len(affectedNodes))
	assert.Contains(t, affectedNodes, "read_table1")
	assert.Contains(t, affectedNodes, "join_table1_table2")
	assert.Contains(t, affectedNodes, "file_totals")

	affectedNodes = scriptDef.GetAffectedNodes([]string{"read_table1", "read_table2"})
	assert.Equal(t, 5, len(affectedNodes))
	assert.Contains(t, affectedNodes, "read_table1")
	assert.Contains(t, affectedNodes, "read_table2")
	assert.Contains(t, affectedNodes, "join_table1_table2")
	assert.Contains(t, affectedNodes, "file_totals")

	// Make join manual and see the list of affected nodes shrinking

	assert.Nil(t, scriptDef.Deserialize([]byte(strings.Replace(plainScriptJson, `"start_policy": "auto"`, `"start_policy": "manual"`, 1)), ScriptJson, nil, nil, "", nil))

	affectedNodes = scriptDef.GetAffectedNodes([]string{"read_table1"})
	assert.Equal(t, 2, len(affectedNodes))
	assert.Contains(t, affectedNodes, "read_table1")

	affectedNodes = scriptDef.GetAffectedNodes([]string{"read_table1", "read_table2"})
	assert.Equal(t, 3, len(affectedNodes))
	assert.Contains(t, affectedNodes, "read_table1")
	assert.Contains(t, affectedNodes, "read_table2")
}

func TestAffectedNodesJson(t *testing.T) {
	scriptDef := &ScriptDef{}
	assert.Nil(t, scriptDef.Deserialize([]byte(plainScriptJson), ScriptJson, nil, nil, "", nil))
	testAffectedNodes(t, scriptDef)
}

func TestAffectedNodesYaml(t *testing.T) {
	scriptDef := jsonToYamlToScriptDef(t)
	testAffectedNodes(t, scriptDef)
}

func TestTrickyAffectedNodesJson(t *testing.T) {
	scriptDef := &ScriptDef{}
	assert.Nil(t, scriptDef.Deserialize([]byte(trickyAffectedScriptJson), ScriptJson, nil, nil, "", nil))

	affectedNodes := scriptDef.GetAffectedNodes([]string{"read_table1"})
	assert.Equal(t, 1, len(affectedNodes))
	assert.Contains(t, affectedNodes, "read_table1")

	affectedNodes = scriptDef.GetAffectedNodes([]string{"distinct_table1"})
	assert.Equal(t, 2, len(affectedNodes))
	assert.Contains(t, affectedNodes, "distinct_table1")
	assert.Contains(t, affectedNodes, "join_table1_table1")

	// Make distinct_table1 auto
	assert.Nil(t, scriptDef.Deserialize([]byte(strings.Replace(trickyAffectedScriptJson, `"start_policy": "manual"`, `"start_policy": "auto"`, 1)), ScriptJson, nil, nil, "", nil))

	affectedNodes = scriptDef.GetAffectedNodes([]string{"read_table1"})
	assert.Equal(t, 3, len(affectedNodes))
	assert.Contains(t, affectedNodes, "read_table1")
	assert.Contains(t, affectedNodes, "distinct_table1")
	assert.Contains(t, affectedNodes, "join_table1_table1")

	affectedNodes = scriptDef.GetAffectedNodes([]string{"distinct_table1"})
	assert.Equal(t, 2, len(affectedNodes))
	assert.Contains(t, affectedNodes, "distinct_table1")
	assert.Contains(t, affectedNodes, "join_table1_table1")
}

func TestTrickyAffectedNodesJson2(t *testing.T) {
	scriptDef := &ScriptDef{}
	assert.Nil(t, scriptDef.Deserialize([]byte(trickyAffectedScriptJson), ScriptJson, nil, nil, "", nil))

	affectedNodes := scriptDef.GetAffectedNodes([]string{"read_table1", "distinct_table1"})
	assert.Equal(t, 3, len(affectedNodes))
	assert.Contains(t, affectedNodes, "read_table1")
	assert.Contains(t, affectedNodes, "distinct_table1")
	assert.Contains(t, affectedNodes, "join_table1_table1")
}

func TestAffectedPortfolioScriptJson(t *testing.T) {
	scriptDef := &ScriptDef{}
	assert.Nil(t, scriptDef.Deserialize([]byte(affectedPortfolioScriptJson), ScriptJson, nil, nil, "", nil))

	affectedNodes := scriptDef.GetAffectedNodes([]string{"1_read_accounts", "1_read_txns", "1_read_period_holdings"})
	assert.Equal(t, 3, len(affectedNodes))
	assert.Contains(t, affectedNodes, "1_read_accounts")
	assert.Contains(t, affectedNodes, "1_read_txns")
	assert.Contains(t, affectedNodes, "1_read_period_holdings")

	affectedNodes = scriptDef.GetAffectedNodes([]string{"2_account_txns_outer", "2_account_period_holdings_outer"})
	assert.Equal(t, 2, len(affectedNodes))
	assert.Contains(t, affectedNodes, "2_account_txns_outer")
	assert.Contains(t, affectedNodes, "2_account_period_holdings_outer")

	affectedNodes = scriptDef.GetAffectedNodes([]string{"3_build_account_period_activity"})
	assert.Equal(t, 2, len(affectedNodes))
	assert.Contains(t, affectedNodes, "3_build_account_period_activity")
	assert.Contains(t, affectedNodes, "4_calc_account_period_perf")

	affectedNodes = scriptDef.GetAffectedNodes([]string{"5_tag_by_period"})
	assert.Equal(t, 2, len(affectedNodes))
	assert.Contains(t, affectedNodes, "5_tag_by_period")
	assert.Contains(t, affectedNodes, "5_tag_by_sector")
}

func TestUnusedIndex(t *testing.T) {
	scriptDef := &ScriptDef{}
	err := scriptDef.Deserialize(
		[]byte(strings.Replace(plainScriptJson, `"idx_table2_string2": "unique(field_string2)"`, `"idx_table2_string2": "unique(field_string2)", "idx_bad_extra": "non_unique(field_int2)"`, 1)), ScriptJson,
		nil, nil, "", nil)
	assert.Contains(t, err.Error(), "consider removing this index")
}

func TestDistinct(t *testing.T) {
	scriptDef := &ScriptDef{}
	err := scriptDef.Deserialize(
		[]byte(strings.Replace(plainScriptJson, `"rerun_policy": "fail"`, `"rerun_policy": "rerun"`, 1)), ScriptJson,
		nil, nil, "", nil)
	assert.Contains(t, err.Error(), "distinct_table node must have fail policy")

	scriptDef = &ScriptDef{}
	err = scriptDef.Deserialize(
		[]byte(strings.Replace(plainScriptJson, `"idx_distinct_table1_field_int1": "unique(field_int1)"`, `"idx_bad_non_unique": "non_unique(field_int1)"`, 1)), ScriptJson,
		nil, nil, "", nil)
	assert.Contains(t, err.Error(), "expected exactly one unique idx definition")

	scriptDef = &ScriptDef{}
	err = scriptDef.Deserialize(
		[]byte(strings.Replace(plainScriptJson, `"idx_distinct_table1_field_int1": "unique(field_int1)"`, `"idx_distinct_table1_field_int1": "unique(field_int1)", "idx_bad_extra_unique": "unique(field_string1)"`, 1)), ScriptJson,
		nil, nil, "", nil)
	assert.Contains(t, err.Error(), "expected exactly one unique idx definition")
}
