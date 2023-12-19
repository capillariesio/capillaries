package sc

import (
	"strings"
	"testing"

	"github.com/capillariesio/capillaries/pkg/eval"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
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
		"file_totals": {
			"type": "table_file",
			"r": {
				"table": "joined_table1_table2"
			},
			"w": {
				"top": {
					"order": "field_int1(asc),item_count(asc)",
					"limit": 500000
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
		"current_active_first_stopped_nogo":` + DefaultPolicyCheckerConf +
	`		
	}
}`

func TestCreatorFieldRefs(t *testing.T) {
	var err error

	newScript := &ScriptDef{}
	assert.Nil(t, newScript.Deserialize([]byte(plainScriptJson), nil, nil, "", nil))

	tableFieldRefs := newScript.ScriptNodes["read_table2"].TableCreator.GetFieldRefsWithAlias(CreatorAlias)
	var tableFieldRef *FieldRef
	tableFieldRef, _ = tableFieldRefs.FindByFieldName("field_int2")
	assert.Equal(t, CreatorAlias, tableFieldRef.TableName)
	assert.Equal(t, FieldTypeInt, tableFieldRef.FieldType)

	fileFieldRefs := newScript.ScriptNodes["file_totals"].FileCreator.getFieldRefs()
	var fileFieldRef *FieldRef
	fileFieldRef, _ = fileFieldRefs.FindByFieldName("total_value")
	assert.Equal(t, CreatorAlias, fileFieldRef.TableName)
	assert.Equal(t, FieldTypeDecimal2, fileFieldRef.FieldType)

	// Duplicate creator

	err = newScript.Deserialize(
		[]byte(strings.Replace(plainScriptJson, `"name": "table2"`, `"name": "table1"`, 1)),
		nil, nil, "", nil)
	assert.Contains(t, err.Error(), "duplicate table name: table1")

	// Bad readertable name

	err = newScript.Deserialize(
		[]byte(strings.Replace(plainScriptJson, `"table": "table1"`, `"table": "bad_table_name"`, 1)),
		nil, nil, "", nil)
	assert.Contains(t, err.Error(), "cannot find the node that creates table [bad_table_name]")
}

func TestCreatorCalculateHaving(t *testing.T) {
	var isHaving bool

	newScript := &ScriptDef{}
	assert.Nil(t, newScript.Deserialize([]byte(plainScriptJson), nil, nil, "", nil))

	// Table writer: calculate having
	var tableRecord map[string]any
	tableCreator := newScript.ScriptNodes["join_table1_table2"].TableCreator

	tableRecord = map[string]any{"total_value": 3}
	isHaving, _ = tableCreator.CheckTableRecordHavingCondition(tableRecord)
	assert.True(t, isHaving)

	tableRecord = map[string]any{"total_value": 2}
	isHaving, _ = tableCreator.CheckTableRecordHavingCondition(tableRecord)
	assert.False(t, isHaving)

	// File writer: calculate having
	var colVals []any
	fileCreator := newScript.ScriptNodes["file_totals"].FileCreator

	colVals = make([]any, 0)
	colVals = append(colVals, 0, "a", 4, 0)
	isHaving, _ = fileCreator.CheckFileRecordHavingCondition(colVals)
	assert.True(t, isHaving)

	colVals = make([]any, 0)
	colVals = append(colVals, 0, "a", 3, 0)
	isHaving, _ = fileCreator.CheckFileRecordHavingCondition(colVals)
	assert.False(t, isHaving)
}

func TestCreatorCalculateOutput(t *testing.T) {
	var err error
	var vars eval.VarValuesMap

	newScript := &ScriptDef{}
	assert.Nil(t, newScript.Deserialize([]byte(plainScriptJson), nil, nil, "", nil))

	// Table creator: calculate fields

	var fields map[string]any
	vars = eval.VarValuesMap{"r": {"field_int1": int64(1), "field_string1": "a"}}
	fields, _ = newScript.ScriptNodes["join_table1_table2"].TableCreator.CalculateTableRecordFromSrcVars(true, vars)
	if len(fields) == 4 {
		assert.Equal(t, int64(1), fields["field_int1"])
		assert.Equal(t, "a", fields["field_string1"])
		assert.Equal(t, int64(1), fields["total_value"])
		assert.Equal(t, int64(1), fields["item_count"])
	}

	// Table creator: bad field expression, tweak sum

	err = newScript.Deserialize(
		[]byte(strings.Replace(plainScriptJson, `sum(l.field_int2)`, `sum(l.field_int2`, 1)),
		nil, nil, "", nil)
	assert.Contains(t, err.Error(), "cannot parse field expression [sum(l.field_int2]")

	// File creator: calculate columns

	var cols []any
	vars = eval.VarValuesMap{"r": {"field_int1": int64(1), "field_string1": "a", "total_value": decimal.NewFromInt(1), "item_count": int64(1)}}
	cols, _ = newScript.ScriptNodes["file_totals"].FileCreator.CalculateFileRecordFromSrcVars(vars)
	assert.Equal(t, 4, len(cols))
	if len(cols) == 4 {
		assert.Equal(t, int64(1), cols[0])
		assert.Equal(t, "a", cols[1])
		assert.Equal(t, decimal.NewFromInt(1), cols[2])
		assert.Equal(t, int64(1), cols[3])
	}

	// File creator: bad column expression, tweak decimal2()

	err = newScript.Deserialize(
		[]byte(strings.Replace(plainScriptJson, `decimal2(r.total_value)`, `decimal2(r.total_value`, 1)),
		nil, nil, "", nil)
	assert.Contains(t, err.Error(), "[cannot parse column expression [decimal2(r.total_value]")

}

func TestLookup(t *testing.T) {
	var err error
	var vars eval.VarValuesMap
	var isMatch bool

	newScript := &ScriptDef{}
	assert.Nil(t, newScript.Deserialize([]byte(plainScriptJson), nil, nil, "", nil))

	// Invalid (writer) field in aggregate, tweak sum() arg

	err = newScript.Deserialize(
		[]byte(strings.Replace(plainScriptJson, `"expression": "sum(l.field_int2)"`, `"expression": "sum(w.field_int1)"`, 1)),
		nil, nil, "", nil)
	assert.Contains(t, err.Error(), "invalid field(s) in target table field expression: [prohibited field w.field_int1]")

	// Filter calculation

	vars = eval.VarValuesMap{"l": {"field_int2": 101}}
	isMatch, _ = newScript.ScriptNodes["join_table1_table2"].Lookup.CheckFilterCondition(vars)
	assert.True(t, isMatch)

	vars = eval.VarValuesMap{"l": {"field_int2": 100}}
	isMatch, _ = newScript.ScriptNodes["join_table1_table2"].Lookup.CheckFilterCondition(vars)
	assert.False(t, isMatch)

	// bad index_name, tweak it

	err = newScript.Deserialize(
		[]byte(strings.Replace(plainScriptJson, `"index_name": "idx_table2_string2"`, `"index_name": "idx_table2_string2_bad"`, 1)),
		nil, nil, "", nil)
	assert.Contains(t, err.Error(), "cannot find the node that creates index [idx_table2_string2_bad]")

	// bad join_on, tweak it

	err = newScript.Deserialize(
		[]byte(strings.Replace(plainScriptJson, `"join_on": "r.field_string1"`, `"join_on": ""`, 1)),
		nil, nil, "", nil)
	assert.Contains(t, err.Error(), "expected a comma-separated list of <table_name>.<field_name>, got []")

	err = newScript.Deserialize(
		[]byte(strings.Replace(plainScriptJson, `"join_on": "r.field_string1"`, `"join_on": "bla"`, 1)),
		nil, nil, "", nil)
	assert.Contains(t, err.Error(), "expected a comma-separated list of <table_name>.<field_name>, got [bla]")

	err = newScript.Deserialize(
		[]byte(strings.Replace(plainScriptJson, `"join_on": "r.field_string1"`, `"join_on": "bla.bla"`, 1)),
		nil, nil, "", nil)
	assert.Contains(t, err.Error(), "source table name [bla] unknown, expected [r]")

	err = newScript.Deserialize(
		[]byte(strings.Replace(plainScriptJson, `"join_on": "r.field_string1"`, `"join_on": "r.field_string1_bad"`, 1)),
		nil, nil, "", nil)
	assert.Contains(t, err.Error(), "source [r] does not produce field [field_string1_bad]")

	err = newScript.Deserialize(
		[]byte(strings.Replace(plainScriptJson, `"join_on": "r.field_string1"`, `"join_on": "r.field_int1"`, 1)),
		nil, nil, "", nil)
	assert.Contains(t, err.Error(), "left-side field field_int1 has type int, while index field field_string2 has type string")

	// bad filter, tweak it

	err = newScript.Deserialize(
		[]byte(strings.Replace(plainScriptJson, `"filter": "l.field_int2 > 100"`, `"filter": "r.field_int2 > 100"`, 1)),
		nil, nil, "", nil)
	assert.Contains(t, err.Error(), "invalid field in lookup filter [r.field_int2 > 100], only fields from the lookup table [table2](alias l) are allowed: [unknown field r.field_int2]")

	// bad join_type, tweak it

	err = newScript.Deserialize(
		[]byte(strings.Replace(plainScriptJson, `"join_type": "left"`, `"join_type": "left_bad"`, 1)),
		nil, nil, "", nil)
	assert.Contains(t, err.Error(), "invalid join type, expected inner or left, left_bad is not supported")
}

func TestBadCreatorHaving(t *testing.T) {
	var err error

	newScript := &ScriptDef{}
	assert.Nil(t, newScript.Deserialize([]byte(plainScriptJson), nil, nil, "", nil))

	// Bad expression, tweak having expression

	err = newScript.Deserialize(
		[]byte(strings.Replace(plainScriptJson, `"having": "w.total_value > 2"`, `"having": "w.total_value &> 2"`, 1)),
		nil, nil, "", nil)
	assert.Contains(t, err.Error(), "cannot parse table creator 'having' condition [w.total_value &> 2]")

	err = newScript.Deserialize(
		[]byte(strings.Replace(plainScriptJson, `"having": "w.total_value > 3"`, `"having": "w.bad_field &> 3"`, 1)),
		nil, nil, "", nil)
	assert.Contains(t, err.Error(), "cannot parse file creator 'having' condition [w.bad_field &> 3]")

	// Unknown field in having

	err = newScript.Deserialize(
		[]byte(strings.Replace(plainScriptJson, `"having": "w.total_value > 2"`, `"having": "w.bad_field > 2"`, 1)),
		nil, nil, "", nil)
	assert.Contains(t, err.Error(), "invalid field in table creator 'having' condition: [unknown field w.bad_field]")

	err = newScript.Deserialize(
		[]byte(strings.Replace(plainScriptJson, `"having": "w.total_value > 3"`, `"having": "w.bad_field > 3"`, 1)),
		nil, nil, "", nil)
	assert.Contains(t, err.Error(), "invalid field in file creator 'having' condition: [unknown field w.bad_field]]")

	// Prohibited reader field in having

	err = newScript.Deserialize(
		[]byte(strings.Replace(plainScriptJson, `"having": "w.total_value > 2"`, `"having": "r.field_int1 > 2"`, 1)),
		nil, nil, "", nil)
	assert.Contains(t, err.Error(), "invalid field in table creator 'having' condition: [prohibited field r.field_int1]")

	err = newScript.Deserialize(
		[]byte(strings.Replace(plainScriptJson, `"having": "w.total_value > 3"`, `"having": "r.field_int1 > 3"`, 1)),
		nil, nil, "", nil)
	assert.Contains(t, err.Error(), "invalid field in file creator 'having' condition: [prohibited field r.field_int1]")

	// Prohibited lookup field in table creator having

	err = newScript.Deserialize(
		[]byte(strings.Replace(plainScriptJson, `"having": "w.total_value > 2"`, `"having": "l.field_int2 > 2"`, 1)),
		nil, nil, "", nil)
	assert.Contains(t, err.Error(), "invalid field in table creator 'having' condition: [prohibited field l.field_int2]")

	// Type mismatch in having

	err = newScript.Deserialize(
		[]byte(strings.Replace(plainScriptJson, `"having": "w.total_value > 2"`, `"having": "w.total_value == true"`, 1)),
		nil, nil, "", nil)
	assert.Contains(t, err.Error(), "cannot evaluate table creator 'having' expression [w.total_value == true]: [cannot perform binary comp op, incompatible arg types '0(int64)' == 'true(bool)' ]")

	err = newScript.Deserialize(
		[]byte(strings.Replace(plainScriptJson, `"having": "w.total_value > 3"`, `"having": "w.total_value == true"`, 1)),
		nil, nil, "", nil)
	assert.Contains(t, err.Error(), "cannot evaluate file creator 'having' expression [w.total_value == true]: [cannot perform binary comp op, incompatible arg types '0(decimal.Decimal)' == 'true(bool)' ]")

}

func TestTopLimit(t *testing.T) {
	var err error

	newScript := &ScriptDef{}
	assert.Nil(t, newScript.Deserialize([]byte(plainScriptJson), nil, nil, "", nil))

	// Tweak limit beyond allowed maximum

	err = newScript.Deserialize(
		[]byte(strings.Replace(plainScriptJson, `"limit": 500000`, `"limit": 500001`, 1)),
		nil, nil, "", nil)
	assert.Contains(t, err.Error(), "top.limit cannot exceed 500000")

	// Remove limit altogether

	assert.Nil(t, newScript.Deserialize(
		[]byte(strings.Replace(plainScriptJson, `"limit": 500000`, `"some_bogus_setting": 500000`, 1)),
		nil, nil, "", nil))
	assert.Equal(t, 500000, newScript.ScriptNodes["file_totals"].FileCreator.Top.Limit)
}

func TestBatchIntervalsCalculation(t *testing.T) {
	newScript := &ScriptDef{}
	assert.Nil(t, newScript.Deserialize([]byte(plainScriptJson), nil, nil, "", nil))

	var intervals [][]int64

	tableReaderNodeDef := newScript.ScriptNodes["join_table1_table2"]
	intervals, _ = tableReaderNodeDef.GetTokenIntervalsByNumberOfBatches()

	assert.Equal(t, 2, len(intervals))
	if len(intervals) == 2 {
		assert.Equal(t, int64(-9223372036854775808), intervals[0][0])
		assert.Equal(t, int64(-2), intervals[0][1])
		assert.Equal(t, int64(-1), intervals[1][0])
		assert.Equal(t, int64(9223372036854775807), intervals[1][1])
	}

	fileReaderNodeDef := newScript.ScriptNodes["read_table1"]
	intervals, _ = fileReaderNodeDef.GetTokenIntervalsByNumberOfBatches()

	assert.Equal(t, 1, len(intervals))
	if len(intervals) == 1 {
		assert.Equal(t, int64(0), intervals[0][0])
		assert.Equal(t, int64(0), intervals[0][1])
	}

	fileCreatorNodeDef := newScript.ScriptNodes["file_totals"]
	intervals, _ = fileCreatorNodeDef.GetTokenIntervalsByNumberOfBatches()

	assert.Equal(t, 1, len(intervals))
	if len(intervals) == 1 {
		assert.Equal(t, int64(-9223372036854775808), intervals[0][0])
		assert.Equal(t, int64(9223372036854775807), intervals[0][1])
	}
}

func TestUniqueIndexesFieldRefs(t *testing.T) {
	newScript := &ScriptDef{}
	assert.Nil(t, newScript.Deserialize([]byte(plainScriptJson), nil, nil, "", nil))

	fileReaderNodeDef := newScript.ScriptNodes["read_table2"]
	fieldRefs := fileReaderNodeDef.GetUniqueIndexesFieldRefs()
	assert.Equal(t, 1, len(*fieldRefs))
	if len(*fieldRefs) == 1 {
		assert.Equal(t, "table2", (*fieldRefs)[0].TableName)
		assert.Equal(t, "field_string2", (*fieldRefs)[0].FieldName)
		assert.Equal(t, FieldTypeString, (*fieldRefs)[0].FieldType)
	}
}

func TestAffectedNodes(t *testing.T) {
	var affectedNodes []string

	newScript := &ScriptDef{}
	assert.Nil(t, newScript.Deserialize([]byte(plainScriptJson), nil, nil, "", nil))

	affectedNodes = newScript.GetAffectedNodes([]string{"read_table1"})
	assert.Equal(t, 3, len(affectedNodes))
	assert.Contains(t, affectedNodes, "read_table1")
	assert.Contains(t, affectedNodes, "join_table1_table2")
	assert.Contains(t, affectedNodes, "file_totals")

	affectedNodes = newScript.GetAffectedNodes([]string{"read_table1", "read_table2"})
	assert.Equal(t, 4, len(affectedNodes))
	assert.Contains(t, affectedNodes, "read_table1")
	assert.Contains(t, affectedNodes, "read_table2")
	assert.Contains(t, affectedNodes, "join_table1_table2")
	assert.Contains(t, affectedNodes, "file_totals")

	// Make join manual and see the list of affected nodes shrinking

	assert.Nil(t, newScript.Deserialize([]byte(strings.Replace(plainScriptJson, `"start_policy": "auto"`, `"start_policy": "manual"`, 1)), nil, nil, "", nil))

	affectedNodes = newScript.GetAffectedNodes([]string{"read_table1"})
	assert.Equal(t, 1, len(affectedNodes))
	assert.Contains(t, affectedNodes, "read_table1")

	affectedNodes = newScript.GetAffectedNodes([]string{"read_table1", "read_table2"})
	assert.Equal(t, 2, len(affectedNodes))
	assert.Contains(t, affectedNodes, "read_table1")
	assert.Contains(t, affectedNodes, "read_table2")
}
