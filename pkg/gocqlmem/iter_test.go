package gocqlmem

import (
	"testing"

	gocql "github.com/apache/cassandra-gocql-driver/v2"
	"github.com/stretchr/testify/assert"
	"gopkg.in/inf.v0"
)

func TestColumnsAndRowData(t *testing.T) {
	s := NewGocqlmemSession()
	assert.Nil(t, s.Query("CREATE KEYSPACE ks1").Exec())
	assert.Nil(t, s.Query("CREATE TABLE ks1.t1 (f_int int, f_text text, f_bool boolean, f_float float, f_dec decimal, primary key (f_int))").Exec())

	dest := map[string]interface{}{}
	var isApplied bool
	var err error
	isApplied, err = s.Query("INSERT INTO ks1.t1 (f_int, f_text, f_bool, f_float, f_dec) VALUES (1,'1', TRUE, 1.1, 2.2)").MapScanCAS(dest)
	assert.Nil(t, err)
	assert.True(t, isApplied)

	iter := s.Query(`SELECT f_int, f_text, f_bool, f_float, f_dec FROM ks1.t1`).Iter()
	assert.Nil(t, iter.Err())
	cols := iter.Columns()

	assert.Equal(t, "ks1", cols[0].Keyspace)
	assert.Equal(t, "t1", cols[0].Table)
	assert.Equal(t, "f_int", cols[0].Name)
	assert.Equal(t, gocql.TypeInt, cols[0].TypeInfo.Type())

	assert.Equal(t, "ks1", cols[1].Keyspace)
	assert.Equal(t, "t1", cols[1].Table)
	assert.Equal(t, "f_text", cols[1].Name)
	assert.Equal(t, gocql.TypeText, cols[1].TypeInfo.Type())

	assert.Equal(t, "ks1", cols[2].Keyspace)
	assert.Equal(t, "t1", cols[2].Table)
	assert.Equal(t, "f_bool", cols[2].Name)
	assert.Equal(t, gocql.TypeBoolean, cols[2].TypeInfo.Type())

	assert.Equal(t, "ks1", cols[3].Keyspace)
	assert.Equal(t, "t1", cols[3].Table)
	assert.Equal(t, "f_float", cols[3].Name)
	assert.Equal(t, gocql.TypeFloat, cols[3].TypeInfo.Type())

	assert.Equal(t, "ks1", cols[4].Keyspace)
	assert.Equal(t, "t1", cols[4].Table)
	assert.Equal(t, "f_dec", cols[4].Name)
	assert.Equal(t, gocql.TypeDecimal, cols[4].TypeInfo.Type())

	rowData, err := iter.RowData()
	assert.Nil(t, err)
	assert.Equal(t, 5, len(rowData.Columns))
	assert.Equal(t, int64(0), rowData.Values[0])
	assert.Equal(t, "", rowData.Values[1])
	assert.Equal(t, false, rowData.Values[2])
	assert.Equal(t, float64(0.0), rowData.Values[3])
	assert.Equal(t, *new(inf.Dec), rowData.Values[4])
}
func TestIterScan(t *testing.T) {
	s := NewGocqlmemSession()
	assert.Nil(t, s.Query("CREATE KEYSPACE ks1").Exec())
	assert.Nil(t, s.Query("CREATE TABLE ks1.t1 (f_int int, f_text text, f_bool boolean, f_float float, f_dec decimal, primary key (f_int))").Exec())

	var err error
	err = s.Query("INSERT INTO ks1.t1 (f_int, f_text, f_bool, f_float, f_dec) VALUES (1, '1', TRUE, 1.1, 2.2)").Exec()
	assert.Nil(t, err)

	iter := s.Query(`SELECT f_int, f_text, f_bool, f_float, f_dec FROM ks1.t1`).Iter()
	assert.Nil(t, iter.Err())

	resultInt := int32(0)
	resultText := ""
	resultBool := false
	resultFloat := float32(0.0)
	resultDec := *float64ToDecNoCheck(float64(0.0))
	ok := iter.Scan(&resultInt, &resultText, &resultBool, &resultFloat, &resultDec)
	assert.True(t, ok)
	assert.Equal(t, int32(1), resultInt)
	assert.Equal(t, "1", resultText)
	assert.Equal(t, true, resultBool)
	assert.Equal(t, float32(1.1), resultFloat)
	assert.Equal(t, *float64ToDecNoCheck(float64(2.2)), resultDec)
}

func TestScanner(t *testing.T) {
	s := NewGocqlmemSession()
	assert.Nil(t, s.Query("CREATE KEYSPACE ks1").Exec())
	assert.Nil(t, s.Query("CREATE TABLE ks1.t1 (f_int int, f_text text, f_bool boolean, f_float float, f_dec decimal, primary key (f_int))").Exec())

	dest := map[string]interface{}{}
	var isApplied bool
	var err error
	isApplied, err = s.Query("INSERT INTO ks1.t1 (f_int, f_bigint,  f_text, f_bool, f_float, f_dec) VALUES (1, 2, '1', TRUE, 1.1, 2.2)").MapScanCAS(dest)
	assert.Nil(t, err)
	assert.True(t, isApplied)

	resultInt := int32(0)
	resultBigint := int64(0)
	resultText := ""
	resultBool := false
	resultFloat := float32(0.0)
	resultDec := *float64ToDecNoCheck(float64(2.2))

	iter := s.Query(`SELECT f_int, f_bigint, f_text, f_bool, f_float, f_dec FROM ks1.t1`).Iter()
	assert.Nil(t, err)
	scanner := iter.Scanner()
	for scanner.Next() {
		err = scanner.Scan(&resultInt, &resultText, &resultBool, &resultFloat, &resultDec)
		assert.Nil(t, err)
		assert.Equal(t, int32(1), resultInt)
		assert.Equal(t, int64(2), resultBigint)
		assert.Equal(t, "1", resultText)
		assert.Equal(t, true, resultBool)
		assert.Equal(t, float32(1.1), resultFloat)
		assert.Equal(t, *float64ToDecNoCheck(float64(2.2)), resultDec)
	}
}
