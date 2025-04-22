package cql

import (
	"fmt"
	"testing"

	"github.com/capillariesio/capillaries/pkg/sc"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"gopkg.in/inf.v0"
)

func TestValueToCqlParam(t *testing.T) {
	// Simple
	assert.Equal(t, "1.23", valueToCqlParam(decimal.NewFromFloat(1.23)).(*inf.Dec).String())

	// big round up
	assert.Equal(t, "1.24", valueToCqlParam(decimal.NewFromFloat(1.235)).(*inf.Dec).String())

	// small round down
	assert.Equal(t, "0.03", valueToCqlParam(decimal.NewFromFloat(0.0345)).(*inf.Dec).String())
}

func TestInsertRunParams(t *testing.T) {
	qb := NewQB()
	assert.Nil(t, qb.WritePreparedColumn("param_name"))
	assert.Nil(t, qb.WritePreparedValue("param_name", "param_value"))
	q, err := qb.Keyspace("ks1").InsertRunPreparedQuery("table1", 1, IgnoreIfExists)
	assert.Nil(t, err)
	assert.Equal(t, "INSERT INTO ks1.table1_00001 ( param_name ) VALUES ( ? ) IF NOT EXISTS;", q)

	params, _ := qb.InsertRunParams()
	assert.Equal(t, []any([]any{"param_value"}), params)
}

func TestInsert(t *testing.T) {
	const qTemplate string = "INSERT INTO table1%s ( col1, col2, col3 ) VALUES ( 'val1', 2, now() ) IF NOT EXISTS;"
	qb := (&QueryBuilder{}).
		Write("col1", "val1").
		Write("col2", 2).
		WriteForceUnquote("col3", "now()")
	assert.Equal(t, fmt.Sprintf(qTemplate, "_00123"), qb.insertRunUnpreparedQuery("table1", 123, IgnoreIfExists))
}

func TestDropKeyspace(t *testing.T) {
	assert.Equal(t, "DROP KEYSPACE IF EXISTS aaa", (&QueryBuilder{}).Keyspace("aaa").DropKeyspace())
}

func TestSelect(t *testing.T) {
	const qTemplate string = "SELECT col3, col4 FROM somekeyspace.table1%s WHERE col1 > 1 AND col2 = 2 AND col3 IN ( 'val31', 'val32' ) AND col7 IN ( 1, 2 ) ORDER BY col3  LIMIT 10;"
	qb := (&QueryBuilder{}).
		Keyspace("somekeyspace").
		Cond("col1", ">", 1).
		Cond("col2", "=", 2).
		CondInString("col3", []string{"val31", "val32"}).
		CondInInt16("col7", []int16{1, 2}).
		OrderBy("col3").
		Limit(10)

	assert.Equal(t, fmt.Sprintf(qTemplate, "_00123"), qb.SelectRun("table1", 123, []string{"col3", "col4"}))
	assert.Equal(t, fmt.Sprintf(qTemplate, ""), qb.Select("table1", []string{"col3", "col4"}))
}

func TestDelete(t *testing.T) {
	const qTemplate string = "DELETE FROM table1%s WHERE col1 > 1 AND col2 = 2 AND col3 IN ( 'val31', 'val32' ) AND col7 IN ( 1, 2 )"
	qb := (&QueryBuilder{}).
		Cond("col1", ">", 1).
		Cond("col2", "=", 2).
		CondInString("col3", []string{"val31", "val32"}).
		CondInInt("col7", []int64{1, 2})
	assert.Equal(t, fmt.Sprintf(qTemplate, "_00123"), qb.DeleteRun("table1", 123))
	assert.Equal(t, fmt.Sprintf(qTemplate, ""), qb.Delete("table1"))
}

func TestUpdate(t *testing.T) {
	const qTemplate string = "UPDATE table1%s SET col1 = 'val1', col2 = 2 WHERE col1 > 1 AND col2 = '2' IF col1 = 2"
	qb := (&QueryBuilder{}).
		Write("col1", "val1").
		Write("col2", 2).
		Cond("col1", ">", 1).
		Cond("col2", "=", "2").
		If("col1", "=", 2)
	assert.Equal(t, fmt.Sprintf(qTemplate, "_00123"), qb.UpdateRun("table1", 123))
	assert.Equal(t, fmt.Sprintf(qTemplate, ""), qb.Update("table1"))
}

func TestCreateRun(t *testing.T) {
	const qTemplate string = "CREATE TABLE IF NOT EXISTS table1%s ( col_int BIGINT, col_bool BOOLEAN, col_string TEXT, col_datetime TIMESTAMP, col_decimal2 DECIMAL, col_float DOUBLE, PRIMARY KEY((col_int, col_decimal2), col_bool, col_float));"
	qb := (&QueryBuilder{}).
		ColumnDef("col_int", sc.FieldTypeInt).
		ColumnDef("col_bool", sc.FieldTypeBool).
		ColumnDef("col_string", sc.FieldTypeString).
		ColumnDef("col_datetime", sc.FieldTypeDateTime).
		ColumnDef("col_decimal2", sc.FieldTypeDecimal2).
		ColumnDef("col_float", sc.FieldTypeFloat).
		PartitionKey("col_int", "col_decimal2").
		ClusteringKey("col_bool", "col_float")
	assert.Equal(t, fmt.Sprintf(qTemplate, "_00123"), qb.CreateRun("table1", 123, IgnoreIfExists))
}

func TestInsertPrepared(t *testing.T) {
	dataQb := NewQB()
	err := dataQb.WritePreparedColumn("col_int")
	assert.Nil(t, err)
	err = dataQb.WritePreparedValue("col_int", 2)
	assert.Nil(t, err)
	s, _ := dataQb.InsertRunPreparedQuery("table1", 123, IgnoreIfExists)
	assert.Equal(t, "INSERT INTO table1_00123 ( col_int ) VALUES ( ? ) IF NOT EXISTS;", s)
}
