package sc

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gopkg.in/inf.v0"
)

const tableCreatorNodeJson string = `
{
	"name": "test_table_creator",
	"fields": {
		"field_int": {
			"expression": "r.field_int",
			"default_value": "99",
			"type": "int"
		},
		"field_float": {
			"expression": "r.field_float",
			"default_value": "99.0",
			"type": "float"
		},
		"field_decimal2": {
			"expression": "r.field_decimal2",
			"default_value": "123.00",
			"type": "decimal2"
		},
		"field_datetime": {
			"expression": "r.field_datetime",
			"default_value": "1980-02-03T04:05:06.777+00:00",
			"type": "datetime"
		},
		"field_bool": {
			"expression": "r.field_bool",
			"default_value": "true",
			"type": "bool"
		},
		"field_string": {
			"expression": "r.field_string",
			"default_value": "some_string",
			"type": "string"
		}
	},
	"indexes": {
		"idx_1": "unique(field_string(case_sensitive))"
	}
}`

func TestCreatorDefaultFieldValues(t *testing.T) {
	c := TableCreatorDef{}
	assert.Nil(t, c.Deserialize([]byte(tableCreatorNodeJson)))

	var err error
	var val any

	val, err = c.GetFieldDefaultReadyForDb("field_int")
	assert.Nil(t, err)
	assert.Equal(t, int64(99), val.(int64))

	val, err = c.GetFieldDefaultReadyForDb("field_float")
	assert.Nil(t, err)
	assert.Equal(t, float64(99.0), val.(float64))

	val, err = c.GetFieldDefaultReadyForDb("field_decimal2")
	assert.Nil(t, err)
	assert.Equal(t, inf.NewDec(12300, 2), val.(*inf.Dec))

	val, err = c.GetFieldDefaultReadyForDb("field_datetime")
	assert.Nil(t, err)
	dt, _ := time.Parse(CassandraDatetimeFormat, "1980-02-03T04:05:06.777+00:00")
	assert.Equal(t, dt, val.(time.Time))

	val, err = c.GetFieldDefaultReadyForDb("field_bool")
	assert.Nil(t, err)
	assert.Equal(t, true, val.(bool))

	val, err = c.GetFieldDefaultReadyForDb("field_string")
	assert.Nil(t, err)
	assert.Equal(t, "some_string", val.(string))

	confReplacer := strings.NewReplacer(
		`"default_value": "99",`, ``,
		`"default_value": "99.0",`, ``,
		`"default_value": "123.00",`, ``,
		`"default_value": "1980-02-03T04:05:06.777+00:00",`, ``,
		`"default_value": "true",`, ``,
		`"default_value": "some_string",`, ``)

	assert.Nil(t, c.Deserialize([]byte(confReplacer.Replace(tableCreatorNodeJson))))

	val, err = c.GetFieldDefaultReadyForDb("field_int")
	assert.Nil(t, err)
	assert.Equal(t, int64(0), val.(int64))

	val, err = c.GetFieldDefaultReadyForDb("field_float")
	assert.Nil(t, err)
	assert.Equal(t, float64(0.0), val.(float64))

	val, err = c.GetFieldDefaultReadyForDb("field_decimal2")
	assert.Nil(t, err)
	assert.Equal(t, inf.NewDec(0, 0), val.(*inf.Dec))

	val, err = c.GetFieldDefaultReadyForDb("field_datetime")
	assert.Nil(t, err)
	assert.Equal(t, DefaultDateTime(), val.(time.Time))

	val, err = c.GetFieldDefaultReadyForDb("field_bool")
	assert.Nil(t, err)
	assert.False(t, val.(bool))

	val, err = c.GetFieldDefaultReadyForDb("field_string")
	assert.Nil(t, err)
	assert.Equal(t, "", val.(string))

	// Failures
	err = c.Deserialize([]byte(strings.ReplaceAll(tableCreatorNodeJson, "test_table_creator", "&")))
	assert.Contains(t, err.Error(), "invalid table name [&]: allowed regex is")

	err = c.Deserialize([]byte(strings.ReplaceAll(tableCreatorNodeJson, "test_table_creator", "idx_a")))
	assert.Contains(t, err.Error(), "invalid table name [idx_a]: prohibited regex is")

	err = c.Deserialize([]byte(strings.ReplaceAll(tableCreatorNodeJson, "string", "bad_type")))
	assert.Contains(t, err.Error(), "invalid field type [bad_type]")

	c = TableCreatorDef{}
	err = c.Deserialize([]byte(strings.ReplaceAll(tableCreatorNodeJson, "idx_1", "bad_idx_name")))
	assert.Contains(t, err.Error(), "invalid index name [bad_idx_name]: allowed regex is")

	// Check default fields
	_, err = c.GetFieldDefaultReadyForDb("bad_field")
	assert.Contains(t, err.Error(), "default for unknown field bad_field")

	c = TableCreatorDef{}

	err = c.Deserialize([]byte(strings.ReplaceAll(tableCreatorNodeJson, "99", "aaa")))
	assert.Nil(t, err)
	_, err = c.GetFieldDefaultReadyForDb("field_int")
	assert.Contains(t, err.Error(), "cannot read int64 field field_int from default value string 'aaa'")

	err = c.Deserialize([]byte(strings.ReplaceAll(tableCreatorNodeJson, "99.0", "aaa")))
	assert.Nil(t, err)
	_, err = c.GetFieldDefaultReadyForDb("field_float")
	assert.Contains(t, err.Error(), "cannot read float64 field field_float from default value string 'aaa'")

	err = c.Deserialize([]byte(strings.ReplaceAll(tableCreatorNodeJson, "123.00", "aaa")))
	assert.Nil(t, err)
	_, err = c.GetFieldDefaultReadyForDb("field_decimal2")
	assert.Contains(t, err.Error(), "cannot read decimal2 field field_decimal2 from default value string 'aaa'")

	err = c.Deserialize([]byte(strings.ReplaceAll(tableCreatorNodeJson, "1980-02-03T04:05:06.777+00:00", "aaa")))
	assert.Nil(t, err)
	_, err = c.GetFieldDefaultReadyForDb("field_datetime")
	assert.Contains(t, err.Error(), "cannot read time field field_datetime from default value string 'aaa'")

	err = c.Deserialize([]byte(strings.ReplaceAll(tableCreatorNodeJson, "true", "aaa")))
	assert.Nil(t, err)
	_, err = c.GetFieldDefaultReadyForDb("field_bool")
	assert.Contains(t, err.Error(), "cannot read bool field field_bool, from default value string 'aaa'")

}
