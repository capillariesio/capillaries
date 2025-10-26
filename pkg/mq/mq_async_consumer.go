package mq

import (
	"github.com/capillariesio/capillaries/pkg/l"
	"github.com/capillariesio/capillaries/pkg/wfmodel"
)

type AcknowledgerCmd int

const (
	AcknowledgerCmdAck AcknowledgerCmd = iota
	AcknowledgerCmdRetry
	AcknowledgerCmdHeartbeat
)

type AknowledgerToken struct {
	MsgId string
	Cmd   AcknowledgerCmd
}

type MqAsyncConsumer interface {
	Start(logger *l.CapiLogger, listenerChannel chan *wfmodel.Message, acknowledgerChannel chan AknowledgerToken) error
	StopListener(logger *l.CapiLogger) error
	StopAcknowledger(logger *l.CapiLogger) error
	SupportsHearbeat() bool
}
