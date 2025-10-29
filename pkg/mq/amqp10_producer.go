package mq

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	amqp10 "github.com/Azure/go-amqp"
	"github.com/capillariesio/capillaries/pkg/wfmodel"
)

const Amqp10ProducerOpenTimeout time.Duration = 2000
const Amqp10ProducerCloseTimeout time.Duration = 2000
const Amqp10ProducerSendTimeout time.Duration = 2000

type Amqp10Producer struct {
	url     string
	address string
	conn    *amqp10.Conn
	session *amqp10.Session
	sender  *amqp10.Sender
}

func NewAmqp10Producer(url string, address string) *Amqp10Producer {
	return &Amqp10Producer{
		url:     url,
		address: address,
	}
}

func (p *Amqp10Producer) Open() error {
	var err error
	openCtx, openCancel := context.WithTimeout(context.Background(), Amqp10ProducerOpenTimeout*time.Millisecond)
	p.conn, err = amqp10.Dial(openCtx, p.url, nil)
	openCancel()
	if err != nil {
		return err
	}

	openCtx, openCancel = context.WithTimeout(context.Background(), Amqp10ProducerOpenTimeout*time.Millisecond)
	p.session, err = p.conn.NewSession(openCtx, nil)
	openCancel()
	if err != nil {
		p.conn.Close()
		p.conn = nil
		return err
	}
	openCtx, openCancel = context.WithTimeout(context.Background(), Amqp10ProducerOpenTimeout*time.Millisecond)
	p.sender, err = p.session.NewSender(openCtx, p.address, nil)
	openCancel()
	if err != nil {
		closeCtx, closeCancel := context.WithTimeout(context.Background(), Amqp10ProducerCloseTimeout*time.Millisecond)
		p.session.Close(closeCtx)
		closeCancel()
		p.session = nil
		p.conn.Close()
		p.conn = nil
		return err
	}

	return nil
}

func (p *Amqp10Producer) SendBulk(msgs []*wfmodel.Message) error {
	return errors.New("SendBulk not supported")
}

func (p *Amqp10Producer) Send(msg *wfmodel.Message) error {
	if p.sender == nil {
		return fmt.Errorf("cannot send, nil sender")
	}

	msgBytes, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("cannot send, error when serializing msg: %s", err.Error())
	}

	sendCtx, sendCancel := context.WithTimeout(context.Background(), Amqp10ProducerSendTimeout*time.Millisecond)
	err = p.sender.Send(sendCtx, amqp10.NewMessage(msgBytes), nil)
	sendCancel()
	if err != nil {
		return fmt.Errorf("cannot send: %s", err.Error())
	}

	return nil
}

func (p *Amqp10Producer) Close() error {
	sb := strings.Builder{}
	if p.sender != nil {
		closeCtx, closeCancel := context.WithTimeout(context.Background(), Amqp10ProducerCloseTimeout*time.Millisecond)
		if err := p.sender.Close(closeCtx); err != nil {
			sb.WriteString(err.Error() + "; ")
		}
		closeCancel()
	}
	p.sender = nil

	if p.session != nil {
		closeCtx, closeCancel := context.WithTimeout(context.Background(), Amqp10ProducerCloseTimeout*time.Millisecond)
		if err := p.session.Close(closeCtx); err != nil {
			sb.WriteString(err.Error() + "; ")
		}
		closeCancel()
	}
	p.session = nil

	if p.conn != nil {
		if err := p.conn.Close(); err != nil {
			sb.WriteString(err.Error() + "; ")
		}
	}
	p.conn = nil

	if sb.Len() > 0 {
		return errors.New(sb.String())
	}

	return nil
}

func (p *Amqp10Producer) SupportsSendBulk() bool {
	return false
}
