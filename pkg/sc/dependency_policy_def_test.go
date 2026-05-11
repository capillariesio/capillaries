package sc

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDependencyPolicyBad(t *testing.T) {

	polDef := DependencyPolicyDef{}

	conf := `{"event_priority_order": "run_i(asc)",	"rules": []}`
	err := polDef.Deserialize([]byte(conf), ScriptJson)
	assertErrorPrefix(t, "cannot parse event order string 'run_i(asc)'", err.Error())

	conf = `{"event_priority_order": "run_id(", "rules": []}`
	err = polDef.Deserialize([]byte(conf), ScriptJson)
	assertErrorPrefix(t, "cannot parse event order string 'run_id(': cannot parse order def 'non_unique(run_id()'", err.Error())

	conf = `{"event_priority_order": "run_id(bad)", "rules": []}`
	err = polDef.Deserialize([]byte(conf), ScriptJson)
	assertErrorPrefix(t, "cannot parse event order string 'run_id(bad)'", err.Error())

	conf = `{"event_priority_order": "run_id(asc)",	"rules": [{	"cmd": "go", "expression": "nrs.run_is_current && nrs.run_status == bad"	}]}`
	err = polDef.Deserialize([]byte(conf), ScriptJson)
	assertErrorPrefix(t, "cannot parse rule expression 'nrs.run_is_current && nrs.run_status == bad': plain (non-selector) identifiers", err.Error())
}

func TestDependencyPolicyGood(t *testing.T) {

	polDef := DependencyPolicyDef{}

	conf := `{"event_priority_order": "run_id(asc),run_is_current(desc)", "rules": [
		{"cmd": "go", "expression": "nrs.run_is_current && nrs.run_status == wfmodel.RunStart"	},
		{"cmd": "go", "expression": "nrs.run_is_current == true"	}
	]}`
	assert.Nil(t, polDef.Deserialize([]byte(conf), ScriptJson))

	assert.Equal(t, "run_id", polDef.OrderIdxDef.Components[0].FieldName)
	assert.Equal(t, IdxSortAsc, polDef.OrderIdxDef.Components[0].SortOrder)
	assert.Equal(t, "run_is_current", polDef.OrderIdxDef.Components[1].FieldName)
	assert.Equal(t, IdxSortDesc, polDef.OrderIdxDef.Components[1].SortOrder)
	assert.Equal(t, NodeGo, polDef.Rules[0].Cmd)

	assert.Nil(t, polDef.evalRuleExpressionsAndCheckType())
}
