package gocqlmem

import (
	"testing"

	gocql "github.com/apache/cassandra-gocql-driver/v2"
	"github.com/stretchr/testify/assert"
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
