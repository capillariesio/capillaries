package cql

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/capillariesio/capillaries/pkg/evalcapi"
	"github.com/capillariesio/capillaries/pkg/sc"
	"github.com/shopspring/decimal"
	"gopkg.in/inf.v0"
)

func SumOfExpBackoffDelaysMs(startDelayMs int64, expBackoffFactorMultiplier int64, iteration int) int64 {
	s := int64(0)
	for i := range iteration + 1 {
		s += startDelayMs * int64(math.Pow(float64(expBackoffFactorMultiplier), float64(i)))
	}
	return s
}

type PreparedQuery struct {
	Qb    *QueryBuilder
	Query string
}

/*
Data/idx table name for each run needs run id as a suffix
*/
func RunIdSuffix(runId int16) string {
	if runId > 0 {
		return fmt.Sprintf(sc.RunIdSuffixFormat, runId)
	}
	return ""
}

/*
Helper used in query builder
*/
func valueToString(value any, quotePolicy QuotePolicyType) string {
	switch v := value.(type) {
	case string:
		if quotePolicy == ForceUnquote {
			return strings.ReplaceAll(v, "'", "''")
		}
		return fmt.Sprintf("'%s'", strings.ReplaceAll(v, "'", "''"))
	case time.Time:
		if quotePolicy == ForceUnquote {
			return v.Format(sc.CassandraDatetimeFormat)
		}
		return v.Format(fmt.Sprintf("'%s'", sc.CassandraDatetimeFormat))
	default:
		return fmt.Sprintf("%v", v)
	}
}

func valueToCqlParam(value any) any {
	switch v := value.(type) {
	case decimal.Decimal:
		f, _ := v.Float64()
		scaled := int64(math.Round(f * 100))
		return inf.NewDec(scaled, 2)
	default:
		return v
	}
}

type queryBuilderColumnDefs struct {
	Columns []string
	Types   []string
}

func (cd *queryBuilderColumnDefs) add(column string, fieldType evalcapi.TableFieldType) {
	cd.Columns = append(cd.Columns, column)
	var cqlType string
	switch fieldType {
	case evalcapi.FieldTypeInt:
		cqlType = "BIGINT" // 64-bit int
	case evalcapi.FieldTypeDecimal2:
		cqlType = "DECIMAL"
	case evalcapi.FieldTypeFloat:
		cqlType = "DOUBLE"
	case evalcapi.FieldTypeString:
		cqlType = "TEXT"
	case evalcapi.FieldTypeBool:
		cqlType = "BOOLEAN"
	case evalcapi.FieldTypeDateTime:
		cqlType = "TIMESTAMP" // Cassandra stores milliseconds since epoch
	default:
		cqlType = fmt.Sprintf("UKNOWN_TYPE_%s", fieldType)
	}
	cd.Types = append(cd.Types, cqlType)
}

type queryBuilderPreparedColumnData struct {
	Columns      []string
	Values       []any
	ColumnIdxMap map[string]int
	ValueIdxMap  map[string]int
}

func (cd *queryBuilderPreparedColumnData) addColumnName(column string) error {
	if _, ok := cd.ColumnIdxMap[column]; ok {
		return fmt.Errorf("cannot add same column %s to a prepared query twice: %v", column, cd.Columns)
	}
	curColCount := len(cd.ColumnIdxMap)
	cd.Columns = append(cd.Columns, column)
	cd.Values = append(cd.Values, nil) // Grow Values in lockstep; addColumnValue sets it by index
	cd.ColumnIdxMap[column] = curColCount
	return nil
}
func (cd *queryBuilderPreparedColumnData) addColumnValue(column string, value any) error {
	colIdx, ok := cd.ColumnIdxMap[column]
	if !ok {
		return fmt.Errorf("cannot set value for non-prepared column %s, available columns are %v", column, cd.Columns)
	}
	cd.Values[colIdx] = valueToCqlParam(value)
	cd.ValueIdxMap[column] = colIdx
	return nil
}

type queryBuilderColumnData struct {
	Columns []string
	Values  []string
}

func (cd *queryBuilderColumnData) add(column string, value any, quotePolicy QuotePolicyType) {
	cd.Values = append(cd.Values, valueToString(value, quotePolicy))
	cd.Columns = append(cd.Columns, column)
}

type queryBuilderConditions struct {
	Items []string
}

func (cc *queryBuilderConditions) addInInt(column string, values []int64) {
	inValues := make([]string, len(values))
	for i, v := range values {
		inValues[i] = fmt.Sprintf("%d", v)
	}
	cc.Items = append(cc.Items, fmt.Sprintf("%s IN ( %s )", column, strings.Join(inValues, ", ")))
}

func (cc *queryBuilderConditions) addInInt16(column string, values []int16) {
	inValues := make([]string, len(values))
	for i, v := range values {
		inValues[i] = fmt.Sprintf("%d", v)
	}
	cc.Items = append(cc.Items, fmt.Sprintf("%s IN ( %s )", column, strings.Join(inValues, ", ")))
}

func (cc *queryBuilderConditions) addInString(column string, values []string) {
	escapedValues := make([]string, len(values))
	for i, v := range values {
		escapedValues[i] = strings.ReplaceAll(v, "'", "''")
	}
	cc.Items = append(cc.Items, fmt.Sprintf("%s IN ( '%s' )", column, strings.Join(escapedValues, "', '")))
}

func (cc *queryBuilderConditions) addSimple(column string, op string, value any) {
	cc.Items = append(cc.Items, fmt.Sprintf("%s %s %s", column, op, valueToString(value, LeaveQuoteAsIs)))
}
func (cc *queryBuilderConditions) addSimpleForceUnquote(column string, op string, value any) {
	cc.Items = append(cc.Items, fmt.Sprintf("%s %s %s", column, op, valueToString(value, ForceUnquote)))
}

/*
QueryBuilder - very simple cql query builder that does not require db connection
*/
type QueryBuilder struct {
	ColumnDefs           queryBuilderColumnDefs
	PartitionKeyColumns  []string
	ClusteringKeyColumns []string
	ColumnData           queryBuilderColumnData
	PreparedColumnData   queryBuilderPreparedColumnData
	Conditions           queryBuilderConditions
	IfConditions         queryBuilderConditions
	SelectLimit          int
	FormattedKeyspace    string
	OrderByColumns       []string
}

func NewQB() *QueryBuilder {
	var qb QueryBuilder
	qb.PreparedColumnData.ColumnIdxMap = map[string]int{}
	qb.PreparedColumnData.ValueIdxMap = map[string]int{}
	return &qb
}

func (qb *QueryBuilder) ColumnDef(column string, fieldType evalcapi.TableFieldType) *QueryBuilder {
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
func (qb *QueryBuilder) Write(column string, value any) *QueryBuilder {
	qb.ColumnData.add(column, value, LeaveQuoteAsIs)
	return qb
}

func (qb *QueryBuilder) WritePreparedColumn(column string) error {
	return qb.PreparedColumnData.addColumnName(column)
}

func (qb *QueryBuilder) WritePreparedValue(column string, value any) error {
	return qb.PreparedColumnData.addColumnValue(column, value)
}

/*
WriteForceUnquote - add a column for INSERT or UPDATE
*/
func (qb *QueryBuilder) WriteForceUnquote(column string, value any) *QueryBuilder {
	qb.ColumnData.add(column, value, ForceUnquote)
	return qb
}

/*
Cond - add condition for SELECT, UPDATE or DELETE
*/
func (qb *QueryBuilder) Cond(column string, op string, value any) *QueryBuilder {
	qb.Conditions.addSimple(column, op, value)
	return qb
}

func (qb *QueryBuilder) CondPrepared(column string, op string) *QueryBuilder {
	qb.Conditions.addSimpleForceUnquote(column, op, "?")
	return qb
}

func (qb *QueryBuilder) CondInPrepared(column string) *QueryBuilder {
	qb.Conditions.addSimpleForceUnquote(column, "IN", "?")
	return qb
}

/*
CondIn - add IN condition for SELECT, UPDATE or DELETE
*/
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

func (qb *QueryBuilder) If(column string, op string, value any) *QueryBuilder {
	qb.IfConditions.addSimple(column, op, value)
	return qb
}

/*
Insert - build INSERT query
*/
const RunIdForEmptyRun = -1

func (qb *QueryBuilder) InsertUnpreparedQuery(tableName string, ifNotExists IfNotExistsType) string {
	return qb.insertRunUnpreparedQuery(tableName, RunIdForEmptyRun, ifNotExists)
}
func (qb *QueryBuilder) insertRunUnpreparedQuery(tableName string, runId int16, ifNotExists IfNotExistsType) string {
	ifNotExistsStr := ""
	if ifNotExists == IfNotExistsLwt {
		ifNotExistsStr = "IF NOT EXISTS"
	}
	q := fmt.Sprintf("INSERT INTO %s%s%s ( %s ) VALUES ( %s ) %s;",
		qb.FormattedKeyspace,
		tableName,
		RunIdSuffix(runId),
		strings.Join(qb.ColumnData.Columns, ", "),
		strings.Join(qb.ColumnData.Values, ", "),
		ifNotExistsStr)
	if runId == 0 {
		q = "INVALID runId: " + q
	}
	return q
}

func (qb *QueryBuilder) InsertRunPreparedQuery(tableName string, runId int16, ifNotExists IfNotExistsType) (string, error) {
	ifNotExistsStr := ""
	if ifNotExists == IfNotExistsLwt {
		ifNotExistsStr = "IF NOT EXISTS"
	}
	columnCount := len(qb.PreparedColumnData.ColumnIdxMap)
	paramArray := make([]string, columnCount)
	for paramIdx := 0; paramIdx < columnCount; paramIdx++ {
		paramArray[paramIdx] = "?"
	}
	q := fmt.Sprintf("INSERT INTO %s%s%s ( %s ) VALUES ( %s ) %s;",
		qb.FormattedKeyspace,
		tableName,
		RunIdSuffix(runId),
		strings.Join(qb.PreparedColumnData.Columns[:columnCount], ", "),
		strings.Join(paramArray, ", "),
		ifNotExistsStr)
	if runId == 0 {
		return "", fmt.Errorf("invalid runId=0 in %s", q)
	}
	return q, nil
}

func (qb *QueryBuilder) InsertRunParams() ([]any, error) {
	if len(qb.PreparedColumnData.ColumnIdxMap) != len(qb.PreparedColumnData.ValueIdxMap) {
		return nil, fmt.Errorf("cannot produce insert params, length mismatch: columns %v, values %v", qb.PreparedColumnData.ColumnIdxMap, qb.PreparedColumnData.ValueIdxMap)
	}
	return qb.PreparedColumnData.Values[:len(qb.PreparedColumnData.ValueIdxMap)], nil
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
	fmt.Fprintf(&b, "SELECT %s FROM %s%s%s",
		strings.Join(columns, ", "),
		qb.FormattedKeyspace,
		tableName,
		RunIdSuffix(runId))
	if len(qb.Conditions.Items) > 0 {
		b.WriteString(" WHERE ")
		b.WriteString(strings.Join(qb.Conditions.Items, " AND "))
	}
	if len(qb.OrderByColumns) > 0 {
		fmt.Fprintf(&b, " ORDER BY %s ", strings.Join(qb.OrderByColumns, ","))
	}
	if qb.SelectLimit > 0 {
		fmt.Fprintf(&b, " LIMIT %d", qb.SelectLimit)
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
		RunIdSuffix(runId),
		strings.Join(qb.Conditions.Items, " AND "))
	if runId == 0 {
		q = "DEV ERROR, INVALID runId: " + q
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
	assignments := make([]string, len(qb.ColumnData.Columns))
	for i := 0; i < len(qb.ColumnData.Columns); i++ {
		assignments[i] = fmt.Sprintf("%s = %s", qb.ColumnData.Columns[i], qb.ColumnData.Values[i])
	}
	q := fmt.Sprintf("UPDATE %s%s%s SET %s WHERE %s",
		qb.FormattedKeyspace,
		tableName,
		RunIdSuffix(runId),
		strings.Join(assignments, ", "),
		strings.Join(qb.Conditions.Items, " AND "))

	if len(qb.IfConditions.Items) > 0 {
		q += " IF " + strings.Join(qb.IfConditions.Items, " AND ")
	}
	if runId == 0 {
		q = "INVALID runId: " + q
	}
	return q
}

func (qb *QueryBuilder) CreateRun(tableName string, runId int16, ifNotExists IfNotExistsType, createProperties string) string {
	var b strings.Builder
	if runId == 0 {
		b.WriteString("INVALID runId: ")
	}
	b.WriteString("CREATE TABLE ")
	if ifNotExists == IfNotExistsLwt {
		b.WriteString("IF NOT EXISTS ")
	}
	fmt.Fprintf(&b, "%s%s%s ( ", qb.FormattedKeyspace, tableName, RunIdSuffix(runId))
	for i := 0; i < len(qb.ColumnDefs.Columns); i++ {
		b.WriteString(qb.ColumnDefs.Columns[i])
		b.WriteString(" ")
		b.WriteString(qb.ColumnDefs.Types[i])
		if i < len(qb.ColumnDefs.Columns)-1 {
			b.WriteString(", ")
		}
	}
	if len(qb.PartitionKeyColumns) > 0 {
		b.WriteString(", ")
		fmt.Fprintf(&b, "PRIMARY KEY((%s)", strings.Join(qb.PartitionKeyColumns, ", "))
		if len(qb.ClusteringKeyColumns) > 0 {
			b.WriteString(", ")
			b.WriteString(strings.Join(qb.ClusteringKeyColumns, ", "))
		}
		b.WriteString(")")
	}
	b.WriteString(")")
	// If needed, can add for Amazon Keyspaces
	// WITH CUSTOM_PROPERTIES = {'capacity_mode':{'throughput_mode':'PROVISIONED','write_capacity_units':1000,'read_capacity_units':1000}}
	if strings.Trim(createProperties, " ") != "" {
		b.WriteString(" ")
		b.WriteString(strings.Trim(createProperties, " "))
	}
	b.WriteString(";")

	return b.String()
}

// Currently not used, leave it commented out just in case
// func (qb *QueryBuilder) Drop(tableName string) string {
// 	return qb.DropRun(tableName, RunIdForEmptyRun)
// }
// func (qb *QueryBuilder) DropRun(tableName string, runId int16) string {
// 	q := fmt.Sprintf("DROP TABLE IF EXISTS %s%s%s", qb.FormattedKeyspace, tableName, RunIdSuffix(runId))
// 	if runId == 0 {
// 		q = "INVALID runId: " + q
// 	}
// 	return q
// }

func (qb *QueryBuilder) DropKeyspace() string {
	return fmt.Sprintf("DROP KEYSPACE IF EXISTS %s", strings.ReplaceAll(qb.FormattedKeyspace, ".", ""))
}
