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

	depRunIds, depRunNodesMap, err := MultipleRunsPropertiesToDependencies(rows, []string{"affNode12", "affNode22"})
	assert.Nil(t, err)
	assert.Equal(t, "[1 2]", fmt.Sprintf("%v", depRunIds))
	assert.Equal(t, 2, len(depRunNodesMap))
	assert.Equal(t, "affNode12", strings.Join(depRunNodesMap[1], ","))
	assert.Equal(t, "affNode22", strings.Join(depRunNodesMap[2], ","))

	depRunIds, depRunNodesMap, err = MultipleRunsPropertiesToDependencies(rows, []string{"affNode12"})
	assert.Nil(t, err)
	assert.Equal(t, "[1]", fmt.Sprintf("%v", depRunIds))
	assert.Equal(t, 1, len(depRunNodesMap))
	assert.Equal(t, "affNode12", strings.Join(depRunNodesMap[1], ","))

	depRunIds, depRunNodesMap, err = MultipleRunsPropertiesToDependencies(rows, []string{"affNode12", "affNodeExtra"})
	assert.Contains(t, err.Error(), "dependency node affNodeExtra is not present")

	rows[0]["run_id"] = "a"
	depRunIds, depRunNodesMap, err = MultipleRunsPropertiesToDependencies(rows, []string{"affNode12", "affNodeExtra"})
	assert.Contains(t, err.Error(), "cannot read int16 run_id")
}
