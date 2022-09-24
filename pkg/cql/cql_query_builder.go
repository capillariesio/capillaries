package cql

import (
	"fmt"
	"strings"
	"time"

	"github.com/kleineshertz/capillaries/pkg/sc"
)

type IfNotExistsType int

const (
	IgnoreIfExists IfNotExistsType = 1
	ThrowIfExists  IfNotExistsType = 0
)

/*
Data/idx table name for each run needs run id as a suffix
*/
func runIdSuffix(runId int16) string {
	if runId > 0 {
		return fmt.Sprintf("_%05d", runId)
	} else {
		return ""
	}
}

/*
Helper used in query builder
*/
func valueToString(value interface{}, forceUnquote bool) string {
	switch v := value.(type) {
	case string:
		if forceUnquote {
			return strings.ReplaceAll(v, "'", "''")
		} else {
			return fmt.Sprintf("'%s'", strings.ReplaceAll(v, "'", "''"))
		}
	case time.Time:
		if forceUnquote {
			return v.Format(sc.CassandraDatetimeFormat)
		} else {
			return v.Format(fmt.Sprintf("'%s'", sc.CassandraDatetimeFormat))
		}
	default:
		return fmt.Sprintf("%v", v)
	}
}

type queryBuilderColumnDefs struct {
	Columns [256]string
	Types   [256]string
	Len     int
}

func (cd *queryBuilderColumnDefs) add(column string, fieldType sc.TableFieldType) {
	cd.Columns[cd.Len] = column
	switch fieldType {
	case sc.FieldTypeInt:
		cd.Types[cd.Len] = "BIGINT"
	case sc.FieldTypeDecimal2:
		cd.Types[cd.Len] = "DECIMAL"
	case sc.FieldTypeFloat:
		cd.Types[cd.Len] = "DOUBLE"
	case sc.FieldTypeString:
		cd.Types[cd.Len] = "TEXT"
	case sc.FieldTypeBool:
		cd.Types[cd.Len] = "BOOLEAN"
	case sc.FieldTypeDateTime:
		cd.Types[cd.Len] = "TIMESTAMP" // Cassandra stores milliseconds since epoch
	default:
		cd.Types[cd.Len] = fmt.Sprintf("UKNOWN_TYPE_%s", fieldType)
	}
	cd.Len++
}

type queryBuilderColumnData struct {
	Columns [256]string
	Values  [256]string
	Len     int
}

func (cd *queryBuilderColumnData) add(column string, value interface{}, forceUnquote bool) {
	cd.Values[cd.Len] = valueToString(value, forceUnquote)
	cd.Columns[cd.Len] = column
	cd.Len++
}

type queryBuilderConditions struct {
	Items [256]string
	Len   int
}

func (cc *queryBuilderConditions) addIn(column string, values []interface{}) {
	inValues := make([]string, len(values))
	for i, v := range values {
		inValues[i] = valueToString(v, false)
	}
	cc.Items[cc.Len] = fmt.Sprintf("%s IN ( %s )", column, strings.Join(inValues, ", "))
	cc.Len++
}

func (cc *queryBuilderConditions) addInInt(column string, values []int64) {
	inValues := make([]string, len(values))
	for i, v := range values {
		inValues[i] = fmt.Sprintf("%d", v)
	}
	cc.Items[cc.Len] = fmt.Sprintf("%s IN ( %s )", column, strings.Join(inValues, ", "))
	cc.Len++
}

func (cc *queryBuilderConditions) addInInt16(column string, values []int16) {
	inValues := make([]string, len(values))
	for i, v := range values {
		inValues[i] = fmt.Sprintf("%d", v)
	}
	cc.Items[cc.Len] = fmt.Sprintf("%s IN ( %s )", column, strings.Join(inValues, ", "))
	cc.Len++
}

func (cc *queryBuilderConditions) addInString(column string, values []string) {
	cc.Items[cc.Len] = fmt.Sprintf("%s IN ( '%s' )", column, strings.Join(values, "', '"))
	cc.Len++
}

func (cc *queryBuilderConditions) addSimple(column string, op string, value interface{}) {
	cc.Items[cc.Len] = fmt.Sprintf("%s %s %s", column, op, valueToString(value, false))
	cc.Len++
}

/*
QueryBuilder - very simple cql query builder that does not require db connection
*/
type QueryBuilder struct {
	ColumnDefs           queryBuilderColumnDefs
	PartitionKeyColumns  []string
	ClusteringKeyColumns []string
	ColumnData           queryBuilderColumnData
	Conditions           queryBuilderConditions
	IfConditions         queryBuilderConditions
	SelectLimit          int
	FormattedKeyspace    string
	OrderByColumns       []string
}

func (qb *QueryBuilder) ColumnDef(column string, fieldType sc.TableFieldType) *QueryBuilder {
	qb.ColumnDefs.add(column, fieldType)
	return qb
}

/*
 */
func (qb *QueryBuilder) PartitionKey(column ...string) *QueryBuilder {
	qb.PartitionKeyColumns = column
	return qb
}
func (qb *QueryBuilder) ClusteringKey(column ...string) *QueryBuilder {
	qb.ClusteringKeyColumns = column
	return qb
}

/*
Keyspace - specify keyspace (optional)
*/
func (qb *QueryBuilder) Keyspace(keyspace string) *QueryBuilder {
	if trimmedKeyspace := strings.TrimSpace(keyspace); len(trimmedKeyspace) > 0 {
		qb.FormattedKeyspace = fmt.Sprintf("%s.", trimmedKeyspace)
	} else {
		qb.FormattedKeyspace = ""
	}
	return qb
}

func (qb *QueryBuilder) Limit(limit int) *QueryBuilder {
	qb.SelectLimit = limit
	return qb
}

/*
Write - add a column for INSERT or UPDATE
*/
func (qb *QueryBuilder) Write(column string, value interface{}) *QueryBuilder {
	qb.ColumnData.add(column, value, false)
	return qb
}

/*
WriteForceUnquote - add a column for INSERT or UPDATE
*/
func (qb *QueryBuilder) WriteForceUnquote(column string, value interface{}) *QueryBuilder {
	qb.ColumnData.add(column, value, true)
	return qb
}

/*
Cond - add condition for SELECT, UPDATE or DELETE
*/
func (qb *QueryBuilder) Cond(column string, op string, value interface{}) *QueryBuilder {
	qb.Conditions.addSimple(column, op, value)
	return qb
}

/*
CondIn - add IN condition for SELECT, UPDATE or DELETE
*/
func (qb *QueryBuilder) CondIn(column string, values []interface{}) *QueryBuilder {
	qb.Conditions.addIn(column, values)
	return qb
}

func (qb *QueryBuilder) CondInInt(column string, values []int64) *QueryBuilder {
	qb.Conditions.addInInt(column, values)
	return qb
}

func (qb *QueryBuilder) CondInInt16(column string, values []int16) *QueryBuilder {
	qb.Conditions.addInInt16(column, values)
	return qb
}

func (qb *QueryBuilder) CondInString(column string, values []string) *QueryBuilder {
	qb.Conditions.addInString(column, values)
	return qb
}

func (qb *QueryBuilder) OrderBy(columns ...string) *QueryBuilder {
	qb.OrderByColumns = columns
	return qb
}

func (qb *QueryBuilder) If(column string, op string, value interface{}) *QueryBuilder {
	qb.IfConditions.addSimple(column, op, value)
	return qb
}

/*
Insert - build INSERT query
*/
const RunIdForEmptyRun = -1

func (qb *QueryBuilder) Insert(tableName string, ifNotExists IfNotExistsType) string {
	return qb.InsertRun(tableName, RunIdForEmptyRun, ifNotExists)
}
func (qb *QueryBuilder) InsertRun(tableName string, runId int16, ifNotExists IfNotExistsType) string {
	ifNotExistsStr := ""
	if ifNotExists == IgnoreIfExists {
		ifNotExistsStr = "IF NOT EXISTS"
	}
	q := fmt.Sprintf("INSERT INTO %s%s%s ( %s ) VALUES ( %s ) %s;",
		qb.FormattedKeyspace,
		tableName,
		runIdSuffix(runId),
		strings.Join(qb.ColumnData.Columns[:qb.ColumnData.Len], ", "),
		strings.Join(qb.ColumnData.Values[:qb.ColumnData.Len], ", "),
		ifNotExistsStr)
	if runId == 0 {
		q = "INVALID runId: " + q
	}
	return q
}

/*
Select - build SELECT query
*/
func (qb *QueryBuilder) Select(tableName string, columns []string) string {
	return qb.SelectRun(tableName, RunIdForEmptyRun, columns)
}
func (qb *QueryBuilder) SelectRun(tableName string, runId int16, columns []string) string {
	b := strings.Builder{}
	if runId == 0 {
		b.WriteString("INVALID runId: ")
	}
	b.WriteString(fmt.Sprintf("SELECT %s FROM %s%s%s",
		strings.Join(columns, ", "),
		qb.FormattedKeyspace,
		tableName,
		runIdSuffix(runId)))
	if qb.Conditions.Len > 0 {
		b.WriteString(" WHERE ")
		b.WriteString(strings.Join(qb.Conditions.Items[:qb.Conditions.Len], " AND "))
	}
	if len(qb.OrderByColumns) > 0 {
		b.WriteString(fmt.Sprintf(" ORDER BY %s ", strings.Join(qb.OrderByColumns, ",")))
	}
	if qb.SelectLimit > 0 {
		b.WriteString(fmt.Sprintf(" LIMIT %d", qb.SelectLimit))
	}
	b.WriteString(";")

	return b.String()
}

/*
Delete - build DELETE query
*/
func (qb *QueryBuilder) Delete(tableName string) string {
	return qb.DeleteRun(tableName, RunIdForEmptyRun)
}
func (qb *QueryBuilder) DeleteRun(tableName string, runId int16) string {
	q := fmt.Sprintf("DELETE FROM %s%s%s WHERE %s",
		qb.FormattedKeyspace,
		tableName,
		runIdSuffix(runId),
		strings.Join(qb.Conditions.Items[:qb.Conditions.Len], " AND "))
	if runId == 0 {
		q = "INVALID runId: " + q
	}

	return q
}

/*
Update - build UPDATE query
*/
func (qb *QueryBuilder) Update(tableName string) string {
	return qb.UpdateRun(tableName, RunIdForEmptyRun)
}
func (qb *QueryBuilder) UpdateRun(tableName string, runId int16) string {
	var assignments [256]string
	for i := 0; i < qb.ColumnData.Len; i++ {
		assignments[i] = fmt.Sprintf("%s = %s", qb.ColumnData.Columns[i], qb.ColumnData.Values[i])
	}
	q := fmt.Sprintf("UPDATE %s%s%s SET %s WHERE %s",
		qb.FormattedKeyspace,
		tableName,
		runIdSuffix(runId),
		strings.Join(assignments[:qb.ColumnData.Len], ", "),
		strings.Join(qb.Conditions.Items[:qb.Conditions.Len], " AND "))

	if qb.IfConditions.Len > 0 {
		q += " IF " + strings.Join(qb.IfConditions.Items[:qb.IfConditions.Len], " AND ")
	}
	if runId == 0 {
		q = "INVALID runId: " + q
	}
	return q
}

func (qb *QueryBuilder) Create(tableName string, ifNotExists IfNotExistsType) string {
	return qb.CreateRun(tableName, RunIdForEmptyRun, ifNotExists)
}
func (qb *QueryBuilder) CreateRun(tableName string, runId int16, ifNotExists IfNotExistsType) string {
	var b strings.Builder
	if runId == 0 {
		b.WriteString("INVALID runId: ")
	}
	b.WriteString("CREATE TABLE ")
	if ifNotExists == IgnoreIfExists {
		b.WriteString("IF NOT EXISTS ")
	}
	b.WriteString(fmt.Sprintf("%s%s%s ( ", qb.FormattedKeyspace, tableName, runIdSuffix(runId)))
	for i := 0; i < qb.ColumnDefs.Len; i++ {
		b.WriteString(qb.ColumnDefs.Columns[i])
		b.WriteString(" ")
		b.WriteString(qb.ColumnDefs.Types[i])
		if i < qb.ColumnDefs.Len-1 {
			b.WriteString(", ")
		}
	}
	if len(qb.PartitionKeyColumns) > 0 {
		b.WriteString(", ")
		b.WriteString(fmt.Sprintf("PRIMARY KEY((%s)", strings.Join(qb.PartitionKeyColumns, ", ")))
		if len(qb.ClusteringKeyColumns) > 0 {
			b.WriteString(", ")
			b.WriteString(strings.Join(qb.ClusteringKeyColumns, ", "))
		}
		b.WriteString(")")
	}
	b.WriteString(");")
	return b.String()
}

func (qb *QueryBuilder) Drop(tableName string) string {
	return qb.DropRun(tableName, RunIdForEmptyRun)
}
func (qb *QueryBuilder) DropRun(tableName string, runId int16) string {
	q := fmt.Sprintf("DROP TABLE IF EXISTS %s%s%s", qb.FormattedKeyspace, tableName, runIdSuffix(runId))
	if runId == 0 {
		q = "INVALID runId: " + q
	}
	return q
}
func (qb *QueryBuilder) DropKeyspace() string {
	return fmt.Sprintf("DROP KEYSPACE IF EXISTS %s", strings.ReplaceAll(qb.FormattedKeyspace, ".", ""))
}
