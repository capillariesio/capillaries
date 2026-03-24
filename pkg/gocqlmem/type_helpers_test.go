package gocqlmem

import (
	"fmt"
	"testing"
	"time"

	gocql "github.com/apache/cassandra-gocql-driver/v2"
	"github.com/stretchr/testify/assert"
	"gopkg.in/inf.v0"
)

func TestTypesSelect(t *testing.T) {
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

	result, err := s.Query(`SELECT * FROM ks1.t1`).Iter().SliceMap()
	assert.Nil(t, err)
	assert.Equal(t, 1, len(result))
	assert.Equal(t, "map[f_ascii:3 f_bigint:4 f_blob:[1 2] f_bool:true f_counter:0 f_date:6 f_dec:{{false [13]} 1} f_double:1.2 f_float:1.1 f_int:3 f_smallint:2 f_text:2 f_time:10000000 f_timestamp:2015-07-14 00:13:37.476 +0000 UTC f_timeuuid:00000000-0000-1000-8000-000000000000 f_tinyint:1 f_uuid:00000000-0000-1000-8000-000000000000 f_varchar:1]", fmt.Sprintf("%v", result[0]))

	// The following mostly tests internalValueToClientType()

	// Accept in-kind using RowData()
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

	// Accept int8, int16, int32 into int
	iter = s.Query(`SELECT f_tinyint, f_smallint, f_int, f_date FROM ks1.t1`).Iter()
	assert.Nil(t, iter.Err())
	valuesInt := []int{0, 0, 0, 0}
	assert.True(t, iter.Scan(&valuesInt[0], &valuesInt[1], &valuesInt[2], &valuesInt[3]))
	assert.Nil(t, iter.Err())
	assert.Equal(t, 1, valuesInt[0])
	assert.Equal(t, 2, valuesInt[1])
	assert.Equal(t, 3, valuesInt[2])
	assert.Equal(t, 6, valuesInt[3])

	// Accept int8 into int16
	iter = s.Query(`SELECT f_tinyint FROM ks1.t1`).Iter()
	assert.Nil(t, iter.Err())
	valuesInt16 := []int16{0}
	assert.True(t, iter.Scan(&valuesInt16[0]))
	assert.Nil(t, iter.Err())
	assert.Equal(t, int16(1), valuesInt16[0])

	// Accept int8,int16 into int32
	iter = s.Query(`SELECT f_tinyint, f_smallint FROM ks1.t1`).Iter()
	assert.Nil(t, iter.Err())
	valuesInt32 := []int32{0, 0}
	assert.True(t, iter.Scan(&valuesInt32[0], &valuesInt32[1]))
	assert.Nil(t, iter.Err())
	assert.Equal(t, int32(1), valuesInt32[0])
	assert.Equal(t, int32(2), valuesInt32[1])

	// Accept int8,int16,int32 into int64
	iter = s.Query(`SELECT f_tinyint, f_smallint, f_int, f_date, FROM ks1.t1`).Iter()
	assert.Nil(t, iter.Err())
	valuesInt64 := []int64{0, 0, 0, 0}
	assert.True(t, iter.Scan(&valuesInt64[0], &valuesInt64[1], &valuesInt64[2], &valuesInt64[3]))
	assert.Nil(t, iter.Err())
	assert.Equal(t, int64(1), valuesInt64[0])
	assert.Equal(t, int64(2), valuesInt64[1])
	assert.Equal(t, int64(3), valuesInt64[2])
	assert.Equal(t, int64(6), valuesInt64[3])

	// Accept float32 into float64
	iter = s.Query(`SELECT f_float FROM ks1.t1`).Iter()
	assert.Nil(t, iter.Err())
	valuesFloat64 := []float64{0.0}
	assert.True(t, iter.Scan(&valuesFloat64[0]))
	assert.Nil(t, iter.Err())
	assert.Equal(t, float64(1.100000023841858), valuesFloat64[0]) // Artifacts, the price we pay for float32->float64 conversion
}

func TestTypesSelectWithParams(t *testing.T) {
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
		primary key (f_tinyint,f_smallint,f_int,f_bigint,f_bool,f_float,f_double,f_dec,f_timeuuid,f_uuid,f_blob,f_timestamp,f_time,f_date,f_varchar,f_text,f_ascii))`).Exec())

	// The following mostly tests internalValueToClientType()

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

	result, err := s.Query(`SELECT * FROM ks1.t1`).Iter().SliceMap()
	assert.Nil(t, err)
	assert.Equal(t, 1, len(result))
	assert.Equal(t, "map[f_ascii:3 f_bigint:4 f_blob:[1 2] f_bool:true f_counter:0 f_date:6 f_dec:{{false [13]} 1} f_double:1.2 f_float:1.1 f_int:3 f_smallint:2 f_text:2 f_time:10000000 f_timestamp:2015-07-14 00:13:37.476 +0000 UTC f_timeuuid:00000000-0000-1000-8000-000000000000 f_tinyint:1 f_uuid:00000000-0000-1000-8000-000000000000 f_varchar:1]", fmt.Sprintf("%v", result[0]))

	// The following mostly tests sanitizeToInternalType()()

	// Accept in-kind using RowData()
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

	// Insert again to exercise compareInternalKnownType

	// Tweak the last field f_ascii, so compareInternalKnownType has a chance to walk through all types before hitting a non-equal value
	preparedQueryParams = []any{int8(1), int16(2), int32(3), int64(4), true, 1.1, 1.2, inf.NewDec(13, 1), sometimeuuid, sometimeuuid, []byte{1, 2}, timeToTest, 10000000, 6, "1", "2", "33"}
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
		?,
		?,
		?,
		?,
		?,
		?,
		?,
		?,
		?,
		?,
		?,
		?,
		?,
		?,
		?,
		?,
		?
)`, preparedQueryParams...).Exec()
	assert.Nil(t, err)

	result, err = s.Query(`SELECT * FROM ks1.t1`).Iter().SliceMap()
	assert.Nil(t, err)
	assert.Equal(t, 2, len(result))
	assert.Equal(t, "map[f_ascii:3 f_bigint:4 f_blob:[1 2] f_bool:true f_counter:0 f_date:6 f_dec:{{false [13]} 1} f_double:1.2 f_float:1.1 f_int:3 f_smallint:2 f_text:2 f_time:10000000 f_timestamp:2015-07-14 00:13:37.476 +0000 UTC f_timeuuid:00000000-0000-1000-8000-000000000000 f_tinyint:1 f_uuid:00000000-0000-1000-8000-000000000000 f_varchar:1]", fmt.Sprintf("%v", result[0]))
	assert.Equal(t, "map[f_ascii:33 f_bigint:4 f_blob:[1 2] f_bool:true f_counter:0 f_date:6 f_dec:{{false [13]} 1} f_double:1.2 f_float:1.1 f_int:3 f_smallint:2 f_text:2 f_time:10000000 f_timestamp:2015-07-14 00:13:37.476 +0000 UTC f_timeuuid:00000000-0000-1000-8000-000000000000 f_tinyint:1 f_uuid:00000000-0000-1000-8000-000000000000 f_varchar:1]", fmt.Sprintf("%v", result[1]))
}

func TestTypesInsert(t *testing.T) {
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

	// Exercise sanitizeToInternalKnownType()

	timeToTest := time.Unix(1436832817, 476000000).UTC()
	sometimeuuid := gocql.TimeUUIDWith(0, 0, []byte{})
	// IMPORTANT: this param list does some counter-intuitive things:
	// - uses inf.Dec instead of float32 (just to test sanitizeToInternalKnownType implicit cast)
	// - uses &inf.Dec instead of inf.Dec (gocql seems to work this way)
	preparedQueryParams := []any{int8(1), int16(2), int32(3), int64(4), true, *inf.NewDec(11, 1), 1.2, inf.NewDec(13, 1), sometimeuuid, sometimeuuid, []byte{1, 2}, timeToTest, 10000000, 6, "1", "2", "3"}
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
		?,
		?,
		?,
		?,
		?,
		?,
		?,
		?,
		?,
		?,
		?,
		?,
		?,
		?,
		?,
		?,
		?
)`, preparedQueryParams...).Exec()
	assert.Nil(t, err)

	result, err := s.Query(`SELECT * FROM ks1.t1`).Iter().SliceMap()
	assert.Nil(t, err)
	assert.Equal(t, 1, len(result))
	assert.Equal(t, "map[f_ascii:3 f_bigint:4 f_blob:[1 2] f_bool:true f_counter:0 f_date:6 f_dec:{{false [13]} 1} f_double:1.2 f_float:1.1 f_int:3 f_smallint:2 f_text:2 f_time:10000000 f_timestamp:2015-07-14 00:13:37.476 +0000 UTC f_timeuuid:00000000-0000-1000-8000-000000000000 f_tinyint:1 f_uuid:00000000-0000-1000-8000-000000000000 f_varchar:1]", fmt.Sprintf("%v", result[0]))

	// The following mostly tests sanitizeToInternalType()

	// Accept in-kind using RowData()
	iter := s.Query(`SELECT * FROM ks1.t1 WHERE
		f_tinyint = ? AND
		f_smallint = ? AND
		f_int = ? AND
		f_bigint = ? AND
		f_bool = ? AND
		f_float = ? AND
		f_double = ? AND
		f_dec = ? AND
		f_timeuuid = ? AND
		f_uuid = ? AND
		f_blob = ? AND
		f_timestamp = ? AND
		f_time = ? AND
		f_date = ? AND
		f_varchar = ? AND
		f_text = ? AND
		f_ascii = ?`, preparedQueryParams...).Iter()
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
}

func TestTypesUpdate(t *testing.T) {
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

	result, err := s.Query(`SELECT * FROM ks1.t1`).Iter().SliceMap()
	assert.Nil(t, err)
	assert.Equal(t, 1, len(result))
	assert.Equal(t, "map[f_ascii:3 f_bigint:4 f_blob:[1 2] f_bool:true f_counter:0 f_date:6 f_dec:{{false [13]} 1} f_double:1.2 f_float:1.1 f_int:3 f_smallint:2 f_text:2 f_time:10000000 f_timestamp:2015-07-14 00:13:37.476 +0000 UTC f_timeuuid:00000000-0000-1000-8000-000000000000 f_tinyint:1 f_uuid:00000000-0000-1000-8000-000000000000 f_varchar:1]", fmt.Sprintf("%v", result[0]))

	// The following mostly tests sanitizeToInternalType()

	preparedQueryParams = []any{int8(1), int16(2), int32(3), int64(4), true, *inf.NewDec(11, 1), 1.2, inf.NewDec(13, 1), sometimeuuid, sometimeuuid, []byte{1, 2}, timeToTest, 10000000, 6, "1", "2", "3"}
	existingRowMap := map[string]any{}
	isApplied, err := s.Query(`UPDATE ks1.t1 SET f_counter = f_counter + 10 WHERE
		f_tinyint = ? AND
		f_smallint = ? AND
		f_int = ? AND
		f_bigint = ? AND
		f_bool = ? AND
		f_float = ? AND
		f_double = ? AND
		f_dec = ? AND
		f_timeuuid = ? AND
		f_uuid = ? AND
		f_blob = ? AND
		f_timestamp = ? AND
		f_time = ? AND
		f_date = ? AND
		f_varchar = ? AND
		f_text = ? AND
		f_ascii = ?`, preparedQueryParams...).MapScanCAS(existingRowMap)
	assert.Nil(t, err)
	assert.True(t, isApplied)

	result, err = s.Query(`SELECT * FROM ks1.t1`).Iter().SliceMap()
	assert.Nil(t, err)
	assert.Equal(t, 1, len(result))
	// counter == 10
	assert.Equal(t, "map[f_ascii:3 f_bigint:4 f_blob:[1 2] f_bool:true f_counter:10 f_date:6 f_dec:{{false [13]} 1} f_double:1.2 f_float:1.1 f_int:3 f_smallint:2 f_text:2 f_time:10000000 f_timestamp:2015-07-14 00:13:37.476 +0000 UTC f_timeuuid:00000000-0000-1000-8000-000000000000 f_tinyint:1 f_uuid:00000000-0000-1000-8000-000000000000 f_varchar:1]", fmt.Sprintf("%v", result[0]))
}
