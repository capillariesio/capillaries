package wfmodel

import (
	"fmt"
	"reflect"
	"strings"
	"time"
)

const WfmodelNamespace string = "wfmodel"
const PrintTableDelimiter = "/"

const LogTsFormatQuoted = `"2006-01-02T15:04:05.000-0700"`

// GetSpacedHeader - prints formatted struct field names, uses reflection, shoud not be used in prod
func GetSpacedHeader(n any) string {
	t := reflect.TypeOf(n)
	columns := make([]string, t.NumField())
	for i := 0; i < t.NumField(); i++ {
		field := t.FieldByIndex([]int{i})
		h, ok := field.Tag.Lookup("header")
		if ok {
			f, ok := field.Tag.Lookup("format")
			if ok {
				columns[i] = fmt.Sprintf(f, h)
			} else {
				columns[i] = fmt.Sprintf("%v", h)
			}
		} else {
			columns[i] = "N/A"
		}

	}
	return strings.Join(columns, PrintTableDelimiter)
}

/*
GetObjectModelFieldFormats - helper to get formats for each field of an object model
*/
func GetObjectModelFieldFormats(t reflect.Type) []string {
	formats := make([]string, t.NumField())

	for i := 0; i < t.NumField(); i++ {
		field := t.FieldByIndex([]int{i})
		f, ok := field.Tag.Lookup("format")
		if ok {
			formats[i] = f
		} else {
			formats[i] = "%v"
		}
	}
	return formats
}

func GetCreateTableCql(t reflect.Type, keyspace string, tableName string) string {

	columnDefs := make([]string, t.NumField())
	keyDefs := make([]string, t.NumField())
	keyCount := 0

	for i := 0; i < t.NumField(); i++ {
		field := t.FieldByIndex([]int{i})
		cqlColumn, ok := field.Tag.Lookup("column")
		if ok {
			cqlType, ok := field.Tag.Lookup("type")
			if ok {
				columnDefs[i] = fmt.Sprintf("%s %s", cqlColumn, cqlType)
				cqlKeyFlag, ok := field.Tag.Lookup("key")
				if ok && cqlKeyFlag == "true" {
					keyDefs[keyCount] = cqlColumn
					keyCount++
				}
			} else {
				columnDefs[i] = fmt.Sprintf("no type for field %s", field.Name)
			}
		} else {
			columnDefs[i] = fmt.Sprintf("no column name for field %s", field.Name)
		}
	}

	// For Amazon Keyspaces, you may add
	// WITH CUSTOM_PROPERTIES = {'capacity_mode':{'throughput_mode':'PROVISIONED','write_capacity_units':1000,'read_capacity_units':1000}}
	return fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s.%s (%s, PRIMARY KEY(%s));",
		keyspace,
		tableName,
		strings.Join(columnDefs, ", "),
		strings.Join(keyDefs[:keyCount], ", "))
}

func ReadTimeFromRow(fieldName string, r map[string]any) (time.Time, error) {
	v, ok := r[fieldName].(time.Time)
	if !ok {
		return v, fmt.Errorf("cannot read time %s from %v", fieldName, r)
	}
	return v, nil
}

func ReadInt16FromRow(fieldName string, r map[string]any) (int16, error) {
	v, ok := r[fieldName].(int)
	if !ok {
		return int16(0), fmt.Errorf("cannot read int16 %s from %v", fieldName, r)
	}
	return int16(v), nil
}

func ReadInt64FromRow(fieldName string, r map[string]any) (int64, error) {
	v, ok := r[fieldName].(int64)
	if !ok {
		return int64(0), fmt.Errorf("cannot read int64 %s from %v", fieldName, r)
	}
	return v, nil
}

func ReadRunStatusFromRow(fieldName string, r map[string]any) (RunStatusType, error) {
	v, ok := r[fieldName].(int8)
	if !ok {
		return RunNone, fmt.Errorf("cannot read run status %s from %v", fieldName, r)
	}
	return RunStatusType(v), nil
}

func ReadStringFromRow(fieldName string, r map[string]any) (string, error) {
	v, ok := r[fieldName].(string)
	if !ok {
		return v, fmt.Errorf("cannot read string %s from %v", fieldName, r)
	}
	return v, nil
}

func ReadNodeBatchStatusFromRow(fieldName string, r map[string]any) (NodeBatchStatusType, error) {
	v, ok := r[fieldName].(int8)
	if !ok {
		return NodeBatchNone, fmt.Errorf("cannot read node/batch status %s from %v", fieldName, r)
	}
	return NodeBatchStatusType(v), nil
}
