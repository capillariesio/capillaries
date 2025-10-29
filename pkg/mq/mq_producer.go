package mq

import (
	"github.com/capillariesio/capillaries/pkg/wfmodel"
)

type MqProducer interface {
	Open() error
	Close() error
	Send(msg *wfmodel.Message) error
	SendBulk(msgs []*wfmodel.Message) error
	SupportsSendBulk() bool
}
