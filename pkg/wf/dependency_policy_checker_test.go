package wf

import (
	"testing"
	"time"

	"github.com/capillariesio/capillaries/pkg/env"
	"github.com/capillariesio/capillaries/pkg/l"
	"github.com/capillariesio/capillaries/pkg/sc"
	"github.com/capillariesio/capillaries/pkg/wfmodel"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestDependencyPolicyChecker(t *testing.T) {
	logger, err := l.NewLoggerFromEnvConfig(&env.EnvConfig{HandlerExecutableType: "TestDependencyPolicyChecker", ZapConfig: zap.NewDevelopmentConfig()})
	if err != nil {
		t.Error(err.Error())
		return
	}

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

	events[0].RunIsCurrent = true

	events[0].NodeStatus = wfmodel.NodeBatchSuccess
	cmd, runId, _ := CheckDependencyPolicyAgainstNodeEventList(logger, &polDef, events)
	assert.Equal(t, sc.NodeGo, cmd)
	assert.Equal(t, int16(10), runId)

	events[0].NodeStatus = wfmodel.NodeBatchNone
	cmd, _, _ = CheckDependencyPolicyAgainstNodeEventList(logger, &polDef, events)
	assert.Equal(t, sc.NodeWait, cmd)

	events[0].NodeStatus = wfmodel.NodeBatchStart
	cmd, _, _ = CheckDependencyPolicyAgainstNodeEventList(logger, &polDef, events)
	assert.Equal(t, sc.NodeWait, cmd)

	events[0].NodeStatus = wfmodel.NodeBatchFail
	cmd, _, _ = CheckDependencyPolicyAgainstNodeEventList(logger, &polDef, events)
	assert.Equal(t, sc.NodeNogo, cmd)

	events[0].RunIsCurrent = false

	events[0].NodeStatus = wfmodel.NodeBatchSuccess
	cmd, runId, _ = CheckDependencyPolicyAgainstNodeEventList(logger, &polDef, events)
	assert.Equal(t, sc.NodeGo, cmd)
	assert.Equal(t, int16(10), runId)

	events[0].NodeStatus = wfmodel.NodeBatchNone
	cmd, _, _ = CheckDependencyPolicyAgainstNodeEventList(logger, &polDef, events)
	assert.Equal(t, sc.NodeWait, cmd)

	events[0].NodeStatus = wfmodel.NodeBatchStart
	cmd, _, _ = CheckDependencyPolicyAgainstNodeEventList(logger, &polDef, events)
	assert.Equal(t, sc.NodeWait, cmd)

	events[0].RunFinalStatus = wfmodel.RunComplete

	events[0].NodeStatus = wfmodel.NodeBatchSuccess
	cmd, runId, _ = CheckDependencyPolicyAgainstNodeEventList(logger, &polDef, events)
	assert.Equal(t, sc.NodeGo, cmd)
	assert.Equal(t, int16(10), runId)

	events[0].NodeStatus = wfmodel.NodeBatchFail
	cmd, _, _ = CheckDependencyPolicyAgainstNodeEventList(logger, &polDef, events)
	assert.Equal(t, sc.NodeNogo, cmd)
}
