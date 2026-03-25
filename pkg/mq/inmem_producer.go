package mq

import (
	"slices"

	"github.com/capillariesio/capillaries/pkg/wfmodel"
)

// Used in tests

type TestInmemProducer struct {
	Msgs []*wfmodel.Message
}

// Used in tests

// Receive from queue
func (p *TestInmemProducer) PeekHead() *wfmodel.Message {
	if len(p.Msgs) == 0 {
		return nil
	}
	return p.Msgs[0]
}

// Ack
func (p *TestInmemProducer) RemoveHead() {
	if len(p.Msgs) != 0 {
		p.Msgs = slices.Delete(p.Msgs, 0, 1)
	}
}

// Return to queue
func (p *TestInmemProducer) MoveHeadToTail() {
	if len(p.Msgs) != 0 {
		head := p.Msgs[0]
		p.Msgs = append(slices.Delete(p.Msgs, 0, 1), head)
	}
}

// Implement produces interface

func (p *TestInmemProducer) Open() error {
	return nil
}

func (p *TestInmemProducer) Close() error {
	return nil
}
func (p *TestInmemProducer) Send(msg *wfmodel.Message) error {
	p.Msgs = append(p.Msgs, msg)
	return nil
}

func (p *TestInmemProducer) SendBulk(msgs []*wfmodel.Message) error {
	p.Msgs = append(p.Msgs, msgs...)
	return nil
}

func (p *TestInmemProducer) SupportsSendBulk() bool {
	return true
}
