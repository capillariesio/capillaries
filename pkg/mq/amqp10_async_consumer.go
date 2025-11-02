package mq

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	amqp10 "github.com/Azure/go-amqp"
	"github.com/capillariesio/capillaries/pkg/l"
	"github.com/capillariesio/capillaries/pkg/wfmodel"
)

type AckMethodType string

const (
	AckMethodRelease AckMethodType = "release"
	AckMethodReject  AckMethodType = "reject"
	AckMethodUnknown AckMethodType = "unknown"
)

func StringToAckMethod(s string) (AckMethodType, error) {
	switch s {
	case string(AckMethodReject):
		return AckMethodReject, nil
	case string(AckMethodRelease):
		return AckMethodRelease, nil
	default:
		return AckMethodUnknown, fmt.Errorf("unknown ack method '%s'", s)
	}
}

const Amqp10ListenerOpenTimeout time.Duration = 2000
const Amqp10ListenerReceiveTimeout time.Duration = 2000
const Amqp10ListenerAckTimeout time.Duration = 2000
const Amqp10ListenerCloseTimeout time.Duration = 2000
const Amqp10ListenerReconnectTimeout time.Duration = 2000
const Amqp10ListenerTotalTimeout time.Duration = Amqp10ListenerOpenTimeout + Amqp10ListenerReceiveTimeout + Amqp10ListenerAckTimeout + Amqp10ListenerCloseTimeout + Amqp10ListenerReconnectTimeout + 1000

const Amqp10AcknowledgerOpenTimeout time.Duration = 2000
const Amqp10AcknowledgerAckTimeout time.Duration = 2000
const Amqp10AcknowledgerCloseTimeout time.Duration = 2000
const Amqp10AcknowledgerReconnectTimeout time.Duration = 2000
const Amqp10AcknowledgerTotalTimeout time.Duration = Amqp10AcknowledgerOpenTimeout + Amqp10AcknowledgerAckTimeout + Amqp10AcknowledgerCloseTimeout + Amqp10AcknowledgerReconnectTimeout + 1000

const Amqp10FullListenerChannelTimeout time.Duration = 50

// The idea behind this async consumer is to Receive AMQP messages with one go-amqp receiver (we call it listener),
// and Ack/Retry AMQP messages with another go-amqp receiver (we call it acknoledger). This way,
// we do not need to introduce any multi-thread protection for the Open/Close code.
// It uses amqpMessagesInHandlingMutex for protecting the helper map, which is cheap.
// To work properly, and gracefully close async consumer, the caller should do this:
// asyncConsumer.Start(listenerChannel, acknowledgerChannel)
// ... caller reads messages from listenerChannel, passes them to processors that write to acknowledgerChannel
// asyncConsumer.StopListener()
// close(listenerChannel)
// waitForAllProcessorsToCompleteSoTheyDoNotWriteToAcknowledgerChannel
// asyncConsumer.StopAcknowledger()
// close(acknowledgerChannel)
// The size of listenerChannel should correlate with the number of processor threads
// The size of acknowledgerChannel is between 1 and any reasonale value <= number of processor threads
type Amqp10AsyncConsumer struct {
	url                         string
	address                     string
	ackMethod                   AckMethodType
	credit                      int32
	maxProcessors               int
	activeProcessors            atomic.Int64
	listener                    Amqp10Consumer
	acknowledger                Amqp10Consumer
	listenerStopping            bool
	acknowledgerStopping        bool
	amqpMessagesInHandling      map[string]*amqp10.Message
	amqpMessagesInHandlingMutex sync.RWMutex
}

func NewAmqp10Consumer(url string, address string, credit int32, ackMethod AckMethodType, maxProcessors int) *Amqp10AsyncConsumer {
	return &Amqp10AsyncConsumer{
		url:                         url,
		address:                     address,
		ackMethod:                   ackMethod,
		credit:                      credit,
		maxProcessors:               maxProcessors,
		listener:                    Amqp10Consumer{},
		acknowledger:                Amqp10Consumer{},
		listenerStopping:            false,
		acknowledgerStopping:        false,
		amqpMessagesInHandling:      map[string]*amqp10.Message{},
		amqpMessagesInHandlingMutex: sync.RWMutex{},
	}
}

func (dc *Amqp10AsyncConsumer) listenerWorker(logger *l.CapiLogger, listenerChannel chan *wfmodel.Message) {
	logger.PushF("Amqp10AsyncConsumer.listenerWorker")
	defer logger.PopF()

	for !dc.listenerStopping {
		// Do not claim until at least one procesor is ready, otherwise we risk a msg sitting
		// in the channel without sending heartbits, so by the time a processor start handling it,
		// CapiMQ already may consider it dead
		if int(dc.activeProcessors.Load()) == dc.maxProcessors {
			time.Sleep(Amqp10FullListenerChannelTimeout * time.Millisecond)
			continue
		}
		if !dc.listener.isOpen() {
			openCtx, openCancel := context.WithTimeout(context.Background(), Amqp10ListenerOpenTimeout*time.Millisecond)
			if err := dc.listener.open(openCtx, dc.url, dc.address, dc.credit); err != nil {
				logger.Error("cannot reconnect to %s, address %s, credit %d: %s", dc.url, dc.address, dc.credit, err.Error())
				time.Sleep(Amqp10ListenerReconnectTimeout * time.Millisecond)
			}
			openCancel()
		}

		if dc.listener.isOpen() {
			recCtx, recCancel := context.WithTimeout(context.Background(), Amqp10ListenerReceiveTimeout*time.Millisecond)
			amqpMsg, recErr := dc.listener.receiver.Receive(recCtx, nil)
			recCancel()
			if recErr == nil {
				var wfmodelMsg wfmodel.Message
				if err := json.Unmarshal(slices.Concat(amqpMsg.Data...), &wfmodelMsg); err != nil {
					logger.Error("cannot unmarshal wfmodel.Message, will ack this mq message: %s, %v", err.Error(), amqpMsg)
					ackCtx, ackCancel := context.WithTimeout(context.Background(), Amqp10ListenerAckTimeout*time.Millisecond)
					if err = dc.listener.receiver.AcceptMessage(ackCtx, amqpMsg); err != nil {
						logger.Error("cannot ack unmarshaled mq message, will abandon it: %s", err.Error())
					}
					ackCancel()
				} else {
					dc.amqpMessagesInHandlingMutex.Lock()
					dc.amqpMessagesInHandling[wfmodelMsg.Id] = amqpMsg
					mapSize := len(dc.amqpMessagesInHandling)
					dc.amqpMessagesInHandlingMutex.Unlock()
					// Safeguard, dev error: we do not expect a 1000-powerful processor thread pool
					if mapSize > int(dc.credit) && mapSize%1000 == 0 {
						logger.Warn("unexpected: listener map size is too big (%d), max expected %d", mapSize, dc.credit)
					}
					// WARNING: make sure the caller does not close listenerChannel before listenerWorker() completes
					listenerChannel <- &wfmodelMsg
					dc.activeProcessors.Add(1)

				}
			} else {
				if recErr != context.DeadlineExceeded {
					connError := &amqp10.ConnError{}
					sessionError := &amqp10.SessionError{}
					linkError := &amqp10.LinkError{}
					if errors.As(recErr, &connError) || errors.As(recErr, &sessionError) || errors.As(recErr, &linkError) {
						// Connectivity error or RabbitMQ complaining about queue not found (linkError)
						logger.Error("cannot receive, connectivity error: %s", recErr.Error())
					} else {
						logger.Error("cannot receive, unknown error: %s", recErr.Error())
					}
					closeCtx, closeCancel := context.WithTimeout(context.Background(), Amqp10ListenerCloseTimeout*time.Millisecond)
					if err := dc.listener.close(closeCtx); err != nil {
						logger.Error("cannot properly close after failed receive: %s", err.Error())
					}
					closeCancel()
				}
				time.Sleep(Amqp10FullListenerChannelTimeout * time.Millisecond)
			}
		}
	}

	// Cleanup on exit
	if dc.listener.isOpen() {
		closeCtx, closeCancel := context.WithTimeout(context.Background(), Amqp10ListenerCloseTimeout*time.Millisecond)
		if err := dc.listener.close(closeCtx); err != nil {
			logger.Error("cannot properly close on exit: %s", err.Error())
		}
		closeCancel()
	}

	// Signal that listener is done
	dc.listener.done <- true
}

func (dc *Amqp10AsyncConsumer) acknowledgerWorker(logger *l.CapiLogger, acknowledgerChannel chan AknowledgerToken) {
	logger.PushF("Amqp10AsyncConsumer.aknowledgerWorker")
	defer logger.PopF()

	for !dc.acknowledgerStopping {
		timeoutChannel := make(chan bool, 1)
		go func() {
			time.Sleep(1 * time.Second)
			timeoutChannel <- true
		}()
		select {
		// WARNING: make sure the caller does not close acknowledgerChannel before acknowledgerWorker() completes
		case token := <-acknowledgerChannel:
			if !dc.acknowledger.isOpen() {
				// We intentionally set acknowledger.receiver's linkCredit to -1 because go-amqp implementation
				// reacts to it with autoSendFlow=false, effectively setting AMQP prefetch to zero,
				// so acknowledger.receiver never prefetches any AMQP messages from the broker
				// (and it should not, because it only has to Ack and Retry). An attempt to run acknowledger.receiver
				// with autoSendFlow=true may result in messages stuck in limbo state - they are prefetched for
				// acknowledger.receiver, but never claimed by acknowledger.receiver.Receive().
				// Fun fact: go-amqp receiver.Receive() with linkCredit=-1 returns context.DeadlineExceeded,
				// not amqp:link:transfer-limit-exceeded or resource-limit-exceeded.
				openCtx, openCancel := context.WithTimeout(context.Background(), Amqp10AcknowledgerOpenTimeout*time.Millisecond)
				if err := dc.acknowledger.open(openCtx, dc.url, dc.address, -1); err != nil {
					logger.Error("cannot reconnect to %s, address %s: %s", dc.url, dc.address, err.Error())
					time.Sleep(Amqp10AcknowledgerReconnectTimeout * time.Millisecond)
				}
				openCancel()
			}
			if dc.acknowledger.isOpen() {
				dc.amqpMessagesInHandlingMutex.RLock()
				amqpMsg, ok := dc.amqpMessagesInHandling[token.MsgId]
				dc.amqpMessagesInHandlingMutex.RUnlock()
				if ok {
					dc.amqpMessagesInHandlingMutex.Lock()
					delete(dc.amqpMessagesInHandling, token.MsgId)
					dc.amqpMessagesInHandlingMutex.Unlock()
					ackCtx, ackCancel := context.WithTimeout(context.Background(), Amqp10AcknowledgerAckTimeout*time.Millisecond)
					var ackError error
					switch token.Cmd {
					case AcknowledgerCmdAck:
						dc.activeProcessors.Add(-1)
						if ackError = dc.acknowledger.receiver.AcceptMessage(ackCtx, amqpMsg); ackError != nil {
							logger.Error("cannot ack, expect some daemon instance to perform DeleteDataAndUniqueIndexesByBatchIdx for %s: %s", token.MsgId, ackError.Error())
						}
					case AcknowledgerCmdRetry:
						// ActiveMQ Artemis:
						// - Reject makes Artemis put the msg to DLQ without honoring redelivery-delay or discard it (if no DLQ configured) regardless of other settings
						// - Release works
						// For troubleshooting, whatch for Artemis warnings:
						// - AMQ222149: Sending message ... to Dead Letter Address DLQ from ...
						// - AMQ222150: Sending message ... to Dead Letter Address, but there is no Dead Letter Address configured for queue ... so dropping it
						// ActiveMQ classic:
						// - Reject works
						// - Release does not trigger configured redeliveryPlugin (see ./test/docker/activemq/classic/activemq.xml)
						// RabbitMQ 4:
						// - Reject works, see some details About Release/Reject at https://www.rabbitmq.com/docs/amqp
						// - Release does not trigger the dead letter queue process configured in docker-compose.yml
						dc.activeProcessors.Add(-1)
						switch dc.ackMethod {
						case AckMethodReject:
							if ackError = dc.acknowledger.receiver.RejectMessage(ackCtx, amqpMsg, &amqp10.Error{Condition: amqp10.ErrCondInternalError, Description: fmt.Sprintf("capidaemon %s asked to retry", logger.ZapMachine.String)}); ackError != nil {
								logger.Error("cannot retry(reject), expect some daemon instance to perform DeleteDataAndUniqueIndexesByBatchIdx for %s: %s", token.MsgId, ackError.Error())
							}
						case AckMethodRelease:
							if ackError = dc.acknowledger.receiver.ReleaseMessage(ackCtx, amqpMsg); ackError != nil {
								logger.Error("cannot retry(release), expect some daemon instance to perform DeleteDataAndUniqueIndexesByBatchIdx for %s: %s", token.MsgId, ackError.Error())
							}
						default:
							logger.Error("invalid ack_method configuration: %s", dc.ackMethod)
						}
					case AcknowledgerCmdHeartbeat:
						logger.Error("unexpected acknowledger heartbeat cmd, it is not supported by AMQP message brokers")
					default:
						logger.Error("unexpected acknowledger cmd %d", token.Cmd)
					}
					ackCancel()

					if ackError != nil {
						connError := &amqp10.ConnError{}
						sessionError := &amqp10.SessionError{}
						linkError := &amqp10.LinkError{}
						if errors.As(ackError, &connError) || errors.As(ackError, &sessionError) || errors.As(ackError, &linkError) {
							// Biz as usual, do not bother logging here, open() call above will log an error if any
							closeCtx, closeCancel := context.WithTimeout(context.Background(), Amqp10AcknowledgerCloseTimeout*time.Millisecond)
							if err := dc.acknowledger.close(closeCtx); err != nil {
								logger.Error("cannot properly close after failed ack/release: %s", err.Error())
							}
							closeCancel()
						} else if ackError != context.DeadlineExceeded {
							logger.Error("cannot ack/retry, unknown error, will not reconnect: %s", ackError.Error())
						}
					}
				}
			}
		case <-timeoutChannel:
			// Biz as usual, check for acknowledgerStopping and select again if still in the loop
		}

	}

	// Cleanup on exit
	if dc.acknowledger.isOpen() {
		closeCtx, closeCancel := context.WithTimeout(context.Background(), Amqp10AcknowledgerCloseTimeout*time.Millisecond)
		if err := dc.acknowledger.close(closeCtx); err != nil {
			logger.Error("cannot properly close on exit: %s", err.Error())
		}
		closeCancel()
	}

	// Signal that Acknowledger is done
	dc.acknowledger.done <- true
}

func (dc *Amqp10AsyncConsumer) Start(logger *l.CapiLogger, listenerChannel chan *wfmodel.Message, acknowledgerChannel chan AknowledgerToken) error {
	listenerLogger, err := l.NewLoggerFromLogger(logger)
	if err != nil {
		return err
	}
	go dc.listenerWorker(listenerLogger, listenerChannel)

	acknowledgererLogger, err := l.NewLoggerFromLogger(logger)
	if err != nil {
		return err
	}
	go dc.acknowledgerWorker(acknowledgererLogger, acknowledgerChannel)

	return nil
}

func (dc *Amqp10AsyncConsumer) StopListener(logger *l.CapiLogger) error {
	dc.listenerStopping = true

	timeoutChannel := make(chan bool, 1)
	go func() {
		time.Sleep(Amqp10ListenerTotalTimeout * time.Second)
		timeoutChannel <- true
	}()
	select {
	case <-dc.listener.done:
		// Happy path, the caller can close the listener channel
		return nil
	case <-timeoutChannel:
		return fmt.Errorf("cannot stop Listener gracefully, caller closing Listener channel may result in panic")
	}
}

func (dc *Amqp10AsyncConsumer) StopAcknowledger(logger *l.CapiLogger) error {
	dc.acknowledgerStopping = true

	timeoutChannel := make(chan bool, 1)
	go func() {
		time.Sleep(Amqp10AcknowledgerTotalTimeout * time.Second)
		timeoutChannel <- true
	}()
	select {
	case <-dc.acknowledger.done:
		// Happy path, the caller can close the Acknowledger channel
		return nil
	case <-timeoutChannel:
		return fmt.Errorf("cannot stop Acknowledger gracefully, caller closing Acknowledger channel may result in panic")
	}
}

func (dc *Amqp10AsyncConsumer) SupportsHearbeat() bool {
	return false
}
