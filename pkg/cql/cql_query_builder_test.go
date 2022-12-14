package cql

import (
	"testing"

	"github.com/capillariesio/capillaries/pkg/sc"
)

func TestInsert(t *testing.T) {
	const q = "INSERT INTO table1_00123 ( col1, col2, col3 ) VALUES ( 'val1', 2, now() ) IF NOT EXISTS;"
	qb := QueryBuilder{}
	s := qb.
		Write("col1", "val1").
		Write("col2", 2).
		WriteForceUnquote("col3", "now()").
		InsertRun("table1", 123, IgnoreIfExists)
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
		CondIn("col3", []interface{}{"val31", "val32"}).
		CondIn("col7", []interface{}{1, 2}).
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
		CondIn("col3", []interface{}{"val31", "val32"}).
		CondIn("col7", []interface{}{1, 2}).
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
