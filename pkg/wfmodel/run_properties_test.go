package wfmodel

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func (e *RunProperties) ToMap() map[string]any {
	m := map[string]any{}
	for _, fieldName := range RunPropertiesAllFields() {
		switch fieldName {
		case "run_id":
			m[fieldName] = e.RunId
		case "start_nodes":
			m[fieldName] = e.StartNodes
		case "affected_nodes":
			m[fieldName] = e.AffectedNodes
		case "script_url":
			m[fieldName] = e.ScriptUrl
		case "script_params_url":
			m[fieldName] = e.ScriptParamsUrl
		case "run_description":
			m[fieldName] = e.RunDescription
		default:
		}
	}
	return m
}

func TestMultipleRunsPropertiesToDependencies(t *testing.T) {
	rows := []map[string]any{
		(&RunProperties{
			RunId:         int16(1),
			AffectedNodes: "affNode11,affNode12",
		}).ToMap(),
		(&RunProperties{
			RunId:         int16(2),
			AffectedNodes: "affNode21,affNode22",
		}).ToMap(),
	}
	runPropertiesFields := []string{"run_id", "affected_nodes"}

	depRunIds, depRunNodesMap, err := MultipleRunsPropertiesToDependencies(rows, []string{"affNode12", "affNode22"}, runPropertiesFields)
	assert.Nil(t, err)
	assert.Equal(t, "[1 2]", fmt.Sprintf("%v", depRunIds))
	assert.Equal(t, 2, len(depRunNodesMap))
	assert.Equal(t, "affNode12", strings.Join(depRunNodesMap[1], ","))
	assert.Equal(t, "affNode22", strings.Join(depRunNodesMap[2], ","))

	depRunIds, depRunNodesMap, err = MultipleRunsPropertiesToDependencies(rows, []string{"affNode12"}, runPropertiesFields)
	assert.Nil(t, err)
	assert.Equal(t, "[1]", fmt.Sprintf("%v", depRunIds))
	assert.Equal(t, 1, len(depRunNodesMap))
	assert.Equal(t, "affNode12", strings.Join(depRunNodesMap[1], ","))

	_, _, err = MultipleRunsPropertiesToDependencies(rows, []string{"affNode12", "affNodeExtra"}, runPropertiesFields)
	assert.Contains(t, err.Error(), "dependency node affNodeExtra is not present")

	rows[0]["run_id"] = "a"
	_, _, err = MultipleRunsPropertiesToDependencies(rows, []string{"affNode12", "affNodeExtra"}, runPropertiesFields)
	assert.Contains(t, err.Error(), "cannot read int16 run_id")
}

func TestNewRunPropertiesFromMap(t *testing.T) {
	row := (&RunProperties{
		RunId:           int16(1),
		StartNodes:      "startNode11,startNode12",
		AffectedNodes:   "affNode11,affNode12",
		ScriptUrl:       "scripturl",
		ScriptParamsUrl: "scriptparamsurl",
		RunDescription:  "rundesc",
	}).ToMap()

	runProps, err := NewRunPropertiesFromMap(row, RunPropertiesAllFields())
	assert.Nil(t, err)
	assert.Equal(t, row["run_id"], runProps.RunId)
	assert.Equal(t, row["start_nodes"], runProps.StartNodes)
	assert.Equal(t, row["affected_nodes"], runProps.AffectedNodes)
	assert.Equal(t, row["script_url"], runProps.ScriptUrl)
	assert.Equal(t, row["script_params_url"], runProps.ScriptParamsUrl)
	assert.Equal(t, row["run_description"], runProps.RunDescription)
}
