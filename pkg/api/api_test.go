package api

import (
	"testing"

	"github.com/capillariesio/capillaries/pkg/custom/py_calc"
	"github.com/capillariesio/capillaries/pkg/custom/tag_and_denormalize"
	"github.com/capillariesio/capillaries/pkg/db"
	"github.com/capillariesio/capillaries/pkg/env"
	"github.com/capillariesio/capillaries/pkg/l"
	"github.com/capillariesio/capillaries/pkg/mq"
	"github.com/capillariesio/capillaries/pkg/sc"
	"github.com/stretchr/testify/assert"
)

type TestProcessorDefFactory struct {
}

func (f *TestProcessorDefFactory) Create(processorType string) (sc.CustomProcessorDef, bool) {
	switch processorType {
	case py_calc.ProcessorPyCalcName:
		return &py_calc.PyCalcProcessorDef{}, true
	case tag_and_denormalize.ProcessorTagAndDenormalizeName:
		return &tag_and_denormalize.TagAndDenormalizeProcessorDef{}, true
	default:
		return nil, false
	}
}

func TestRun(t *testing.T) {
	envConfig := env.EnvConfig{
		Cassandra:                         env.CassandraConfig{WriterWorkers: 1},
		Log:                               env.LogConfig{Level: "INFO"},
		CustomProcessorDefFactoryInstance: &TestProcessorDefFactory{},
		UseGocqlmem:                       true,
	}

	logger, err := l.NewLoggerFromEnvConfig(&envConfig)
	assert.Nil(t, err)

	mqProducer := mq.TestInmemProducer{}

	gocqlmemSession, cassandraEngineType, err := db.NewSession(&envConfig, "testkeyspace", db.CreateKeyspaceOnConnect)
	assert.Nil(t, err)

	_, err = StartRun(&envConfig, logger, &mqProducer, "/tmp/capi_cfg/lookup_quicktest/script_quick.yaml", "/tmp/capi_cfg/lookup_quicktest/script_params_quick_fs_one.yaml", gocqlmemSession, cassandraEngineType, "testkeyspace", []string{"read_orders", "read_order_items"}, "test run")
	assert.Nil(t, err)
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
	}
}
