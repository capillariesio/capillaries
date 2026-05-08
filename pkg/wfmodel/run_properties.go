package wfmodel

import (
	"fmt"
	"strings"
)

const TableNameRunProperties = "wf_run_properties"

// Object model with tags that allow to create cql CREATE TABLE queries and to print object
type RunProperties struct {
	RunId           int16  `header:"run_id" format:"%6d" column:"run_id" type:"int" key:"true" json:"run_id"`
	StartNodes      string `header:"start_nodes" format:"%20v" column:"start_nodes" type:"text" json:"start_nodes"`
	AffectedNodes   string `header:"affected_nodes" format:"%20v" column:"affected_nodes" type:"text" json:"affected_nodes"`
	ScriptUrl       string `header:"script_url" format:"%20v" column:"script_url" type:"text" json:"script_url"`
	ScriptParamsUrl string `header:"script_params_url" format:"%20v" column:"script_params_url" type:"text" json:"script_params_url"`
	RunDescription  string `header:"run_desc" format:"%20v" column:"run_description" type:"text" json:"run_description"`
}

func RunPropertiesAllFields() []string {
	return []string{"run_id", "start_nodes", "affected_nodes", "script_url", "script_params_url", "run_description"}
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
		case "script_url":
			res.ScriptUrl, err = ReadStringFromRow(fieldName, r)
		case "script_params_url":
			res.ScriptParamsUrl, err = ReadStringFromRow(fieldName, r)
		case "run_description":
			res.RunDescription, err = ReadStringFromRow(fieldName, r)
		default:
			return nil, fmt.Errorf("unknown %s field %s", fieldName, TableNameRunProperties)
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

func intersectTwoSlicesOfStrings(slice1, slice2 []string) []string {
	map1 := make(map[string]bool)
	for _, v := range slice1 {
		map1[v] = true
	}

	var result []string
	for _, v := range slice2 {
		if map1[v] {
			result = append(result, v)
			delete(map1, v)
		}
	}
	return result
}

func MultipleRunsPropertiesToDependencies(rows []map[string]any, depNodeNames []string) ([]int16, map[int16][]string, error) {
	depRunIds := make([]int16, 0)
	depRunNodesMap := map[int16][]string{}
	for _, r := range rows {
		rec, err := NewRunPropertiesFromMap(r, RunPropertiesAllFields())
		if err != nil {
			return nil, nil, err
		}

		// Take only dependency nodes (0, 1 or 2 - since there can be only a reader and a lookot dependency)
		affectedDepNodes := intersectTwoSlicesOfStrings(strings.Split(rec.AffectedNodes, ","), depNodeNames)
		if len(affectedDepNodes) > 0 {
			depRunIds = append(depRunIds, rec.RunId)
			depRunNodesMap[rec.RunId] = intersectTwoSlicesOfStrings(strings.Split(rec.AffectedNodes, ","), depNodeNames)
		}
	}
	return depRunIds, depRunNodesMap, nil
}
