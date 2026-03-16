package gocqlmem

import (
	"fmt"
	"go/ast"
	"strings"
	"testing"

	gocql "github.com/apache/cassandra-gocql-driver/v2"
	"github.com/stretchr/testify/assert"
)

func TestCreateKeyspace(t *testing.T) {
	cmds, err := ParseCommands(`CREATE KEYSPACE IF NOT EXISTS ks1 WITH REPLICATION { 'class': 'NetworkTopologyStrategy', 'datacenter1': 3, 'datacenter2': 3}`, nil)
	assert.Nil(t, err)
	cmd, ok := cmds[0].(*CommandCreateKeyspace)
	assert.True(t, ok)

	assert.True(t, cmd.IfNotExists)
	assert.Equal(t, "ks1", cmd.KeyspaceName)

	assert.Equal(t, "class", cmd.WithReplication[0].K)
	assert.Equal(t, LexemStringLiteral, cmd.WithReplication[0].V.T)
	assert.Equal(t, "NetworkTopologyStrategy", cmd.WithReplication[0].V.V)

	assert.Equal(t, "datacenter1", cmd.WithReplication[1].K)
	assert.Equal(t, LexemNumberLiteral, cmd.WithReplication[1].V.T)
	assert.Equal(t, "3", cmd.WithReplication[1].V.V)
}

func TestUseKeyspace(t *testing.T) {
	cmds, err := ParseCommands(`USE ks1; select f1 FROM t1;`, nil)
	assert.Nil(t, err)

	cmd, ok := cmds[0].(*CommandUseKeyspace)
	assert.True(t, ok)
	assert.Equal(t, "ks1", cmd.KeyspaceName)

	_, ok = cmds[1].(*CommandSelect)
	assert.True(t, ok)
	assert.Equal(t, "ks1", cmds[1].GetCtxKeyspace())

	cmds, err = ParseCommands(`select f1 FROM t1;`, nil)
	assert.Contains(t, err.Error(), "cannot detect keyspace for command 0")
}

func TestDropKeyspace(t *testing.T) {
	cmds, err := ParseCommands(`DROP KEYSPACE IF EXISTS ks1;`, nil)
	assert.Nil(t, err)

	cmd, ok := cmds[0].(*CommandDropKeyspace)
	assert.True(t, ok)
	assert.True(t, cmd.IfExists)
	assert.Equal(t, "ks1", cmd.KeyspaceName)
}

func TestCreateTable(t *testing.T) {
	cmds, err := ParseCommands(`CREATE TABLE IF NOT EXISTS ks1.t1 (f1 TEXT, f2 TIMESTAMP, f3 BIGINT, f4 BIGINT, PRIMARY KEY((f1,f2), f3, f4) ) WITH CLUSTERING ORDER BY (f3 ASC, f4 DESC)`, nil)
	assert.Nil(t, err)
	cr, ok := cmds[0].(*CommandCreateTable)
	assert.True(t, ok)

	assert.True(t, cr.IfNotExists)
	assert.Equal(t, "ks1", cr.CtxKeyspace)
	assert.Equal(t, "t1", cr.TableName)

	assert.Equal(t, "f1", cr.ColumnDefs[0].Name)
	assert.Equal(t, gocql.TypeText, cr.ColumnDefs[0].ColumnType)
	assert.Equal(t, "f2", cr.ColumnDefs[1].Name)
	assert.Equal(t, gocql.TypeTimestamp, cr.ColumnDefs[1].ColumnType)

	assert.Equal(t, "f1", cr.PartitionKeyColumns[0])
	assert.Equal(t, "f2", cr.PartitionKeyColumns[1])

	assert.Equal(t, "f3", cr.ClusteringKeyColumns[0])

	assert.Equal(t, "f3", cr.ClusteringOrderBy[0].FieldName)
	assert.Equal(t, ClusteringOrderAsc, cr.ClusteringOrderBy[0].ClusteringOrder)

	assert.Equal(t, "f4", cr.ClusteringOrderBy[1].FieldName)
	assert.Equal(t, ClusteringOrderDesc, cr.ClusteringOrderBy[1].ClusteringOrder)

	cmds, err = ParseCommands(`USE ks1;CREATE  TABLE   t1  ( f1  TEXT,  PRIMARY  KEY ( ( f1 ) )  ) `, nil)
	assert.Nil(t, err)
	cr, ok = cmds[1].(*CommandCreateTable)
	assert.True(t, ok)

	assert.False(t, cr.IfNotExists)
	assert.Equal(t, "ks1", cr.CtxKeyspace)
	assert.Equal(t, "t1", cr.TableName)

	assert.Equal(t, "f1", cr.ColumnDefs[0].Name)
	assert.Equal(t, gocql.TypeText, cr.ColumnDefs[0].ColumnType)

	assert.Equal(t, "f1", cr.PartitionKeyColumns[0])

	cmds, err = ParseCommands(`CREATE  TABLE   ks1.t1  ( f1  TEXT, f2 BIGINT,  PRIMARY  KEY ( f1, f2 )  ) `, nil)
	assert.Nil(t, err)
	cr, ok = cmds[0].(*CommandCreateTable)
	assert.True(t, ok)

	assert.False(t, cr.IfNotExists)
	assert.Equal(t, "ks1", cr.CtxKeyspace)
	assert.Equal(t, "t1", cr.TableName)

	assert.Equal(t, "f1", cr.ColumnDefs[0].Name)
	assert.Equal(t, gocql.TypeText, cr.ColumnDefs[0].ColumnType)

	assert.Equal(t, "f1", cr.PartitionKeyColumns[0])
	assert.Equal(t, "f2", cr.ClusteringKeyColumns[0])

	cmds, err = ParseCommands(`CREATE  TABLE   ks1.t1  ( f1  TEXT ) `, nil)
	assert.Contains(t, err.Error(), "expected PRIMARY KEY")

	cmds, err = ParseCommands(`CREATE TABLE ks1.t1 ( PRIMARY KEY() )`, nil)
	assert.Contains(t, err.Error(), "cannot parse CREATE TABLE with empty columnn def list")

	cmds, err = ParseCommands(`CREATE TABLE ks1.t1 ( f1 TEXT, PRIMARY KEY() )`, nil)
	assert.Contains(t, err.Error(), "cannot parse CREATE TABLE with empty partition column list")

	cmds, err = ParseCommands(`CREATE TABLE IF NOT EXISTS ks1.t1 (f1 TEXT, PRIMARY KEY(f1, f2) ) WITH CLUSTERING ORDER BY (f1 ASC)`, nil)
	assert.Contains(t, err.Error(), "clustering order field f1 specified, but it's not among clustering keys")

	cmds, err = ParseCommands(`CREATE TABLE IF NOT EXISTS ks1.t1 (f1 TEXT, PRIMARY KEY(f2) )`, nil)
	assert.Contains(t, err.Error(), "partition key f2 not found in column definitions")

	cmds, err = ParseCommands(`CREATE TABLE IF NOT EXISTS ks1.t1 (f1 TEXT, PRIMARY KEY(f1, f2) )`, nil)
	assert.Contains(t, err.Error(), "clustering key f2 not found in column definitions")

	cmds, err = ParseCommands(`CREATE TABLE IF NOT EXISTS ks1.t1 (f1 TEXT, PRIMARY KEY(f1, f1) )`, nil)
	assert.Contains(t, err.Error(), "clustering key f1 duplication")
}

func TestTruncateTable(t *testing.T) {
	cmds, err := ParseCommands(`USE ks1;TRUNCATE t1;TRUNCATE ks2.t1`, nil)
	assert.Nil(t, err)
	cmd, ok := cmds[1].(*CommandTruncateTable)
	assert.True(t, ok)
	assert.Equal(t, "ks1", cmd.CtxKeyspace)
	assert.Equal(t, "t1", cmd.TableName)
	cmd, ok = cmds[2].(*CommandTruncateTable)
	assert.True(t, ok)
	assert.Equal(t, "ks2", cmd.CtxKeyspace)
	assert.Equal(t, "t1", cmd.TableName)
}

func TestDropTable(t *testing.T) {
	cmds, err := ParseCommands(`USE ks1;DROP TABLE IF EXISTS t1;DROP TABLE ks2.t1`, nil)
	assert.Nil(t, err)
	cmd, ok := cmds[1].(*CommandDropTable)
	assert.True(t, ok)
	assert.True(t, cmd.IfExists)
	assert.Equal(t, "ks1", cmd.CtxKeyspace)
	assert.Equal(t, "t1", cmd.TableName)
	cmd, ok = cmds[2].(*CommandDropTable)
	assert.True(t, ok)
	assert.False(t, cmd.IfExists)
	assert.Equal(t, "ks2", cmd.CtxKeyspace)
	assert.Equal(t, "t1", cmd.TableName)
}

func TestSelect(t *testing.T) {
	cmds, err := ParseCommands(`SELECT sum(f1) - ' FROM ', f2 + ' , ', (f3 * 3 = 9) = TRUE AND 1=NULL OR FALSE, cast(f3 as int) FROM ks1.t1 WHERE f1 / 2 = 10 and (f2 = 'a"') ORDER BY f1 asc, f2 desc LIMIT 10`, nil)
	assert.Nil(t, err)
	cmd, ok := cmds[0].(*CommandSelect)
	assert.True(t, ok)

	assert.Equal(t, "ks1", cmd.CtxKeyspace)

	assert.Equal(t, "sum", cmd.SelectExpLexems[0][0].V)
	assert.Equal(t, "(", cmd.SelectExpLexems[0][1].V)
	assert.Equal(t, "f1", cmd.SelectExpLexems[0][2].V)
	assert.Equal(t, ")", cmd.SelectExpLexems[0][3].V)
	assert.Equal(t, "-", cmd.SelectExpLexems[0][4].V)
	assert.Equal(t, " FROM ", cmd.SelectExpLexems[0][5].V)

	assert.Equal(t, "f2", cmd.SelectExpLexems[1][0].V)
	assert.Equal(t, "+", cmd.SelectExpLexems[1][1].V)
	assert.Equal(t, " , ", cmd.SelectExpLexems[1][2].V)

	assert.Equal(t, "(", cmd.SelectExpLexems[2][0].V)
	assert.Equal(t, "f3", cmd.SelectExpLexems[2][1].V)
	assert.Equal(t, "*", cmd.SelectExpLexems[2][2].V)
	assert.Equal(t, "3", cmd.SelectExpLexems[2][3].V)
	assert.Equal(t, "==", cmd.SelectExpLexems[2][4].V)
	assert.Equal(t, "9", cmd.SelectExpLexems[2][5].V)
	assert.Equal(t, ")", cmd.SelectExpLexems[2][6].V)
	assert.Equal(t, "==", cmd.SelectExpLexems[2][7].V)
	assert.Equal(t, "TRUE", cmd.SelectExpLexems[2][8].V)
	assert.Equal(t, "&&", cmd.SelectExpLexems[2][9].V)
	assert.Equal(t, "1", cmd.SelectExpLexems[2][10].V)
	assert.Equal(t, "==", cmd.SelectExpLexems[2][11].V)
	assert.Equal(t, "NULL", cmd.SelectExpLexems[2][12].V)
	assert.Equal(t, LexemNull, cmd.SelectExpLexems[2][12].T)
	assert.Equal(t, "||", cmd.SelectExpLexems[2][13].V)
	assert.Equal(t, "FALSE", cmd.SelectExpLexems[2][14].V)

	assert.Equal(t, "cast", cmd.SelectExpLexems[3][0].V)
	assert.Equal(t, "(", cmd.SelectExpLexems[3][1].V)
	assert.Equal(t, "f3", cmd.SelectExpLexems[3][2].V)
	assert.Equal(t, ",", cmd.SelectExpLexems[3][3].V)
	assert.Equal(t, "int", cmd.SelectExpLexems[3][4].V)
	assert.Equal(t, ")", cmd.SelectExpLexems[3][5].V)

	assert.Equal(t, "t1", cmd.TableName)

	assert.Equal(t, "f1", cmd.WhereExpLexems[0].V)
	assert.Equal(t, "/", cmd.WhereExpLexems[1].V)
	assert.Equal(t, "2", cmd.WhereExpLexems[2].V)
	assert.Equal(t, "==", cmd.WhereExpLexems[3].V)
	assert.Equal(t, "10", cmd.WhereExpLexems[4].V)
	assert.Equal(t, "&&", cmd.WhereExpLexems[5].V)
	assert.Equal(t, "(", cmd.WhereExpLexems[6].V)
	assert.Equal(t, "f2", cmd.WhereExpLexems[7].V)
	assert.Equal(t, "==", cmd.WhereExpLexems[8].V)
	assert.Equal(t, `a"`, cmd.WhereExpLexems[9].V)
	assert.Equal(t, ")", cmd.WhereExpLexems[10].V)

	assert.Equal(t, "f1", cmd.OrderByFields[0].FieldName)
	assert.Equal(t, ClusteringOrderAsc, cmd.OrderByFields[0].ClusteringOrder)
	assert.Equal(t, "f2", cmd.OrderByFields[1].FieldName)
	assert.Equal(t, ClusteringOrderDesc, cmd.OrderByFields[1].ClusteringOrder)

	assert.Equal(t, "10", cmd.Limit.V)

	cmds, err = ParseCommands(`SELECT f1 AS ff11 FROM ks1.t1; ; ;SELECT f2 FROM ks1.t2; ; ;`, nil)
	assert.Nil(t, err)

	assert.Equal(t, 2, len(cmds))

	cmd, ok = cmds[0].(*CommandSelect)
	assert.True(t, ok)

	assert.Equal(t, "ks1", cmd.CtxKeyspace)
	assert.Equal(t, "f1", cmd.SelectExpLexems[0][0].V)
	assert.Equal(t, LexemAs, cmd.SelectExpLexems[0][1].T)
	assert.Equal(t, "ff11", cmd.SelectExpLexems[0][2].V)

	assert.Equal(t, "t1", cmd.TableName)

	cmd, ok = cmds[1].(*CommandSelect)
	assert.True(t, ok)
	assert.Equal(t, "ks1", cmd.CtxKeyspace)
	assert.Equal(t, "f2", cmd.SelectExpLexems[0][0].V)
	assert.Equal(t, "t2", cmd.TableName)

	cmds, err = ParseCommands(`SELECT FROM ks1.t1`, nil)
	assert.Contains(t, err.Error(), "expected select expressions")

	cmds, err = ParseCommands(`SELECT f1 FROM t1;bla`, nil)
	assert.Contains(t, err.Error(), "unexpected command text, a semicolon expected")
}

func TestAs(t *testing.T) {
	cmds, err := ParseCommands(`SELECT max(cast(f1 * 32.0 as int)) as f11, f2 FROM ks1.t1 WHERE cast(f1 as text) = '1'`, nil)
	assert.Nil(t, err)
	cmd, ok := cmds[0].(*CommandSelect)
	assert.True(t, ok)

	assert.Equal(t, "ks1", cmd.CtxKeyspace)

	assert.Equal(t, "max", cmd.SelectExpLexems[0][0].V)
	assert.Equal(t, "(", cmd.SelectExpLexems[0][1].V)
	assert.Equal(t, "cast", cmd.SelectExpLexems[0][2].V)
	assert.Equal(t, "(", cmd.SelectExpLexems[0][3].V)
	assert.Equal(t, "f1", cmd.SelectExpLexems[0][4].V)
	assert.Equal(t, "*", cmd.SelectExpLexems[0][5].V)
	assert.Equal(t, "32.0", cmd.SelectExpLexems[0][6].V)
	assert.Equal(t, ",", cmd.SelectExpLexems[0][7].V)
	assert.Equal(t, "int", cmd.SelectExpLexems[0][8].V)
	assert.Equal(t, ")", cmd.SelectExpLexems[0][9].V)
	assert.Equal(t, ")", cmd.SelectExpLexems[0][10].V)
	assert.Equal(t, "AS", cmd.SelectExpLexems[0][11].V)
	assert.Equal(t, "f11", cmd.SelectExpLexems[0][12].V)
	assert.Equal(t, "f2", cmd.SelectExpLexems[1][0].V)

	assert.Equal(t, "t1", cmd.TableName)

	assert.Equal(t, "cast", cmd.WhereExpLexems[0].V)
	assert.Equal(t, "(", cmd.WhereExpLexems[1].V)
	assert.Equal(t, "f1", cmd.WhereExpLexems[2].V)
	assert.Equal(t, ",", cmd.WhereExpLexems[3].V)
	assert.Equal(t, "text", cmd.WhereExpLexems[4].V)
	assert.Equal(t, ")", cmd.WhereExpLexems[5].V)
}

func lexemSliceToString(lexems []*Lexem) string {
	sb := strings.Builder{}
	for i, l := range lexems {
		if i != 0 {
			sb.WriteString(" ")
		}
		sb.WriteString(l.V)
	}
	return sb.String()
}

func TestInNotIn(t *testing.T) {
	cmds, err := ParseCommands(`SELECT f1 * 2 IN (3+4,5+6), f1 * 7 NOT IN (?, ?) FROM ks1.t1 WHERE f1 IN ? AND f2 NOT IN (?, ?)`, []any{8, 9, []int16{10, 11}, "12", "13"})
	assert.Nil(t, err)
	cmd, ok := cmds[0].(*CommandSelect)
	assert.True(t, ok)

	assert.Equal(t, "ks1", cmd.CtxKeyspace)

	assert.Equal(t, "f1 * 2 == cqlin ( 3 + 4 , 5 + 6 )", lexemSliceToString(cmd.SelectExpLexems[0]))
	assert.Contains(t, fmt.Sprintf("%v", cmd.SelectExpAsts[0]), "&{cqlin ")
	assert.Equal(t, "f1 * 7 == cqlnotin ( params.param000 , params.param001 )", lexemSliceToString(cmd.SelectExpLexems[1]))
	assert.Contains(t, fmt.Sprintf("%v", cmd.SelectExpAsts[1]), "&{cqlnotin ")
	assert.Equal(t, "t1", cmd.TableName)
	assert.Equal(t, "f1 == cqlin ( params.param002_000 , params.param002_001 ) && f2 == cqlnotin ( params.param003 , params.param004 )", lexemSliceToString(cmd.WhereExpLexems))
	assert.Contains(t, fmt.Sprintf("%v", cmd.WhereExpAst.(*ast.BinaryExpr).X), "&{cqlin ")
	assert.Contains(t, fmt.Sprintf("%v", cmd.WhereExpAst.(*ast.BinaryExpr).Y), "&{cqlnotin ")
}

func TestInsert(t *testing.T) {
	cmds, err := ParseCommands(`USE ks1;INSERT INTO t1 (f1,f2,f3) values ('a',NULL,TRUE) IF NOT EXISTS`, nil)
	assert.Nil(t, err)
	cmd, ok := cmds[1].(*CommandInsert)
	assert.True(t, ok)

	assert.Equal(t, "ks1", cmd.CtxKeyspace)
	assert.Equal(t, "t1", cmd.TableName)

	assert.Equal(t, "f1", cmd.ColumnNames[0])
	assert.Equal(t, "a", cmd.ColumnValueLexems[0][0].V)
	assert.Equal(t, "f2", cmd.ColumnNames[1])
	assert.Equal(t, "NULL", cmd.ColumnValueLexems[1][0].V)
	assert.Equal(t, LexemNull, cmd.ColumnValueLexems[1][0].T)
	assert.Equal(t, LexemNull, cmd.ColumnValueLexems[1][0].T)

	assert.True(t, cmd.IfNotExists)

	cmds, err = ParseCommands(`USE ks1;INSERT INTO t1 (f1,f2,f3) values (?,?,?)`, []any{1, 2, 3})
	assert.Nil(t, err)
	cmd, ok = cmds[1].(*CommandInsert)
	assert.True(t, ok)

	assert.Equal(t, "ks1", cmd.CtxKeyspace)
	assert.Equal(t, "t1", cmd.TableName)

	assert.Equal(t, 1, cmd.ColumnValues[0])
	assert.Equal(t, 2, cmd.ColumnValues[1])
	assert.Equal(t, 3, cmd.ColumnValues[2])

	cmds, err = ParseCommands(`INSERT INTO ks1.t1 () values ()`, nil)
	assert.Contains(t, err.Error(), "column list cannot be empty")

	cmds, err = ParseCommands(`INSERT INTO ks1.t1 (f1) values ()`, nil)
	assert.Contains(t, err.Error(), "value list length (0) should match column list length (1)")
}

func TestUpdate(t *testing.T) {
	cmds, err := ParseCommands(`UPDATE ks1.t1 SET f1 = 1+2, f2 = 'a'='b', f3 = token(f2), f4=NULL WHERE f1 = 10 AND f2 IN ( 'c', 'd') OR f2 NOT IN ('e') IF EXISTS`, nil)
	assert.Nil(t, err)
	cmd, ok := cmds[0].(*CommandUpdate)
	assert.True(t, ok)

	assert.Equal(t, "ks1", cmd.CtxKeyspace)
	assert.Equal(t, "t1", cmd.TableName)

	assert.Equal(t, "f1", cmd.ColumnSetExpressions[0].Name)
	assert.Equal(t, "1 + 2", lexemSliceToString(cmd.ColumnSetExpressions[0].ExpLexems))
	assert.Equal(t, "f2", cmd.ColumnSetExpressions[1].Name)
	assert.Equal(t, "a == b", lexemSliceToString(cmd.ColumnSetExpressions[1].ExpLexems))
	assert.Equal(t, "f3", cmd.ColumnSetExpressions[2].Name)
	assert.Equal(t, "token ( f2 )", lexemSliceToString(cmd.ColumnSetExpressions[2].ExpLexems))
	assert.Equal(t, "f4", cmd.ColumnSetExpressions[3].Name)
	assert.Equal(t, "NULL", lexemSliceToString(cmd.ColumnSetExpressions[3].ExpLexems))

	assert.Equal(t, "f1 == 10 && f2 == cqlin ( c , d ) || f2 == cqlnotin ( e )", lexemSliceToString(cmd.WhereExpLexems))

	assert.True(t, cmd.IfExists)

	cmds, err = ParseCommands(`use ks1;UPDATE t1 SET f1 = 1 IF f2 IN (4)`, nil)
	assert.Nil(t, err)
	cmd, ok = cmds[1].(*CommandUpdate)
	assert.True(t, ok)

	assert.Equal(t, "ks1", cmd.CtxKeyspace)
	assert.Equal(t, "t1", cmd.TableName)

	assert.Equal(t, "f1", cmd.ColumnSetExpressions[0].Name)
	assert.Equal(t, "1", lexemSliceToString(cmd.ColumnSetExpressions[0].ExpLexems))
	assert.Equal(t, "&{1 2 INT 1}", fmt.Sprintf("%v", cmd.ColumnSetExpAsts[0]))

	assert.False(t, cmd.IfExists)
	assert.Equal(t, "f2 == cqlin ( 4 )", lexemSliceToString(cmd.IfExpLexems))
	assert.Contains(t, fmt.Sprintf("%v", cmd.IfExpAst), "&{cqlin ")
}

func TestDelete(t *testing.T) {
	cmds, err := ParseCommands(`USE ks1;DELETE t1.col1 FROM t1 WHERE f1 = NULL IF EXISTS;DELETE FROM ks2.t1`, nil)
	assert.Nil(t, err)
	cmd, ok := cmds[1].(*CommandDelete)
	assert.True(t, ok)

	assert.Equal(t, "ks1", cmd.CtxKeyspace)
	assert.Equal(t, "t1", cmd.TableName)

	assert.Equal(t, "col1", cmd.ColumnsToDelete[0])

	assert.Equal(t, "f1", cmd.WhereExpLexems[0].V)
	assert.Equal(t, "==", cmd.WhereExpLexems[1].V)
	assert.Equal(t, LexemNull, cmd.WhereExpLexems[2].T)
	assert.Equal(t, "NULL", cmd.WhereExpLexems[2].V)

	assert.True(t, cmd.IfExists)

	cmd, ok = cmds[2].(*CommandDelete)
	assert.True(t, ok)

	assert.Equal(t, "ks2", cmd.CtxKeyspace)
	assert.Equal(t, "t1", cmd.TableName)

	assert.False(t, cmd.IfExists)
}
