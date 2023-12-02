package wfmodel

import (
	"fmt"
)

const TableNameRunAffectedNodes = "wf_run_affected_nodes"

// Object model with tags that allow to create cql CREATE TABLE queries and to print object
type RunProperties struct {
	RunId           int16  `header:"run_id" format:"%6d" column:"run_id" type:"int" key:"true" json:"run_id"`
	StartNodes      string `header:"start_nodes" format:"%20v" column:"start_nodes" type:"text" json:"start_nodes"`
	AffectedNodes   string `header:"affected_nodes" format:"%20v" column:"affected_nodes" type:"text" json:"affected_nodes"`
	ScriptUri       string `header:"script_uri" format:"%20v" column:"script_uri" type:"text" json:"script_uri"`
	ScriptParamsUri string `header:"script_params_uri" format:"%20v" column:"script_params_uri" type:"text" json:"script_params_uri"`
	RunDescription  string `header:"run_desc" format:"%20v" column:"run_description" type:"text" json:"run_description"`
}

func RunPropertiesAllFields() []string {
	return []string{"run_id", "start_nodes", "affected_nodes", "script_uri", "script_params_uri", "run_description"}
}

func NewRunPropertiesFromMap(r map[string]any, fields []string) (*RunProperties, error) {
	res := &RunProperties{}
	for _, fieldName := range fields {
		var err error
		switch fieldName {
		case "run_id":
			res.RunId, err = ReadInt16FromRow(fieldName, r)
		case "start_nodes":
			res.StartNodes, err = ReadStringFromRow(fieldName, r)
		case "affected_nodes":
			res.AffectedNodes, err = ReadStringFromRow(fieldName, r)
		case "script_uri":
			res.ScriptUri, err = ReadStringFromRow(fieldName, r)
		case "script_params_uri":
			res.ScriptParamsUri, err = ReadStringFromRow(fieldName, r)
		case "run_description":
			res.RunDescription, err = ReadStringFromRow(fieldName, r)
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
// func (n RunProperties) ToSpacedString() string {
// 	t := reflect.TypeOf(n)
// 	formats := GetObjectModelFieldFormats(t)
// 	values := make([]string, t.NumField())

// 	v := reflect.ValueOf(&n).Elem()
// 	for i := 0; i < v.NumField(); i++ {
// 		fv := v.Field(i)
// 		values[i] = fmt.Sprintf(formats[i], fv)
// 	}
// 	return strings.Join(values, PrintTableDelimiter)
// }
