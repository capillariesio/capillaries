package deploy

import (
	"encoding/json"
	"fmt"

	"github.com/itchyny/gojq"
)

func ExecLocalAndGetJsonValue(prj *Project, cmdPath string, params []string, query string) (any, ExecResult) {
	er := ExecLocal(prj, cmdPath, params, prj.CliEnvVars, "")
	if er.Error != nil {
		return nil, er
	}

	q, err := gojq.Parse(query)
	if err != nil {
		er.Error = err
		return nil, er
	}

	// This is a brutal way to unmarshal incoming JSON, but it should work
	var jsonObj map[string]any
	if err := json.Unmarshal([]byte(er.Stdout), &jsonObj); err != nil {
		er.Error = fmt.Errorf("cannot unmarshal json, error %s, json %s", err.Error(), er.Stdout)
		return nil, er
	}

	iter := q.Run(jsonObj)
	if v, ok := iter.Next(); ok {
		return v, er
	}

	er.Error = fmt.Errorf("no values found by query %s in %s", query, er.Stdout)
	return nil, er
}

func ExecLocalAndGetJsonString(prj *Project, cmdPath string, params []string, query string, allowEmpty bool) (string, ExecResult) {
	v, er := ExecLocalAndGetJsonValue(prj, cmdPath, params, query)
	if er.Error != nil {
		return "", er
	}

	if v == nil {
		if !allowEmpty {
			er.Error = fmt.Errorf("string value returned by query %s is empty", query)
		}
		return "", er

	}

	switch typedVal := v.(type) {
	case string:
		return typedVal, er
	default:
		er.Error = fmt.Errorf("non-string value %v returned by query %s", v, query)
		return "", er
	}
}

func ExecLocalAndGetJsonBool(prj *Project, cmdPath string, params []string, query string) (bool, ExecResult) {
	v, er := ExecLocalAndGetJsonValue(prj, cmdPath, params, query)
	if er.Error != nil {
		return false, er
	}
	switch typedVal := v.(type) {
	case bool:
		return typedVal, er
	default:
		er.Error = fmt.Errorf("non-bool value %v returned by query %s", v, query)
		return false, er
	}
}
