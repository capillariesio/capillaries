package mq

import (
	"context"
	"errors"
	"strings"

	amqp10 "github.com/Azure/go-amqp"
)

type Amqp10Consumer struct {
	conn     *amqp10.Conn
	session  *amqp10.Session
	receiver *amqp10.Receiver
	done     chan bool
}

func (c *Amqp10Consumer) openInternal(ctx context.Context, url string, address string, credits int32) error {
	c.done = make(chan bool)

	var err error
	c.conn, err = amqp10.Dial(ctx, url, nil)
	if err != nil {
		return err
	}

	c.session, err = c.conn.NewSession(ctx, nil)
	if err != nil {
		c.conn.Close()
		c.conn = nil
		return err
	}

	c.receiver, err = c.session.NewReceiver(ctx, address, &amqp10.ReceiverOptions{Credit: credits})
	if err != nil {
		c.session.Close(ctx)
		c.session = nil
		c.conn.Close()
		c.conn = nil
		return err
	}

	return nil
}

func (c *Amqp10Consumer) isOpen() bool {
	return c.conn != nil && c.session != nil && c.receiver != nil
}

func (c *Amqp10Consumer) closeInternal(ctx context.Context) error {
	sb := strings.Builder{}
	if c.receiver != nil {
		if err := c.receiver.Close(ctx); err != nil {
			sb.WriteString(err.Error() + "; ")
		}
	}
	c.receiver = nil

	if c.session != nil {
		if err := c.session.Close(ctx); err != nil {
			sb.WriteString(err.Error() + "; ")
		}
	}
	c.session = nil

	if c.conn != nil {
		if err := c.conn.Close(); err != nil {
			sb.WriteString(err.Error() + "; ")
		}
	}
	c.conn = nil

	if sb.Len() > 0 {
		return errors.New(sb.String())
	}

	return nil
}

func (c *Amqp10Consumer) Open(ctx context.Context, url string, address string) error {
	return c.openInternal(ctx, url, address, 32)
}

func (c *Amqp10Consumer) Receiver() *amqp10.Receiver { return c.receiver }
