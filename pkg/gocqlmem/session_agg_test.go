// Tests borrowed from test/unit/org/apache/cassandra/cql3/validation/operations/AggregationTest.java
package gocqlmem

import (
	"fmt"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/inf.v0"
)

func TestFunctions(t *testing.T) {
	s := NewGocqlmemSession()
	assert.Nil(t, s.Query("CREATE KEYSPACE ks1").Exec())
	assert.Nil(t, s.Query("CREATE TABLE ks1.t1 (a int, b int, c double, d decimal, e smallint, f tinyint, primary key (a, b))").Exec())

	// Test with empty table

	assertIterSliceMap(t,
		"[map[avg(b):0 avg(c):0 avg(d):0 avg(e):0 avg(f):0 max(b):<nil> max(c):<nil> max(e):<nil> max(f):<nil> min(b):<nil> min(e):<nil> min(f):<nil> sum(b):0 sum(c):0 sum(d):0 sum(e):0 sum(f):0]]", "", s,
		"SELECT max(b), min(b), sum(b), avg(b), max(c), sum(c), avg(c), sum(d), avg(d), max(e), min(e), sum(e), avg(e), max(f), min(f), sum(f), avg(f) FROM ks1.t1")
	assertIterScan(t,
		`[[<nil>,<nil>,0,0,<nil>,0,0,0,0,<nil>,<nil>,0,0,<nil>,<nil>,0,0]]`, s,
		"SELECT max(b), min(b), sum(b), avg(b), max(c), sum(c), avg(c), sum(d), avg(d), max(e), min(e), sum(e), avg(e), max(f), min(f), sum(f), avg(f) FROM ks1.t1")

	existingRowMap := map[string]interface{}{}

	assertUpserMapScanCas(t, true, s, "INSERT INTO ks1.t1 (a, b, c, d, e, f) VALUES (1, 1, 11.5, 11.5, 1, 1)", existingRowMap)
	assertUpserMapScanCas(t, true, s, "INSERT INTO ks1.t1 (a, b, c, d, e, f) VALUES (1, 2, 9.5, 1.5, 2, 2)", existingRowMap)
	assertUpserMapScanCas(t, true, s, "INSERT INTO ks1.t1 (a, b, c, d, e, f) VALUES (1, 3, 9.0, 2.0, 3, 3)", existingRowMap)

	assertIterScan(t,
		`[[3,1,6,2,11.5,30,10,15,5,3,1,6,2,3,1,6,2]]`, s,
		"SELECT max(b), min(b), sum(b), avg(b) , max(c), sum(c), avg(c), sum(d), avg(d),max(e), min(e), sum(e), avg(e),max(f), min(f), sum(f), avg(f) FROM ks1.t1")

	assertUpserMapScanCas(t, true, s, "INSERT INTO ks1.t1 (a, b, d) VALUES (1, 5, 1.0)", existingRowMap)

	assertIterScan(t, `[[4]]`, s, "SELECT COUNT(*) FROM ks1.t1")
	assertIterScan(t, `[[4]]`, s, "SELECT COUNT(1) FROM ks1.t1")
	assertIterScan(t, `[[4,3,3,3]]`, s, "SELECT COUNT(b), count(c), count(e), count(f) FROM ks1.t1")

	// Makes sure that LIMIT does not affect the result of aggregates

	assertIterScan(t, `[[4,3,3,3]]`, s, "SELECT COUNT(b), count(c), count(e), count(f) FROM ks1.t1 LIMIT 2")
	assertIterScan(t, `[[4,3,3,3]]`, s, "SELECT COUNT(b), count(c), count(e), count(f) FROM ks1.t1 WHERE a = 1 LIMIT 2")
	assertIterScan(t, `[[2.75]]`, s, "SELECT AVG(CAST(b AS double)) FROM ks1.t1")
}

func TestCountStar(t *testing.T) {
	s := NewGocqlmemSession()
	assert.Nil(t, s.Query("CREATE KEYSPACE ks1").Exec())
	assert.Nil(t, s.Query("CREATE TABLE ks1.t1 (a int, b int, c double, primary key (a, b))").Exec())

	// One star count
	assertIterSliceMap(t, "[map[count(*):0]]", "", s, "SELECT count(*) FROM ks1.t1")
	assertIterSliceMap(t, "[map[count(t1.*):0]]", "", s, "SELECT count(t1.*) FROM ks1.t1")

	// One star count with alias
	assertIterSliceMap(t, "[map[cnt:0]]", "", s, "SELECT count(*) AS cnt FROM ks1.t1")
	assertIterSliceMap(t, "[map[cnt:0]]", "", s, "SELECT count(t1.*) AS cnt FROM ks1.t1")

	// Two star counts
	assertIterSliceMap(t, "[map[count(*):0 count(1):0]]", "", s, "SELECT count(*), count(1) FROM ks1.t1")
	assertIterScan(t, "[[0,0]]", s, "SELECT count(*), count(*) FROM ks1.t1")
	assertIterMapScan(t, "[[map[count(*):0]]]", s, "SELECT count(*), count(*) FROM ks1.t1")
	assertIterMapScan(t, "[[map[count(*):0 count(1):0]]]", s, "SELECT count(*), count(1) FROM ks1.t1")
	assertScanner(t, "[[0,0]]", s, "SELECT count(*), count(*) FROM ks1.t1")

	// count(1)
	assertIterSliceMap(t, "[map[count(1):0]]", "", s, "SELECT count(1) FROM ks1.t1")
	assertIterSliceMap(t, "[map[count(1):0]]", "", s, "SELECT count(1) FROM ks1.t1")

	// count(1) with alias
	assertIterSliceMap(t, "[map[cnt:0]]", "", s, "SELECT count(1) AS cnt FROM ks1.t1")

	// Star with other aggregates
	assertIterSliceMap(t, "[map[count(*):0 max(a):<nil>]]", "", s, "SELECT count(*), max(a) FROM ks1.t1")

	existingRowMap := map[string]interface{}{}
	assertUpserMapScanCas(t, true, s, "INSERT INTO ks1.t1 (a, b, c) VALUES (1, 1, 11.5)", existingRowMap)
	assertUpserMapScanCas(t, true, s, "INSERT INTO ks1.t1 (a, b, c) VALUES (1, 2, 9.5)", existingRowMap)
	assertUpserMapScanCas(t, true, s, "INSERT INTO ks1.t1 (a, b, c) VALUES (1, 3, 9.0)", existingRowMap)
	assertUpserMapScanCas(t, true, s, "INSERT INTO ks1.t1 (a, b, c) VALUES (1, 5, 1.0)", existingRowMap)

	assertIterScan(t, "[[4]]", s, "SELECT COUNT(*) FROM ks1.t1")
	assertIterScan(t, "[[4]]", s, "SELECT COUNT(1) FROM ks1.t1")
	assertIterScan(t, "[[5,1,4]]", s, "SELECT max(b), b, COUNT(*) FROM ks1.t1")
	assertIterScan(t, "[[5,4,1]]", s, "SELECT max(b), COUNT(1), b FROM ks1.t1")
	// // Makes sure that LIMIT does not affect the result of aggregates
	assertIterScan(t, "[[5,4,1]]", s, "SELECT max(b), COUNT(1), b FROM ks1.t1 LIMIT 2")
	assertIterScan(t, "[[5,4,1]]", s, "SELECT max(b), COUNT(1), b FROM ks1.t1 WHERE a = 1 LIMIT 2")
}

func TestMaxAggregationDescending(t *testing.T) {
	s := NewGocqlmemSession()
	assert.Nil(t, s.Query("CREATE KEYSPACE ks1").Exec())
	assert.Nil(t, s.Query("CREATE TABLE ks1.t1 (a int, b int, primary key (a, b)) WITH CLUSTERING ORDER BY (b DESC)").Exec())

	existingRowMap := map[string]interface{}{}

	assertUpserMapScanCas(t, true, s, "INSERT INTO ks1.t1 (a, b) VALUES (1, 1000)", existingRowMap)
	assertUpserMapScanCas(t, true, s, "INSERT INTO ks1.t1 (a, b) VALUES (1, 100)", existingRowMap)
	assertUpserMapScanCas(t, true, s, "INSERT INTO ks1.t1 (a, b) VALUES (1, 1)", existingRowMap)

	assertIterScan(t, "[[3,1000]]", s, "SELECT count(b), max(b) as max FROM ks1.t1 WHERE a = 1")

	assertUpserMapScanCas(t, true, s, "INSERT INTO ks1.t1 (a, b) VALUES (2, 4000)", existingRowMap)
	assertUpserMapScanCas(t, true, s, "INSERT INTO ks1.t1 (a, b) VALUES (3, 100)", existingRowMap)
	assertUpserMapScanCas(t, true, s, "INSERT INTO ks1.t1 (a, b) VALUES (4, 0)", existingRowMap)

	assertIterScan(t, "[[6,4000]]", s, "SELECT count(b), max(b) as max FROM ks1.t1")
}

func TestMinAggregationDescending(t *testing.T) {
	s := NewGocqlmemSession()
	assert.Nil(t, s.Query("CREATE KEYSPACE ks1").Exec())
	assert.Nil(t, s.Query("CREATE TABLE ks1.t1 (a int, b int, primary key (a, b)) WITH CLUSTERING ORDER BY (b DESC)").Exec())

	existingRowMap := map[string]interface{}{}

	assertUpserMapScanCas(t, true, s, "INSERT INTO ks1.t1 (a, b) VALUES (1, 1000)", existingRowMap)
	assertUpserMapScanCas(t, true, s, "INSERT INTO ks1.t1 (a, b) VALUES (1, 100)", existingRowMap)
	assertUpserMapScanCas(t, true, s, "INSERT INTO ks1.t1 (a, b) VALUES (1, 1)", existingRowMap)

	assertIterScan(t, "[[3,1]]", s, "SELECT count(b), min(b) as min FROM ks1.t1 WHERE a = 1")

	assertUpserMapScanCas(t, true, s, "INSERT INTO ks1.t1 (a, b) VALUES (2, 4000)", existingRowMap)
	assertUpserMapScanCas(t, true, s, "INSERT INTO ks1.t1 (a, b) VALUES (3, 100)", existingRowMap)
	assertUpserMapScanCas(t, true, s, "INSERT INTO ks1.t1 (a, b) VALUES (4, 0)", existingRowMap)

	assertIterScan(t, "[[6,0]]", s, "SELECT count(b), min(b) as min FROM ks1.t1")
}

func TestMaxAggregationAscending(t *testing.T) {
	s := NewGocqlmemSession()
	assert.Nil(t, s.Query("CREATE KEYSPACE ks1").Exec())
	assert.Nil(t, s.Query("CREATE TABLE ks1.t1 (a int, b int, primary key (a, b)) WITH CLUSTERING ORDER BY (b ASC)").Exec())

	existingRowMap := map[string]interface{}{}

	assertUpserMapScanCas(t, true, s, "INSERT INTO ks1.t1 (a, b) VALUES (1, 1000)", existingRowMap)
	assertUpserMapScanCas(t, true, s, "INSERT INTO ks1.t1 (a, b) VALUES (1, 100)", existingRowMap)
	assertUpserMapScanCas(t, true, s, "INSERT INTO ks1.t1 (a, b) VALUES (1, 1)", existingRowMap)

	assertIterScan(t, "[[3,1000]]", s, "SELECT count(b), max(b) as max FROM ks1.t1 WHERE a = 1")

	assertUpserMapScanCas(t, true, s, "INSERT INTO ks1.t1 (a, b) VALUES (2, 4000)", existingRowMap)
	assertUpserMapScanCas(t, true, s, "INSERT INTO ks1.t1 (a, b) VALUES (3, 100)", existingRowMap)
	assertUpserMapScanCas(t, true, s, "INSERT INTO ks1.t1 (a, b) VALUES (4, 0)", existingRowMap)

	assertIterScan(t, "[[6,4000]]", s, "SELECT count(b), max(b) as max FROM ks1.t1")
}

func TestMinAggregationAscending(t *testing.T) {
	s := NewGocqlmemSession()
	assert.Nil(t, s.Query("CREATE KEYSPACE ks1").Exec())
	assert.Nil(t, s.Query("CREATE TABLE ks1.t1 (a int, b int, primary key (a, b)) WITH CLUSTERING ORDER BY (b ASC)").Exec())

	existingRowMap := map[string]interface{}{}

	assertUpserMapScanCas(t, true, s, "INSERT INTO ks1.t1 (a, b) VALUES (1, 1000)", existingRowMap)
	assertUpserMapScanCas(t, true, s, "INSERT INTO ks1.t1 (a, b) VALUES (1, 100)", existingRowMap)
	assertUpserMapScanCas(t, true, s, "INSERT INTO ks1.t1 (a, b) VALUES (1, 1)", existingRowMap)

	assertIterScan(t, "[[3,1]]", s, "SELECT count(b), min(b) as min FROM ks1.t1 WHERE a = 1")

	assertUpserMapScanCas(t, true, s, "INSERT INTO ks1.t1 (a, b) VALUES (2, 4000)", existingRowMap)
	assertUpserMapScanCas(t, true, s, "INSERT INTO ks1.t1 (a, b) VALUES (3, 100)", existingRowMap)
	assertUpserMapScanCas(t, true, s, "INSERT INTO ks1.t1 (a, b) VALUES (4, 0)", existingRowMap)

	assertIterScan(t, "[[6,0]]", s, "SELECT count(b), min(b) as min FROM ks1.t1")
}

func TestAggregateWithColumns(t *testing.T) {
	s := NewGocqlmemSession()
	assert.Nil(t, s.Query("CREATE KEYSPACE ks1").Exec())
	assert.Nil(t, s.Query("CREATE TABLE ks1.t1 (a int, b int, c int, primary key (a, b))").Exec())

	assertIterSliceMap(t, "[map[b:<nil> count(b):0 first:<nil> max:<nil>]]", "", s, "SELECT count(b), max(b) as max, b, c as first FROM ks1.t1")
	assertIterScan(t, "[[0,<nil>,<nil>,<nil>]]", s, "SELECT count(b), max(b) as max, b, c as first FROM ks1.t1")

	existingRowMap := map[string]interface{}{}

	assertUpserMapScanCas(t, true, s, "INSERT INTO ks1.t1 (a, b, c) VALUES (1, 2, null)", existingRowMap)
	assertUpserMapScanCas(t, true, s, "INSERT INTO ks1.t1 (a, b, c) VALUES (2, 4, 6)", existingRowMap)
	assertUpserMapScanCas(t, true, s, "INSERT INTO ks1.t1 (a, b, c) VALUES (4, 8, 12)", existingRowMap)

	assertIterScan(t, "[[3,8,2,<nil>]]", s, "SELECT count(b), max(b) as max, b, c as first FROM ks1.t1")
}

func TestAggregateOnCounters(t *testing.T) {
	s := NewGocqlmemSession()
	assert.Nil(t, s.Query("CREATE KEYSPACE ks1").Exec())
	assert.Nil(t, s.Query("CREATE TABLE ks1.t1 (a int, b counter, primary key (a))").Exec())

	assertIterSliceMap(t, "[map[b:<nil> count(b):0 max:<nil>]]", "", s, "SELECT count(b), max(b) as max, b FROM ks1.t1")
	assertIterScan(t, "[[0,<nil>,<nil>,<nil>]]", s, "SELECT count(b), max(b) as max, b, c as first FROM ks1.t1")

	existingRowMap := map[string]interface{}{}

	assertUpserMapScanCas(t, true, s, "UPDATE ks1.t1 SET b = b + 1 WHERE a = 1", existingRowMap)
	assertIterScan(t, "[[1,1,1,1,1]]", s, "SELECT count(b), max(b) as max, min(b) as min, avg(b) as avg, sum(b) as sum FROM ks1.t1")
	assertUpserMapScanCas(t, true, s, "UPDATE ks1.t1 SET b = b + 1 WHERE a = 1", existingRowMap)
	assertIterScan(t, "[[1,2,2,2,2]]", s, "SELECT count(b), max(b) as max, min(b) as min, avg(b) as avg, sum(b) as sum FROM ks1.t1")

	assertUpserMapScanCas(t, true, s, "UPDATE ks1.t1 SET b = b + 2 WHERE a = 1", existingRowMap)
	assertIterScan(t, "[[1,4,4,4,4]]", s, "SELECT count(b), max(b) as max, min(b) as min, avg(b) as avg, sum(b) as sum FROM ks1.t1")

	assertUpserMapScanCas(t, true, s, "UPDATE ks1.t1 SET b = b - 2 WHERE a = 1", existingRowMap)
	assertIterScan(t, "[[1,2,2,2,2]]", s, "SELECT count(b), max(b) as max, min(b) as min, avg(b) as avg, sum(b) as sum FROM ks1.t1")

	assertUpserMapScanCas(t, true, s, "UPDATE ks1.t1 SET b = b + 1 WHERE a = 2", existingRowMap)
	assertUpserMapScanCas(t, true, s, "UPDATE ks1.t1 SET b = b + 1 WHERE a = 2", existingRowMap)
	assertUpserMapScanCas(t, true, s, "UPDATE ks1.t1 SET b = b + 2 WHERE a = 2", existingRowMap)
	assertIterScan(t, "[[2,4,2,3,6]]", s, "SELECT count(b), max(b) as max, min(b) as min, avg(b) as avg, sum(b) as sum FROM ks1.t1")
}

func TestAggregateWithSetsListMapsTuplesUdtsFunctionsTtlSchemachange(t *testing.T) {
	// Not supported
}

func TestInvalidCalls(t *testing.T) {
	s := NewGocqlmemSession()
	assert.Nil(t, s.Query("CREATE KEYSPACE ks1").Exec())
	assert.Nil(t, s.Query("CREATE TABLE ks1.t1 (a int, b int, c int, primary key (a, b))").Exec())

	existingRowMap := map[string]interface{}{}

	assertUpserMapScanCas(t, true, s, "INSERT INTO ks1.t1 (a, b, c) VALUES (1, 1, 10)", existingRowMap)
	assertUpserMapScanCas(t, true, s, "INSERT INTO ks1.t1 (a, b, c) VALUES (1, 2, 9)", existingRowMap)
	assertUpserMapScanCas(t, true, s, "INSERT INTO ks1.t1 (a, b, c) VALUES (1, 3, 8)", existingRowMap)

	iter := s.Query("SELECT max(b), max(c) FROM ks1.t1 WHERE max(a) = 1").Iter()
	assert.Contains(t, iter.Err().Error(), "cannot evaluate where expression: cannot evaluate max(), context aggregate not enabled")

	iter = s.Query("SELECT max(b), max(c) FROM ks1.t1 WHERE max(a)").Iter()
	assert.Contains(t, iter.Err().Error(), "cannot evaluate where expression: cannot evaluate max(), context aggregate not enabled")
}

func TestReversedType(t *testing.T) {
	s := NewGocqlmemSession()
	assert.Nil(t, s.Query("CREATE KEYSPACE ks1").Exec())
	assert.Nil(t, s.Query("CREATE TABLE ks1.t1 (a int, b int, c int, primary key (a, b)) WITH CLUSTERING ORDER BY (b DESC)").Exec())

	existingRowMap := map[string]interface{}{}

	assertUpserMapScanCas(t, true, s, "INSERT INTO ks1.t1 (a, b, c) VALUES (1, 1, 10)", existingRowMap)
	assertUpserMapScanCas(t, true, s, "INSERT INTO ks1.t1 (a, b, c) VALUES (1, 2, 9)", existingRowMap)
	assertUpserMapScanCas(t, true, s, "INSERT INTO ks1.t1 (a, b, c) VALUES (1, 3, 8)", existingRowMap)
	assertUpserMapScanCas(t, true, s, "INSERT INTO ks1.t1 (a, b, c) VALUES (1, 4, 7)", existingRowMap)
	assertIterScan(t, "[[9,7,8]]", s, "SELECT max(c), min(c), avg(c) FROM ks1.t1 WHERE a = 1 AND b > 1")
}

func TestArithmeticCorrectness(t *testing.T) {
	s := NewGocqlmemSession()
	assert.Nil(t, s.Query("CREATE KEYSPACE ks1").Exec())
	assert.Nil(t, s.Query("CREATE TABLE ks1.t1 (bucket int primary key, val decimal)").Exec())

	existingRowMap := map[string]interface{}{}

	assertUpserMapScanCas(t, true, s, "INSERT INTO ks1.t1 (bucket, val) values (1, 0.25)", existingRowMap)
	assertUpserMapScanCas(t, true, s, "INSERT INTO ks1.t1 (bucket, val) values (2, 0.25)", existingRowMap)
	assertUpserMapScanCas(t, true, s, "INSERT INTO ks1.t1 (bucket, val) values (3, 0.5)", existingRowMap)
	assertIterScan(t, "[[0.3333333333333333]]", s, "SELECT avg(val) FROM ks1.t1 where bucket in (1, 2, 3)")
}

func TestAggregatesWithoutOverflow(t *testing.T) {
	s := NewGocqlmemSession()
	assert.Nil(t, s.Query("CREATE KEYSPACE ks1").Exec())
	assert.Nil(t, s.Query("CREATE TABLE ks1.t1 (bucket int primary key, v1 tinyint, v2 smallint, v3 int, v4 bigint, v5 varint)").Exec())

	existingRowMap := map[string]interface{}{}

	for _, i := range []int{1, 2, 3} {
		assertUpserMapScanCas(t, true, s, fmt.Sprintf("INSERT INTO ks1.t1 (bucket, v1, v2, v3, v4, v5) values (%d, %d, %d, %d, %d, %d)",
			i, math.MaxInt8/3+i, math.MaxInt16/3+i, math.MaxInt32/3+i, math.MaxInt64/3+i, math.MaxInt64/3+i), existingRowMap)
	}
	result := fmt.Sprintf("[[%d,%d,%d,%d,%d]]", math.MaxInt8/3+2, math.MaxInt16/3+2, math.MaxInt32/3+2, math.MaxInt64/3+2, math.MaxInt64/3+2)
	assertIterScan(t, result, s, "SELECT avg(v1), avg(v2), avg(v3), avg(v4), avg(v5) from ks1.t1 where bucket in (1, 2, 3)")

	for _, i := range []int{1, 2, 3} {
		assertUpserMapScanCas(t, true, s, fmt.Sprintf("INSERT INTO ks1.t1 (bucket, v1, v2, v3, v4, v5) values (%d, %d, %d, %d, %d, %d)",
			i+3, 100+i, 100+i, 100+i, i+100, i+100), existingRowMap)
	}
	assertIterScan(t, "[[102,102,102,102,102]]", s, "SELECT avg(v1), avg(v2), avg(v3), avg(v4), avg(v5) from ks1.t1 where bucket in (4, 5, 6)")
}

func TestAggregatesOverflow(t *testing.T) {
	s := NewGocqlmemSession()
	assert.Nil(t, s.Query("CREATE KEYSPACE ks1").Exec())
	assert.Nil(t, s.Query("CREATE TABLE ks1.t1 (bucket int primary key, v1 tinyint, v2 smallint, v3 int, v4 bigint, v5 varint)").Exec())

	existingRowMap := map[string]interface{}{}

	for _, i := range []int{1, 2, 3} {
		assertUpserMapScanCas(t, true, s, fmt.Sprintf("INSERT INTO ks1.t1 (bucket, v1, v2, v3, v4, v5) values (%d, %d, %d, %d, %d, %d)",
			i, math.MaxInt8, math.MaxInt16, math.MaxInt32, math.MaxInt64, math.MaxInt64), existingRowMap)
	}
	result := fmt.Sprintf("[[%d,%d,%d,%d,%d]]", math.MaxInt8, math.MaxInt16, math.MaxInt32, math.MaxInt64, math.MaxInt64)
	assertIterScan(t, result, s, "SELECT avg(v1), avg(v2), avg(v3), avg(v4), avg(v5) from ks1.t1 where bucket in (1, 2, 3)")

	assert.Nil(t, s.Query("TRUNCATE ks1.t1").Exec())

	for _, i := range []int{1, 2, 3} {
		assertUpserMapScanCas(t, true, s, fmt.Sprintf("INSERT INTO ks1.t1 (bucket, v1, v2, v3, v4, v5) values (%d, %d, %d, %d, %d, %d)",
			i, math.MinInt8, math.MinInt16, math.MinInt32, math.MinInt64, math.MinInt64), existingRowMap)
	}
	result = fmt.Sprintf("[[%d,%d,%d,%d,%d]]", math.MinInt8, math.MinInt16, math.MinInt32, math.MinInt64, math.MinInt64)
	assertIterScan(t, result, s, "SELECT avg(v1), avg(v2), avg(v3), avg(v4), avg(v5) from ks1.t1 where bucket in (1, 2, 3)")

}

func TestDoubleAggregatesPrecision(t *testing.T) {
	s := NewGocqlmemSession()
	assert.Nil(t, s.Query("CREATE KEYSPACE ks1").Exec())
	assert.Nil(t, s.Query("CREATE TABLE ks1.t1 (bucket int primary key, v1 float, v2 double, v3 decimal)").Exec())

	existingRowMap := map[string]interface{}{}

	maxPlus2 := new(inf.Dec).Add(float64ToDecNoCheck(float64(math.MaxFloat64)), float64ToDecNoCheck(float64(2.0)))
	for _, i := range []int{1, 2, 3} {
		assertUpserMapScanCas(t, true, s, fmt.Sprintf("INSERT INTO ks1.t1 (bucket, v1, v2, v3) values (%d, %f, %f, %s)", i, math.MaxFloat32, math.MaxFloat64, maxPlus2.String()), existingRowMap)
	}
	// Yeah, we choke on Float64 overflow, so Inf(1). Should we fix it really?
	assertIterScan(t, "[[340282346638528860000000000000000000000,+Inf,179769313486231570000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000002]]", s,
		"SELECT avg(v1), avg(v2), avg(v3) from ks1.t1 where bucket in (1, 2, 3)")

	assertUpserMapScanCas(t, true, s, fmt.Sprintf("INSERT INTO ks1.t1 (bucket, v1, v2, v3) values (%d, %f, %f, %s)", 4, float32(100.10), float64(100.10), float64ToDecNoCheck(float64(100.10)).String()), existingRowMap)
	assertUpserMapScanCas(t, true, s, fmt.Sprintf("INSERT INTO ks1.t1 (bucket, v1, v2, v3) values (%d, %f, %f, %s)", 5, float32(110.11), float64(110.11), float64ToDecNoCheck(float64(110.11)).String()), existingRowMap)
	assertUpserMapScanCas(t, true, s, fmt.Sprintf("INSERT INTO ks1.t1 (bucket, v1, v2, v3) values (%d, %f, %f, %s)", 6, float32(120.12), float64(120.12), float64ToDecNoCheck(float64(120.12)).String()), existingRowMap)

	assertIterScan(t, "[[110.11000066666666,110.11,110.11]]", s, "SELECT avg(v1), avg(v2), avg(v3) from ks1.t1 where bucket in (4,5,6)")
}

func TestNan(t *testing.T) {
	s := NewGocqlmemSession()
	assert.Nil(t, s.Query("CREATE KEYSPACE ks1").Exec())
	assert.Nil(t, s.Query("CREATE TABLE ks1.t1 (bucket int primary key, v1 float, v2 double)").Exec())

	// Yeah, we choke on Java/Golang NaN type. Should we fix it really?
	err := s.Query(fmt.Sprintf("INSERT INTO ks1.t1 (bucket, v1, v2) values (%d, %f, %f)", 1, math.NaN(), math.NaN())).Exec()
	assert.Contains(t, err.Error(), "cannot evaluate ident expression 'NaN'")
}

func TestInfinity(t *testing.T) {
	s := NewGocqlmemSession()
	assert.Nil(t, s.Query("CREATE KEYSPACE ks1").Exec())
	assert.Nil(t, s.Query("CREATE TABLE ks1.t1 (bucket int primary key, v1 float, v2 double)").Exec())

	// Yeah, we choke on Java/Golang NaN type. Should we fix it really?
	err := s.Query(fmt.Sprintf("INSERT INTO ks1.t1 (bucket, v1, v2) values (%d, %f, %f)", 5, math.Inf(1), math.Inf(1))).Exec()
	assert.Contains(t, err.Error(), "value list length (1) should match column list length (3)") // Parser simply ignores "+Inf"
}

func TestSumPrecision(t *testing.T) {
	s := NewGocqlmemSession()
	assert.Nil(t, s.Query("CREATE KEYSPACE ks1").Exec())
	assert.Nil(t, s.Query("CREATE TABLE ks1.t1 (bucket int primary key, v1 float, v2 double, v3 decimal)").Exec())

	existingRowMap := map[string]interface{}{}

	for i := range 17 {
		divIby10 := new(inf.Dec).QuoExact(inf.NewDec(int64(i+1), 0), float64ToDecNoCheck(float64(10.0)))
		assertUpserMapScanCas(t, true, s, fmt.Sprintf("INSERT INTO ks1.t1 (bucket, v1, v2, v3) values (%d, %f, %f, %s)", i+1, float32(i+1)/10.0, float64(i+1)/10.0, divIby10.String()), existingRowMap)
	}
	assertIterScan(t, "[[15.299999999999999,15.299999999999999,15.3]]", s, "SELECT sum(v1), sum(v2), sum(v3) from ks1.t1")
}
