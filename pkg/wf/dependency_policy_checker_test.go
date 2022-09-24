package wf

import (
	"testing"
	"time"

	"github.com/kleineshertz/capillaries/pkg/env"
	"github.com/kleineshertz/capillaries/pkg/l"
	"github.com/kleineshertz/capillaries/pkg/sc"
	"github.com/kleineshertz/capillaries/pkg/wfmodel"
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
			RunId:         10,
			RunIsCurrent:  true,
			RunStartTs:    time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
			RunStatus:     wfmodel.RunStart,
			RunStatusTs:   time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
			NodeIsStarted: true,
			NodeStartTs:   time.Date(2000, 1, 1, 0, 0, 1, 0, time.UTC),
			NodeStatus:    wfmodel.NodeBatchNone,
			NodeStatusTs:  time.Date(2000, 1, 1, 0, 0, 2, 0, time.UTC)}}
	conf := `{"event_priority_order": "run_is_current(desc),node_start_ts(desc)",
		"rules": [
		{"cmd": "go",   "expression": "e.run_is_current == true && e.run_status == wfmodel.RunStart && e.node_status == wfmodel.NodeBatchSuccess"	},
		{"cmd": "wait", "expression": "e.run_is_current == true && e.run_status == wfmodel.RunStart && e.node_status == wfmodel.NodeBatchNone"	    },
		{"cmd": "wait", "expression": "e.run_is_current == true && e.run_status == wfmodel.RunStart && e.node_status == wfmodel.NodeBatchStart"	    },
		{"cmd": "nogo", "expression": "e.run_is_current == true && e.run_status == wfmodel.RunStart && e.node_status == wfmodel.NodeBatchFail"	    },

		{"cmd": "go",   "expression": "e.run_is_current == false && e.run_status == wfmodel.RunStart && e.node_status == wfmodel.NodeBatchSuccess"	},
		{"cmd": "wait",   "expression": "e.run_is_current == false && e.run_status == wfmodel.RunStart && e.node_status == wfmodel.NodeBatchNone"	},
		{"cmd": "wait",   "expression": "e.run_is_current == false && e.run_status == wfmodel.RunStart && e.node_status == wfmodel.NodeBatchStart"	},

		{"cmd": "go",   "expression": "e.run_is_current == false && e.run_status == wfmodel.RunComplete && e.node_status == wfmodel.NodeBatchSuccess"	},
		{"cmd": "nogo",   "expression": "e.run_is_current == false && e.run_status == wfmodel.RunComplete && e.node_status == wfmodel.NodeBatchFail"	}
	]}`
	polDef := sc.DependencyPolicyDef{}
	if err := polDef.Deserialize([]byte(conf)); err != nil {
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

	events[0].RunStatus = wfmodel.RunComplete

	events[0].NodeStatus = wfmodel.NodeBatchSuccess
	cmd, runId, _ = CheckDependencyPolicyAgainstNodeEventList(logger, &polDef, events)
	assert.Equal(t, sc.NodeGo, cmd)
	assert.Equal(t, int16(10), runId)

	events[0].NodeStatus = wfmodel.NodeBatchFail
	cmd, _, _ = CheckDependencyPolicyAgainstNodeEventList(logger, &polDef, events)
	assert.Equal(t, sc.NodeNogo, cmd)
}
