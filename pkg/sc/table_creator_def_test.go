package sc

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gopkg.in/inf.v0"
)

func TestCreatorDefaultFieldValues(t *testing.T) {
	conf := `
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
		}
	}`
	c := TableCreatorDef{}
	assert.Nil(t, c.Deserialize([]byte(conf)))

	var err error
	var val interface{}

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

	assert.Nil(t, c.Deserialize([]byte(confReplacer.Replace(conf))))

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

}
