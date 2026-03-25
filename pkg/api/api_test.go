package api

import (
	"fmt"
	"testing"

	"github.com/capillariesio/capillaries/pkg/custom/pycalc"
	"github.com/capillariesio/capillaries/pkg/custom/taganddenormalize"
	"github.com/capillariesio/capillaries/pkg/db"
	"github.com/capillariesio/capillaries/pkg/env"
	"github.com/capillariesio/capillaries/pkg/l"
	"github.com/capillariesio/capillaries/pkg/mq"
	"github.com/capillariesio/capillaries/pkg/sc"
	"github.com/capillariesio/capillaries/pkg/wfmodel"
	"github.com/stretchr/testify/assert"
)

type TestProcessorDefFactory struct {
}

func (f *TestProcessorDefFactory) Create(processorType string) (sc.CustomProcessorDef, bool) {
	switch processorType {
	case pycalc.ProcessorPyCalcName:
		return &pycalc.PyCalcProcessorDef{}, true
	case taganddenormalize.ProcessorTagAndDenormalizeName:
		return &taganddenormalize.TagAndDenormalizeProcessorDef{}, true
	default:
		return nil, false
	}
}

func noooooTestRun(t *testing.T) {
	envConfig := env.EnvConfig{
		Cassandra:                         env.CassandraConfig{WriterWorkers: 1},
		Log:                               env.LogConfig{Level: "INFO"},
		CustomProcessorDefFactoryInstance: &TestProcessorDefFactory{},
		UseGocqlmem:                       true,
	}
	sc.ScriptDefCache = sc.NewScriptDefCache()
	NodeDependencyReadynessCache = NewNodeDependencyReadynessCache()

	logger, err := l.NewLoggerFromEnvConfig(&envConfig)
	assert.Nil(t, err)

	mqProducer := mq.TestInmemProducer{}

	gocqlmemSession, cassandraEngineType, err := db.NewSession(&envConfig, "testkeyspace", db.CreateKeyspaceOnConnect)
	assert.Nil(t, err)

	_, err = StartRun(&envConfig, logger, &mqProducer, "/tmp/capi_cfg/lookup_quicktest/script_quick.yaml", "/tmp/capi_cfg/lookup_quicktest/script_params_quick_fs_one.yaml", gocqlmemSession, cassandraEngineType, "testkeyspace", []string{"read_orders", "read_order_items"}, "test run")
	assert.Nil(t, err)

	var runStatus wfmodel.RunStatusType
	runHistory, err := GetRunHistory(logger, gocqlmemSession, "testkeyspace")
	assert.Nil(t, err)

	runStatus = runHistory[len(runHistory)-1].Status
	logger.Info("TestRun.RUNSTATUS: %s", runStatus.ToString())
	assert.Equal(t, wfmodel.RunStart, runStatus)

	var nodeRunStatus string
	for {
		msg := mqProducer.PeekHead()
		if msg == nil {
			break
		}
		ackCmd := ProcessDataBatchMsg(&envConfig, logger, msg, 0, nil)
		if ackCmd == mq.AcknowledgerCmdAck {
			mqProducer.RemoveHead()
		} else {
			mqProducer.MoveHeadToTail()
		}
		runHistory, err := GetRunHistory(logger, gocqlmemSession, "testkeyspace")
		assert.Nil(t, err)

		runStatus = runHistory[len(runHistory)-1].Status
		logger.Info("TestRun.RUNSTATUS: %s", runStatus.ToString())

		nodeHistory, err := GetNodeHistoryForRuns(logger, gocqlmemSession, "testkeyspace", []int16{int16(1)})
		assert.Nil(t, err)

		newNodeRunStatusMap := map[string]wfmodel.NodeBatchStatusType{}
		for _, nodeEvent := range nodeHistory {
			newNodeRunStatusMap[nodeEvent.ScriptNode] = nodeEvent.Status
		}

		newNodeRunStatus := fmt.Sprintf("%v", newNodeRunStatusMap)
		if nodeRunStatus != newNodeRunStatus {
			nodeRunStatus = newNodeRunStatus
			logger.Info("TestRun.NODESTATUS: %s", nodeRunStatus)
		}
	}
	assert.Equal(t, wfmodel.RunComplete, runStatus)
}
