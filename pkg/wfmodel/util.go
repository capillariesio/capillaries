package wfmodel

import (
	"fmt"
	"time"
)

const WfmodelNamespace string = "wfmodel"
const PrintTableDelimiter = "/"

const LogTsFormatQuoted = `"2006-01-02T15:04:05.000-0700"`

func ReadTimeFromRow(fieldName string, r map[string]any) (time.Time, error) {
	v, ok := r[fieldName].(time.Time)
	if !ok {
		return v, fmt.Errorf("cannot read time %s from %v", fieldName, r)
	}
	return v, nil
}

func ReadInt16FromRow(fieldName string, r map[string]any) (int16, error) {
	switch typedVal := r[fieldName].(type) {
	case int:
		return int16(typedVal), nil
	case int8:
		return int16(typedVal), nil
	case int16:
		return int16(typedVal), nil
	case int32:
		return int16(typedVal), nil
	case int64:
		return int16(typedVal), nil
	default:
		return int16(0), fmt.Errorf("cannot read int16 %s from %v", fieldName, r)
	}
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
