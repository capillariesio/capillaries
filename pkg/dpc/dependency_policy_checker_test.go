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
	if err := polDef.Deserialize([]byte(sc.DefaultPolicyCheckerConf)); err != nil {
		t.Error(err)
		return
	}

	var cmd sc.ReadyToRunNodeCmdType
	var runId int16
	var checkerLogMsg string
	var err error

	events[0].RunIsCurrent = true

	events[0].NodeStatus = wfmodel.NodeBatchRunStopReceived
	cmd, runId, checkerLogMsg, err = CheckDependencyPolicyAgainstNodeEventList(&polDef, events)
	assert.Nil(t, err)
	assert.Equal(t, sc.NodeNogo, cmd)
	assert.Contains(t, checkerLogMsg, "no rules matched against events")

	events[0].NodeStatus = wfmodel.NodeBatchSuccess
	cmd, runId, checkerLogMsg, err = CheckDependencyPolicyAgainstNodeEventList(&polDef, events)
	assert.Nil(t, err)
	assert.Equal(t, sc.NodeGo, cmd)
	assert.Equal(t, int16(10), runId)
	assert.Contains(t, checkerLogMsg, "matched rule 0(go)")

	events[0].NodeStatus = wfmodel.NodeBatchNone
	cmd, _, checkerLogMsg, err = CheckDependencyPolicyAgainstNodeEventList(&polDef, events)
	assert.Nil(t, err)
	assert.Equal(t, sc.NodeWait, cmd)
	assert.Contains(t, checkerLogMsg, "matched rule 1(wait)")

	events[0].NodeStatus = wfmodel.NodeBatchStart
	cmd, _, checkerLogMsg, err = CheckDependencyPolicyAgainstNodeEventList(&polDef, events)
	assert.Nil(t, err)
	assert.Equal(t, sc.NodeWait, cmd)
	assert.Contains(t, checkerLogMsg, "matched rule 2(wait)")

	events[0].NodeStatus = wfmodel.NodeBatchFail
	cmd, _, checkerLogMsg, err = CheckDependencyPolicyAgainstNodeEventList(&polDef, events)
	assert.Nil(t, err)
	assert.Equal(t, sc.NodeNogo, cmd)
	assert.Contains(t, checkerLogMsg, "matched rule 3(nogo)")

	events[0].RunIsCurrent = false

	events[0].NodeStatus = wfmodel.NodeBatchRunStopReceived
	cmd, runId, checkerLogMsg, err = CheckDependencyPolicyAgainstNodeEventList(&polDef, events)
	assert.Nil(t, err)
	assert.Equal(t, sc.NodeNogo, cmd)
	assert.Contains(t, checkerLogMsg, "no rules matched against events")

	events[0].NodeStatus = wfmodel.NodeBatchSuccess
	cmd, runId, checkerLogMsg, err = CheckDependencyPolicyAgainstNodeEventList(&polDef, events)
	assert.Nil(t, err)
	assert.Equal(t, sc.NodeGo, cmd)
	assert.Equal(t, int16(10), runId)
	assert.Contains(t, checkerLogMsg, "matched rule 4(go)")

	events[0].NodeStatus = wfmodel.NodeBatchNone
	cmd, _, checkerLogMsg, err = CheckDependencyPolicyAgainstNodeEventList(&polDef, events)
	assert.Nil(t, err)
	assert.Equal(t, sc.NodeWait, cmd)
	assert.Contains(t, checkerLogMsg, "matched rule 5(wait)")

	events[0].NodeStatus = wfmodel.NodeBatchStart
	cmd, _, checkerLogMsg, err = CheckDependencyPolicyAgainstNodeEventList(&polDef, events)
	assert.Nil(t, err)
	assert.Equal(t, sc.NodeWait, cmd)
	assert.Contains(t, checkerLogMsg, "matched rule 6(wait)")

	events[0].RunFinalStatus = wfmodel.RunComplete

	events[0].NodeStatus = wfmodel.NodeBatchSuccess
	cmd, runId, checkerLogMsg, err = CheckDependencyPolicyAgainstNodeEventList(&polDef, events)
	assert.Nil(t, err)
	assert.Equal(t, sc.NodeGo, cmd)
	assert.Equal(t, int16(10), runId)
	assert.Contains(t, checkerLogMsg, "matched rule 7(go)")

	events[0].NodeStatus = wfmodel.NodeBatchFail
	cmd, _, checkerLogMsg, err = CheckDependencyPolicyAgainstNodeEventList(&polDef, events)
	assert.Nil(t, err)
	assert.Equal(t, sc.NodeNogo, cmd)
	assert.Contains(t, checkerLogMsg, "matched rule 8(nogo)")

	// Failures

	re := regexp.MustCompile(`"expression": "e\.run[^"]+"`)
	err = polDef.Deserialize([]byte(re.ReplaceAllString(sc.DefaultPolicyCheckerConf, `"expression": "1"`)))
	assert.Nil(t, err)
	_, _, _, err = CheckDependencyPolicyAgainstNodeEventList(&polDef, events)
	assert.Contains(t, err.Error(), "expected result type was bool, got int64")
}
