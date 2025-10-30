package dpc

import (
	"regexp"
	"testing"
	"time"

	"github.com/capillariesio/capillaries/pkg/sc"
	"github.com/capillariesio/capillaries/pkg/wfmodel"
	"github.com/stretchr/testify/assert"
)

func TestDefaultDependencyPolicyChecker(t *testing.T) {
	events := wfmodel.DependencyNodeEvents{
		{
			RunId:          10,
			RunIsCurrent:   true,
			RunStartTs:     time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
			RunFinalStatus: wfmodel.RunStart,
			RunCompletedTs: time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
			RunStoppedTs:   time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
			NodeIsStarted:  true,
			NodeStartTs:    time.Date(2000, 1, 1, 0, 0, 1, 0, time.UTC),
			NodeStatus:     wfmodel.NodeBatchNone,
			NodeStatusTs:   time.Date(2000, 1, 1, 0, 0, 2, 0, time.UTC)}}

	polDef := sc.DependencyPolicyDef{}
	if err := polDef.Deserialize([]byte(sc.DefaultPolicyCheckerConfJson), sc.ScriptJson); err != nil {
		t.Error(err)
		return
	}

	var cmd sc.ReadyToRunNodeCmdType
	var runId int16
	var matchedRuleIdx int
	var err error
	fullBatchId := "some_node"

	events[0].RunIsCurrent = true

	events[0].NodeStatus = wfmodel.NodeBatchRunStopReceived
	cmd, _, matchedRuleIdx, err = CheckDependencyPolicyAgainstNodeEventList(nil, fullBatchId, &polDef, events)
	assert.Nil(t, err)
	assert.Equal(t, sc.NodeWait, cmd)
	assert.Equal(t, -1, matchedRuleIdx) // "no rules matched against events"

	events[0].NodeStatus = wfmodel.NodeBatchSuccess
	cmd, runId, matchedRuleIdx, err = CheckDependencyPolicyAgainstNodeEventList(nil, fullBatchId, &polDef, events)
	assert.Nil(t, err)
	assert.Equal(t, sc.NodeGo, cmd)
	assert.Equal(t, int16(10), runId)
	assert.Equal(t, 0, matchedRuleIdx) // "matched rule 0(go)"

	events[0].NodeStatus = wfmodel.NodeBatchNone
	cmd, _, matchedRuleIdx, err = CheckDependencyPolicyAgainstNodeEventList(nil, fullBatchId, &polDef, events)
	assert.Nil(t, err)
	assert.Equal(t, sc.NodeWait, cmd)
	assert.Equal(t, 1, matchedRuleIdx) // "matched rule 1(wait)")

	events[0].NodeStatus = wfmodel.NodeBatchStart
	cmd, _, matchedRuleIdx, err = CheckDependencyPolicyAgainstNodeEventList(nil, fullBatchId, &polDef, events)
	assert.Nil(t, err)
	assert.Equal(t, sc.NodeWait, cmd)
	assert.Equal(t, 2, matchedRuleIdx) // "matched rule 2(wait)")

	events[0].NodeStatus = wfmodel.NodeBatchFail
	cmd, _, matchedRuleIdx, err = CheckDependencyPolicyAgainstNodeEventList(nil, fullBatchId, &polDef, events)
	assert.Nil(t, err)
	assert.Equal(t, sc.NodeNogo, cmd)
	assert.Equal(t, 3, matchedRuleIdx) // "matched rule 3(nogo)")

	events[0].RunIsCurrent = false

	events[0].NodeStatus = wfmodel.NodeBatchRunStopReceived
	cmd, _, matchedRuleIdx, err = CheckDependencyPolicyAgainstNodeEventList(nil, fullBatchId, &polDef, events)
	assert.Nil(t, err)
	assert.Equal(t, sc.NodeWait, cmd)
	assert.Equal(t, -1, matchedRuleIdx) // "no rules matched against events")

	events[0].NodeStatus = wfmodel.NodeBatchSuccess
	cmd, runId, matchedRuleIdx, err = CheckDependencyPolicyAgainstNodeEventList(nil, fullBatchId, &polDef, events)
	assert.Nil(t, err)
	assert.Equal(t, sc.NodeGo, cmd)
	assert.Equal(t, int16(10), runId)
	assert.Equal(t, 4, matchedRuleIdx) // "matched rule 4(go)")

	events[0].NodeStatus = wfmodel.NodeBatchNone
	cmd, _, matchedRuleIdx, err = CheckDependencyPolicyAgainstNodeEventList(nil, fullBatchId, &polDef, events)
	assert.Nil(t, err)
	assert.Equal(t, sc.NodeWait, cmd)
	assert.Equal(t, 5, matchedRuleIdx) // "matched rule 5(wait)")

	events[0].NodeStatus = wfmodel.NodeBatchStart
	cmd, _, matchedRuleIdx, err = CheckDependencyPolicyAgainstNodeEventList(nil, fullBatchId, &polDef, events)
	assert.Nil(t, err)
	assert.Equal(t, sc.NodeWait, cmd)
	assert.Equal(t, 6, matchedRuleIdx) // "matched rule 6(wait)")

	events[0].RunFinalStatus = wfmodel.RunComplete

	events[0].NodeStatus = wfmodel.NodeBatchSuccess
	cmd, runId, matchedRuleIdx, err = CheckDependencyPolicyAgainstNodeEventList(nil, fullBatchId, &polDef, events)
	assert.Nil(t, err)
	assert.Equal(t, sc.NodeGo, cmd)
	assert.Equal(t, int16(10), runId)
	assert.Equal(t, 7, matchedRuleIdx) // "matched rule 7(go)")

	events[0].NodeStatus = wfmodel.NodeBatchFail
	cmd, _, matchedRuleIdx, err = CheckDependencyPolicyAgainstNodeEventList(nil, fullBatchId, &polDef, events)
	assert.Nil(t, err)
	assert.Equal(t, sc.NodeNogo, cmd)
	assert.Equal(t, 8, matchedRuleIdx) // "matched rule 8(nogo)")

	// Failures

	re := regexp.MustCompile(`"expression": "e\.run[^"]+"`)
	err = polDef.Deserialize([]byte(re.ReplaceAllString(sc.DefaultPolicyCheckerConfJson, `"expression": "1"`)), sc.ScriptJson)
	assert.Nil(t, err)
	_, _, _, err = CheckDependencyPolicyAgainstNodeEventList(nil, fullBatchId, &polDef, events)
	assert.Contains(t, err.Error(), "expected result type was bool, got int64")
}
