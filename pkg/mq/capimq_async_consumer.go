package mq

import (
	"github.com/capillariesio/capillaries/pkg/l"
	"github.com/capillariesio/capillaries/pkg/wfmodel"
)

type CapimqAsyncConsumer struct {
	url string
}

func NewCapimqConsumer(url string) *CapimqAsyncConsumer {
	return &CapimqAsyncConsumer{
		url: url,
	}
}

func (dc *CapimqAsyncConsumer) Start(logger *l.CapiLogger, listenerChannel chan *wfmodel.Message, acknowledgerChannel chan AknowledgerToken) error {
	return nil
}

func (dc *CapimqAsyncConsumer) StopListener(logger *l.CapiLogger) error {
	return nil
}

func (dc *CapimqAsyncConsumer) StopAcknowledger(logger *l.CapiLogger) error {
	return nil
}

func (dc *CapimqAsyncConsumer) SupportsHearbeat() bool {
	return true
}
