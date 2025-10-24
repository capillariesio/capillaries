package mq

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"sync"
	"time"

	amqp10 "github.com/Azure/go-amqp"
	"github.com/capillariesio/capillaries/pkg/l"
	"github.com/capillariesio/capillaries/pkg/wfmodel"
)

type MqConsumer interface {
	Open(ctx context.Context, url string, address string) error
	//	StartListening(ctx context.Context, url string, address string, in chan *wfmodel.Message) error
	Close(ctx context.Context) error
	Receive(ctx context.Context) (amqp10.Message, error)
	Ack(ctx context.Context, msg *wfmodel.Message) error
	ReleaseForRetry(ctx context.Context, msg *wfmodel.Message) error
	//Heartbit(ctx context.Context, msg *wfmodel.Message) error
}

type Amqp10Consumer struct {
	conn     *amqp10.Conn
	session  *amqp10.Session
	receiver *amqp10.Receiver
	done     chan bool
}

// func (c *Amqp10Consumer) StartListening(ctx context.Context, url string, address string, in chan *wfmodel.Message) error {
// 	c.url = url
// 	c.address = address
// 	go func(){
// 		// Listen while the channel is open (it will be clased by the caller at some point)
// 		for {
// 			msg, err := c.receiver.Receive(ctx, nil)

// 		}
// 	}
// 	return nil
// }

func (c *Amqp10Consumer) Open(ctx context.Context, url string, address string) error {
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

	c.receiver, err = c.session.NewReceiver(ctx, address, nil)
	if err != nil {
		c.session.Close(ctx)
		c.session = nil
		c.conn.Close()
		c.conn = nil
		return err
	}
	return nil
}

func (c *Amqp10Consumer) IsOpen() bool {
	return c.conn != nil && c.session != nil && c.receiver != nil
}

func (c *Amqp10Consumer) Close(ctx context.Context) error {
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

func (c *Amqp10Consumer) Receive(ctx context.Context) (*amqp10.Message, error) {
	return c.receiver.Receive(ctx, nil)
}

func (c *Amqp10Consumer) Ack(ctx context.Context, msg *amqp10.Message) error {
	return c.receiver.AcceptMessage(ctx, msg)
}

func (c *Amqp10Consumer) ReleaseForRetry(ctx context.Context, msg *amqp10.Message) error {
	return c.receiver.ReleaseMessage(ctx, msg)
}

type AcknowledgerCmd int

const (
	AcknowledgerCmdAck AcknowledgerCmd = iota
	AcknowledgerCmdRetry
)

type AknowledgerToken struct {
	FullBatchId string
	Cmd         AcknowledgerCmd
}
type DuplexAmqp10Consumer struct {
	Url                         string
	Address                     string
	Listener                    Amqp10Consumer
	Acknowledger                Amqp10Consumer
	listenerStopping            bool
	acknowledgerStopping        bool
	amqpMessagesInHandling      map[string]*amqp10.Message
	amqpMessagesInHandlingMutex sync.RWMutex
}

func (dc *DuplexAmqp10Consumer) listenerWorker(logger *l.CapiLogger, listenerChannel chan *wfmodel.Message) {
	logger.PushF("DuplexAmqp10Consumer.listenerWorker")
	defer logger.PopF()

	for !dc.listenerStopping {
		if !dc.Listener.IsOpen() {
			openCtx, openCancel := context.WithTimeout(context.Background(), 5*time.Second)
			if err := dc.Listener.Open(openCtx, dc.Url, dc.Address); err != nil {
				logger.Error("cannot reconnect: %s", err.Error())
			}
			openCancel()
		}

		if dc.Listener.IsOpen() {
			recCtx, recCancel := context.WithTimeout(context.Background(), 5*time.Second)
			amqpMsg, err := dc.Listener.Receive(recCtx)
			recCancel()
			if err == nil {
				var wfmodelMsg wfmodel.Message
				if err = json.Unmarshal(slices.Concat(amqpMsg.Data...), &wfmodelMsg); err != nil {
					logger.Error("cannot unmarshal wfmodel.Message, will ack this mq message: %s, %v", err.Error(), amqpMsg)
					ackCtx, ackCancel := context.WithTimeout(context.Background(), 5*time.Second)
					if err = dc.Listener.ReleaseForRetry(ackCtx, amqpMsg); err != nil {
						logger.Error("cannot ack unmarshaled mq message, will abandon it: %s", err.Error())
					}
					ackCancel()
				} else {
					dc.amqpMessagesInHandlingMutex.Lock()
					dc.amqpMessagesInHandling[wfmodelMsg.FullBatchId()] = amqpMsg
					mapSize := len(dc.amqpMessagesInHandling)
					dc.amqpMessagesInHandlingMutex.Unlock()
					// Safeguard, dev error
					if mapSize > 0 && mapSize%1000 == 0 {
						logger.Warn("unexpected: listener map size is too big: %d", mapSize)
					}
					// WARNING: make sure the caller does not close listenerChannel before listenerWorker() completes
					listenerChannel <- &wfmodelMsg
				}
			} else {
				logger.Error("cannot receive, will reconnect: %s", err.Error())
				closeCtx, closeCancel := context.WithTimeout(context.Background(), 5*time.Second)
				if err = dc.Listener.Close(closeCtx); err != nil {
					logger.Error("cannot properly close before reconnect: %s", err.Error())
				}
				closeCancel()
			}
		}
	}

	// Cleanup on exit
	if dc.Listener.IsOpen() {
		closeCtx, closeCancel := context.WithTimeout(context.Background(), 5*time.Second)
		if err := dc.Listener.Close(closeCtx); err != nil {
			logger.Error("cannot properly close on exit: %s", err.Error())
		}
		closeCancel()
	}

	// Signal that listener is done
	dc.Listener.done <- true
}

func (dc *DuplexAmqp10Consumer) acknowledgerWorker(logger *l.CapiLogger, acknowledgerChannel chan AknowledgerToken) {
	logger.PushF("DuplexAmqp10Consumer.aknowledgerWorker")
	defer logger.PopF()

	for !dc.acknowledgerStopping {
		timeoutChannel := make(chan bool, 1)
		go func() {
			time.Sleep(1 * time.Second)
			timeoutChannel <- true
		}()
		// WARNING: make sure the caller does not close acknowledgerChannel before acknowledgerWorker() completes
		select {
		case token := <-acknowledgerChannel:
			if !dc.Acknowledger.IsOpen() {
				openCtx, openCancel := context.WithTimeout(context.Background(), 5*time.Second)
				if err := dc.Acknowledger.Open(openCtx, dc.Url, dc.Address); err != nil {
					logger.Error("cannot reconnect: %s", err.Error())
				}
				openCancel()
			}
			if dc.Acknowledger.IsOpen() {
				dc.amqpMessagesInHandlingMutex.RLock()
				amqpMsg, ok := dc.amqpMessagesInHandling[token.FullBatchId]
				dc.amqpMessagesInHandlingMutex.RUnlock()
				if ok {
					dc.amqpMessagesInHandlingMutex.Lock()
					delete(dc.amqpMessagesInHandling, token.FullBatchId)
					dc.amqpMessagesInHandlingMutex.Unlock()
					ackCtx, ackCancel := context.WithTimeout(context.Background(), 5*time.Second)
					if token.Cmd == AcknowledgerCmdAck {
						if err := dc.Acknowledger.Ack(ackCtx, amqpMsg); err != nil {
							logger.Error("cannot ack, expect some daemon instance to perform DeleteDataAndUniqueIndexesByBatchIdx for %s: %s", token.FullBatchId, err.Error())
						}
					} else {
						if err := dc.Acknowledger.ReleaseForRetry(ackCtx, amqpMsg); err != nil {
							logger.Error("cannot release for retry, expect some daemon instance to perform DeleteDataAndUniqueIndexesByBatchIdx for %s: %s", token.FullBatchId, err.Error())
						}
					}
					ackCancel()
				}
			}
		case <-timeoutChannel:
			// Biz as usual, check for acknowledgerStopping and select again if still in the loop
		}

	}

	// Cleanup on exit
	if dc.Acknowledger.IsOpen() {
		closeCtx, closeCancel := context.WithTimeout(context.Background(), 5*time.Second)
		if err := dc.Acknowledger.Close(closeCtx); err != nil {
			logger.Error("cannot properly close on exit: %s", err.Error())
		}
		closeCancel()
	}

	// Signal that Acknowledger is done
	dc.Acknowledger.done <- true
}
