package mq

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	amqp10 "github.com/Azure/go-amqp"
	"github.com/capillariesio/capillaries/pkg/wfmodel"
)

type Amqp10Producer struct {
	conn    *amqp10.Conn
	session *amqp10.Session
	sender  *amqp10.Sender
}

func (p *Amqp10Producer) Open(ctx context.Context, url string, address string) error {
	var err error
	p.conn, err = amqp10.Dial(ctx, url, nil)
	if err != nil {
		return err
	}

	p.session, err = p.conn.NewSession(ctx, nil)
	if err != nil {
		p.conn.Close()
		p.conn = nil
		return err
	}

	p.sender, err = p.session.NewSender(ctx, address, nil)
	if err != nil {
		p.session.Close(ctx)
		p.session = nil
		p.conn.Close()
		p.conn = nil
		return err
	}

	return nil
}

func (p *Amqp10Producer) SendBulk(ctx context.Context, msgs []*wfmodel.Message) error {
	return errors.New("SendBulk not supported")
}

func (p *Amqp10Producer) Send(ctx context.Context, msg *wfmodel.Message) error {
	if p.sender == nil {
		return fmt.Errorf("cannot send, nil sender")
	}

	msgBytes, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("cannot send, error when serializing msg: %s", err.Error())
	}

	err = p.sender.Send(ctx, amqp10.NewMessage(msgBytes), nil)
	if err != nil {
		return fmt.Errorf("cannot send: %s", err.Error())
	}

	return nil
}

func (p *Amqp10Producer) Close(ctx context.Context) error {
	sb := strings.Builder{}
	if p.sender != nil {
		if err := p.sender.Close(ctx); err != nil {
			sb.WriteString(err.Error() + "; ")
		}
	}
	p.sender = nil

	if p.session != nil {
		if err := p.session.Close(ctx); err != nil {
			sb.WriteString(err.Error() + "; ")
		}
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
