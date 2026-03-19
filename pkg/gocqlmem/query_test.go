package gocqlmem

import (
	"fmt"
	"testing"
	"time"

	gocql "github.com/apache/cassandra-gocql-driver/v2"
	"github.com/stretchr/testify/assert"
	"gopkg.in/inf.v0"
)

func TestMapScanCAS(t *testing.T) {
	s := NewGocqlmemSession()
	assert.Nil(t, s.Query("CREATE KEYSPACE ks1").Exec())
	assert.Nil(t, s.Query("CREATE TABLE ks1.t1 (a int, b bigint, c smallint, d tinyint, primary key (a))").Exec())

	dest := map[string]interface{}{}
	var isApplied bool
	var err error
	isApplied, err = s.Query("INSERT INTO ks1.t1 (a,b,c,d) VALUES (1,1,1,1)").MapScanCAS(dest)
	assert.Nil(t, err)
	assert.True(t, isApplied)

	assert.Equal(t, nil, dest["a"])
	assert.Equal(t, nil, dest["b"])
	assert.Equal(t, nil, dest["c"])
	assert.Equal(t, nil, dest["d"])

	isApplied, err = s.Query(`UPDATE ks1.t1 SET b=2 WHERE a=2 IF EXISTS`).MapScanCAS(dest)
	assert.Nil(t, err)
	assert.False(t, isApplied)

	assert.Equal(t, nil, dest["a"])
	assert.Equal(t, nil, dest["b"])
	assert.Equal(t, nil, dest["c"])
	assert.Equal(t, nil, dest["d"])

	isApplied, err = s.Query(`UPDATE ks1.t1 SET b=2,c=2,d=2 WHERE a=1 IF EXISTS`).MapScanCAS(dest)
	assert.Nil(t, err)
	assert.True(t, isApplied)

	assert.Equal(t, nil, dest["a"])
	assert.Equal(t, nil, dest["b"])
	assert.Equal(t, nil, dest["c"])
	assert.Equal(t, nil, dest["d"])

	result := []map[string]interface{}{}
	result, err = s.Query(`SELECT a,b,c,d FROM ks1.t1 WHERE a=1`).Iter().SliceMap()
	assert.Nil(t, err)
	assert.Equal(t, 1, len(result))

	assert.Equal(t, int32(1), result[0]["a"])
	assert.Equal(t, int64(2), result[0]["b"])
	assert.Equal(t, int16(2), result[0]["c"])
	assert.Equal(t, int8(2), result[0]["d"])

	// IF TRUE, b 2->3

	isApplied, err = s.Query(`UPDATE ks1.t1 SET b=3,c=3,d=3 WHERE a=1 IF a=1`).MapScanCAS(dest)
	assert.Nil(t, err)
	assert.True(t, isApplied)

	assert.Equal(t, nil, dest["a"])
	assert.Equal(t, nil, dest["b"])
	assert.Equal(t, nil, dest["c"])
	assert.Equal(t, nil, dest["d"])

	result, err = s.Query(`SELECT a,b,c,d FROM ks1.t1 WHERE a=1`).Iter().SliceMap()
	assert.Nil(t, err)
	assert.Equal(t, 1, len(result))

	assert.Equal(t, int32(1), result[0]["a"])
	assert.Equal(t, int64(3), result[0]["b"])
	assert.Equal(t, int16(3), result[0]["c"])
	assert.Equal(t, int8(3), result[0]["d"])

	// IF FALSE, b remains 3

	isApplied, err = s.Query(`UPDATE ks1.t1 SET b=4 WHERE a=1 IF a=100`).MapScanCAS(dest)
	assert.Nil(t, err)
	assert.True(t, isApplied)

	assert.Equal(t, nil, dest["a"])
	assert.Equal(t, nil, dest["b"])
	assert.Equal(t, nil, dest["c"])
	assert.Equal(t, nil, dest["d"])

	result, err = s.Query(`SELECT a,b,c,d FROM ks1.t1 WHERE a=1`).Iter().SliceMap()
	assert.Nil(t, err)
	assert.Equal(t, 1, len(result))

	assert.Equal(t, int32(1), result[0]["a"])
	assert.Equal(t, int64(3), result[0]["b"])
	assert.Equal(t, int16(3), result[0]["c"])
	assert.Equal(t, int8(3), result[0]["d"])
}

func TestMapScanCASUpsert(t *testing.T) {
	s := NewGocqlmemSession()
	assert.Nil(t, s.Query("CREATE KEYSPACE ks1").Exec())
	assert.Nil(t, s.Query("CREATE TABLE ks1.t1 (a int, b bigint, c smallint, d tinyint, primary key (a))").Exec())

	dest := map[string]interface{}{}
	var isApplied bool
	var err error
	isApplied, err = s.Query("INSERT INTO ks1.t1 (a,b,c,d) VALUES (1,1,1,1)").MapScanCAS(dest)
	assert.Nil(t, err)
	assert.True(t, isApplied)

	isApplied, err = s.Query("INSERT INTO ks1.t1 (a,b,c,d) VALUES (1,3,3,3) IF NOT EXISTS").MapScanCAS(dest)
	assert.Nil(t, err)
	assert.False(t, isApplied)
	assert.Equal(t, int32(1), dest["a"])
	assert.Equal(t, int64(1), dest["b"])
	assert.Equal(t, int16(1), dest["c"])
	assert.Equal(t, int8(1), dest["d"])

	isApplied, err = s.Query("INSERT INTO ks1.t1 (a,b,c,d) VALUES (1,3,3,3)").MapScanCAS(dest)
	assert.Contains(t, "cannot upsert duplicate map[a:1 b:3 c:3 d:3]", err.Error())
	assert.False(t, isApplied)
	assert.Equal(t, int32(1), dest["a"])
	assert.Equal(t, int64(1), dest["b"])
	assert.Equal(t, int16(1), dest["c"])
	assert.Equal(t, int8(1), dest["d"])
}

func TestPageSize(t *testing.T) {
	s := NewGocqlmemSession()
	assert.Nil(t, s.Query("CREATE KEYSPACE ks1").Exec())
	assert.Nil(t, s.Query("CREATE TABLE ks1.t1 (a int, b bigint, primary key (a))").Exec())

	dest := map[string]interface{}{}
	var isApplied bool
	var err error
	isApplied, err = s.Query("INSERT INTO ks1.t1 (a,b) VALUES (1,1)").MapScanCAS(dest)
	assert.Nil(t, err)
	assert.True(t, isApplied)
	isApplied, err = s.Query("INSERT INTO ks1.t1 (a,b) VALUES (2,2)").MapScanCAS(dest)
	assert.Nil(t, err)
	assert.True(t, isApplied)
	isApplied, err = s.Query("INSERT INTO ks1.t1 (a,b) VALUES (3,3)").MapScanCAS(dest)
	assert.Nil(t, err)
	assert.True(t, isApplied)
	isApplied, err = s.Query("INSERT INTO ks1.t1 (a,b) VALUES (4,4)").MapScanCAS(dest)
	assert.Nil(t, err)
	assert.True(t, isApplied)

	resultA := int32(0)
	resultB := int64(0)

	iter := s.Query(`SELECT a,b FROM ks1.t1`).PageSize(2).PageState([]byte{}).Iter()
	assert.Nil(t, err)
	nextPageState := iter.PageState()

	scanner := iter.Scanner()

	assert.True(t, scanner.Next())
	err = scanner.Scan(&resultA, &resultB)
	assert.Nil(t, err)
	assert.Equal(t, int32(1), resultA)
	assert.Equal(t, int64(1), resultB)

	assert.True(t, scanner.Next())
	err = scanner.Scan(&resultA, &resultB)
	assert.Nil(t, err)
	assert.Equal(t, int32(2), resultA)
	assert.Equal(t, int64(2), resultB)

	iter = s.Query(`SELECT a,b FROM ks1.t1`).PageSize(2).PageState(nextPageState).Iter()
	assert.Nil(t, err)
	scanner = iter.Scanner()

	assert.True(t, scanner.Next())
	err = scanner.Scan(&resultA, &resultB)
	assert.Nil(t, err)
	assert.Equal(t, int32(3), resultA)
	assert.Equal(t, int64(3), resultB)

	assert.True(t, scanner.Next())
	err = scanner.Scan(&resultA, &resultB)
	assert.Nil(t, err)
	assert.Equal(t, int32(4), resultA)
	assert.Equal(t, int64(4), resultB)
}

func TestUuid(t *testing.T) {
	s := NewGocqlmemSession()
	assert.Nil(t, s.Query("CREATE KEYSPACE ks1").Exec())
	assert.Nil(t, s.Query("CREATE TABLE ks1.t1 (a int, b uuid, c timeuuid, primary key (a))").Exec())

	var err error
	err = s.Query("INSERT INTO ks1.t1 (a,b,c) VALUES (1,now(),now())").Exec()
	assert.Nil(t, err)

	result := []map[string]interface{}{}
	result, err = s.Query(`SELECT a,b,c FROM ks1.t1 WHERE a=1`).Iter().SliceMap()
	assert.Nil(t, err)
	assert.Equal(t, 1, len(result))
	uB := result[0]["b"].(gocql.UUID)
	uC := result[0]["c"].(gocql.UUID)
	// Some bytes should match
	assert.Equal(t, uB[0], uC[0])
	assert.Equal(t, uB[1], uC[1])
	assert.Equal(t, uB[14], uC[14])
	assert.Equal(t, uB[15], uC[15])
}

func TestTypes(t *testing.T) {
	s := NewGocqlmemSession()
	assert.Nil(t, s.Query("CREATE KEYSPACE ks1").Exec())
	assert.Nil(t, s.Query(`CREATE TABLE ks1.t1 (
		f_tinyint tinyint,
		f_smallint smallint,
		f_int int,
		f_bigint bigint,
		f_counter counter,
		f_bool boolean,
		f_float float,
		f_double double,
		f_dec decimal,
		f_timeuuid timeuuid,
		f_uuid uuid,
		f_blob blob,
		f_timestamp timestamp,
		f_time time,
		f_date date,
		f_varchar varchar,
		f_text text,
		f_ascii ascii,
		primary key (f_tinyint))`).Exec())

	timeToTest := time.Unix(1436832817, 476000000).UTC()
	sometimeuuid := gocql.TimeUUIDWith(0, 0, []byte{})
	preparedQueryParams := []any{sometimeuuid, sometimeuuid, []byte{1, 2}, timeToTest}
	var err error
	err = s.Query(`INSERT INTO ks1.t1 (
		f_tinyint,
		f_smallint,
		f_int,
		f_bigint,
		f_bool,
		f_float,
		f_double,
		f_dec,
		f_timeuuid,
		f_uuid,
		f_blob,
		f_timestamp,
		f_time,
		f_date,
		f_varchar,
		f_text,
		f_ascii
	)
	VALUES (
		1,
		2,
		3,
		4,
		TRUE,
		1.1,
		1.2,
		1.3,
		?,
		?,
		?,
		?,
		10000000,
		6,
		'1',
		'2',
		'3'
)`, preparedQueryParams...).Exec()
	assert.Nil(t, err)

	result := []map[string]interface{}{}
	result, err = s.Query(`SELECT * FROM ks1.t1`).Iter().SliceMap()
	assert.Nil(t, err)
	assert.Equal(t, 1, len(result))
	assert.Equal(t, "map[f_ascii:3 f_bigint:4 f_blob:[1 2] f_bool:true f_counter:0 f_date:6 f_dec:{{false [13]} 1} f_double:1.2 f_float:1.1 f_int:3 f_smallint:2 f_text:2 f_time:10000000 f_timestamp:2015-07-14 00:13:37.476 +0000 UTC f_timeuuid:00000000-0000-1000-8000-000000000000 f_tinyint:1 f_uuid:00000000-0000-1000-8000-000000000000 f_varchar:1]", fmt.Sprintf("%v", result[0]))

	iter := s.Query(`SELECT * FROM ks1.t1`).Iter()
	assert.Nil(t, iter.Err())

	rowData, err := iter.RowData()
	assert.Nil(t, err)

	assert.True(t, iter.Scan(rowData.Values...))
	assert.Nil(t, iter.Err())
	assert.Equal(t, int8(1), *(rowData.Values[0]).(*int8))
	assert.Equal(t, int16(2), *(rowData.Values[1]).(*int16))
	assert.Equal(t, int32(3), *(rowData.Values[2]).(*int32))
	assert.Equal(t, int64(4), *(rowData.Values[3]).(*int64))
	assert.Equal(t, int64(0), *(rowData.Values[4]).(*int64)) // counter
	assert.Equal(t, true, *(rowData.Values[5]).(*bool))
	assert.Equal(t, float32(1.1), *(rowData.Values[6]).(*float32))
	assert.Equal(t, float64(1.2), *(rowData.Values[7]).(*float64))
	assert.Equal(t, *inf.NewDec(13, 1), *(rowData.Values[8]).(*inf.Dec))
	assert.Equal(t, sometimeuuid, *(rowData.Values[9]).(*gocql.UUID))
	assert.Equal(t, sometimeuuid, *(rowData.Values[10]).(*gocql.UUID))
	assert.Equal(t, []byte{1, 2}, *(rowData.Values[11]).(*[]byte))
	assert.Equal(t, timeToTest, *(rowData.Values[12]).(*time.Time))
	assert.Equal(t, int64(10000000), *(rowData.Values[13]).(*int64))
	assert.Equal(t, int32(6), *(rowData.Values[14]).(*int32))
	assert.Equal(t, "1", *(rowData.Values[15]).(*string))
	assert.Equal(t, "2", *(rowData.Values[16]).(*string))
	assert.Equal(t, "3", *(rowData.Values[17]).(*string))

	// Accept int32 into int

	iter = s.Query(`SELECT f_date FROM ks1.t1`).Iter()
	assert.Nil(t, iter.Err())

	rowData, err = iter.RowData()
	assert.Nil(t, err)

	var valInt int
	rowData.Values[0] = &valInt

	assert.True(t, iter.Scan(rowData.Values...))
	assert.Nil(t, iter.Err())
	assert.Equal(t, 6, *(rowData.Values[0]).(*int))
}
