package wfmodel

import (
	"fmt"
	"reflect"
	"strings"
)

const TableNameRunAffectedNodes = "wf_run_affected_nodes"

// Object model with tags that allow to create cql CREATE TABLE queries and to print object
type RunAffectedNodes struct {
	RunId         int16  `header:"run_id" format:"%6d" column:"run_id" type:"int" key:"true"`
	AffectedNodes string `header:"affected_nodes" format:"%20v" column:"affected_nodes" type:"text"`
}

func NewRunAffectedNodesFromMap(r map[string]interface{}, fields []string) (*RunAffectedNodes, error) {
	res := &RunAffectedNodes{}
	for _, fieldName := range fields {
		var err error
		switch fieldName {
		case "run_id":
			res.RunId, err = ReadInt16FromRow(fieldName, r)
		case "affected_nodes":
			res.AffectedNodes, err = ReadStringFromRow(fieldName, r)
		default:
			return nil, fmt.Errorf("unknown %s field %s", fieldName, TableNameRunAffectedNodes)
		}
		if err != nil {
			return nil, err
		}
	}
	return res, nil
}

// ToSpacedString - prints formatted field values, uses reflection, shoud not be used in prod
func (n RunAffectedNodes) ToSpacedString() string {
	t := reflect.TypeOf(n)
	formats := GetObjectModelFieldFormats(t)
	values := make([]string, t.NumField())

	v := reflect.ValueOf(&n).Elem()
	for i := 0; i < v.NumField(); i++ {
		fv := v.Field(i)
		values[i] = fmt.Sprintf(formats[i], fv)
	}
	return strings.Join(values, PrintTableDelimiter)
}
