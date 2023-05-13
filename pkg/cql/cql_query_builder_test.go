package cql

import (
	"testing"

	"github.com/capillariesio/capillaries/pkg/sc"
	"github.com/shopspring/decimal"
	"gopkg.in/inf.v0"
)

func TestValueToCqlParam(t *testing.T) {
	// Simple
	expected := "1.23"
	actual := valueToCqlParam(decimal.NewFromFloat(1.23)).(*inf.Dec).String()
	if actual != expected {
		t.Errorf("Unmatch:\n%v\n%v\n", expected, actual)
	}

	// big round up
	expected = "1.24"
	actual = valueToCqlParam(decimal.NewFromFloat(1.235)).(*inf.Dec).String()
	if actual != expected {
		t.Errorf("Unmatch:\n%v\n%v\n", expected, actual)
	}

	// small round down
	expected = "0.03"
	actual = valueToCqlParam(decimal.NewFromFloat(0.0345)).(*inf.Dec).String()
	if actual != expected {
		t.Errorf("Unmatch:\n%v\n%v\n", expected, actual)
	}

}

func TestInsert(t *testing.T) {
	const q = "INSERT INTO table1_00123 ( col1, col2, col3 ) VALUES ( 'val1', 2, now() ) IF NOT EXISTS;"
	qb := QueryBuilder{}
	s := qb.
		Write("col1", "val1").
		Write("col2", 2).
		WriteForceUnquote("col3", "now()").
		insertRunUnpreparedQuery("table1", 123, IgnoreIfExists)
	if s != q {
		t.Errorf("Unmatch:\n%v\n%v\n", q, s)
	}
}

func TestSelect(t *testing.T) {
	const q = "SELECT col3, col4 FROM somekeyspace.table1_00123 WHERE col1 > 1 AND col2 = 2 AND col3 IN ( 'val31', 'val32' ) AND col7 IN ( 1, 2 );"
	qb := QueryBuilder{}
	s := qb.
		Keyspace("somekeyspace").
		Cond("col1", ">", 1).
		Cond("col2", "=", 2).
		CondInString("col3", []string{"val31", "val32"}).
		CondInInt16("col7", []int16{1, 2}).
		SelectRun("table1", 123, []string{"col3", "col4"})
	if s != q {
		t.Errorf("Unmatch:\n%v\n%v\n", q, s)
	}
}

func TestDelete(t *testing.T) {
	const q = "DELETE FROM table1_00123 WHERE col1 > 1 AND col2 = 2 AND col3 IN ( 'val31', 'val32' ) AND col7 IN ( 1, 2 )"
	qb := QueryBuilder{}
	s := qb.
		Cond("col1", ">", 1).
		Cond("col2", "=", 2).
		CondInString("col3", []string{"val31", "val32"}).
		CondInInt("col7", []int64{1, 2}).
		DeleteRun("table1", 123)
	if s != q {
		t.Errorf("Unmatch:\n%v\n%v\n", q, s)
	}
}

func TestUpdate(t *testing.T) {
	const q = "UPDATE table1_00123 SET col1 = 'val1', col2 = 2 WHERE col1 > 1 AND col2 = '2' IF col1 = 2"
	qb := QueryBuilder{}
	s := qb.
		Write("col1", "val1").
		Write("col2", 2).
		Cond("col1", ">", 1).
		Cond("col2", "=", "2").
		If("col1", "=", 2).
		UpdateRun("table1", 123)
	if s != q {
		t.Errorf("Unmatch:\n%v\n%v\n", q, s)
	}
}

func TestCreate(t *testing.T) {
	const q = "CREATE TABLE IF NOT EXISTS table1_00123 ( col_int BIGINT, col_bool BOOLEAN, col_string TEXT, col_datetime TIMESTAMP, col_decimal2 DECIMAL, col_float DOUBLE, PRIMARY KEY((col_int, col_decimal2), col_bool, col_float));"
	qb := QueryBuilder{}
	s := qb.
		ColumnDef("col_int", sc.FieldTypeInt).
		ColumnDef("col_bool", sc.FieldTypeBool).
		ColumnDef("col_string", sc.FieldTypeString).
		ColumnDef("col_datetime", sc.FieldTypeDateTime).
		ColumnDef("col_decimal2", sc.FieldTypeDecimal2).
		ColumnDef("col_float", sc.FieldTypeFloat).
		PartitionKey("col_int", "col_decimal2").
		ClusteringKey("col_bool", "col_float").
		CreateRun("table1", 123, IgnoreIfExists)
	if s != q {
		t.Errorf("Unmatch:\n%v\n%v\n", q, s)
	}
}

func TestInsertPrepared(t *testing.T) {
	const q = "INSERT INTO table1_00123 ( col_int ) VALUES ( ? ) IF NOT EXISTS;"
	dataQb := NewQB()
	dataQb.WritePreparedColumn("col_int")
	dataQb.WritePreparedValue("col_int", 2)
	s, _ := dataQb.InsertRunPreparedQuery("table1", 123, IgnoreIfExists)
	if s != q {
		t.Errorf("Unmatch:\n%v\n%v\n", q, s)
	}
}
