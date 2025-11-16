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
const Amqp10ListenerDrainTimeout time.Duration = 5000
const Amqp10ListenerTotalTimeout time.Duration = Amqp10ListenerOpenTimeout + Amqp10ListenerReceiveTimeout + Amqp10ListenerAckTimeout + Amqp10ListenerCloseTimeout + Amqp10ListenerReconnectTimeout + Amqp10ListenerDrainTimeout + 1000

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
	url       string
	address   string
	ackMethod AckMethodType
	// https://www.rabbitmq.com/blog/2024/09/02/amqp-flow-control:
	// Granting link credit is not cumulative. When the receiver sends a flow frame with link-credit = N, the receiver sets the
	// current credit to N instead of adding N more credits. For example, if a receiver sends two flow frames with link-credit = 50
	// without any messages being transferred in between, the receiver will have 50 credits, not 100.
	// Absolute min for this value is ThredPoolSize otherwise some threads will be idle. Also, this number should be large enough
	// to avoid often IssueCredit calls when the daemon has to handle a wave of messages that result in "wait, this node is noready yet" status.
	// While IssueCredit is cheap by itself, it triggers an AMQP 1.0 "flow" command, which definitely takes resources.
	// A flow value (maxCreditWindow) too high (apparently) leads to a scenario when a machine prefetches a lot of "heavy" batches,
	// while other machines are quickly handling "light" batches and end up idling until the "heavy-batch" machine finishes a Capillaries node
	// and batches for the next node are allowed to be handled by all machines.
	minCreditWindow             uint32
	maxCreditWindow             uint32
	maxProcessors               int
	activeProcessors            atomic.Int32
	listener                    Amqp10Consumer
	acknowledger                Amqp10Consumer
	listenerStopping            bool
	acknowledgerStopping        bool
	amqpMessagesInHandling      map[string]*amqp10.Message
	amqpMessagesInHandlingMutex sync.RWMutex
	listenerCreditTracker       uint32
	// retryTracker                atomic.Uint32
	// useManualFlow               bool
}

func NewAmqp10Consumer(url string, address string, ackMethod AckMethodType, maxProcessors int, minCreditWindow uint32, _useManualFlow bool) *Amqp10AsyncConsumer {
	return &Amqp10AsyncConsumer{
		url:                         url,
		address:                     address,
		ackMethod:                   ackMethod,
		minCreditWindow:             minCreditWindow,
		maxCreditWindow:             minCreditWindow + minCreditWindow/2,
		maxProcessors:               maxProcessors,
		listener:                    Amqp10Consumer{},
		acknowledger:                Amqp10Consumer{},
		listenerStopping:            false,
		acknowledgerStopping:        false,
		amqpMessagesInHandling:      map[string]*amqp10.Message{},
		amqpMessagesInHandlingMutex: sync.RWMutex{},
		//listenerCreditTracker:       0,
		//useManualFlow:               useManualFlow,
	}
}

// func (dc *Amqp10AsyncConsumer) listenerWorker(logger *l.CapiLogger, listenerChannel chan *wfmodel.Message) {
// 	logger.PushF("Amqp10AsyncConsumer.listenerWorker")
// 	defer logger.Close()

// 	for !dc.listenerStopping {
// 		// Do not claim until at least one procesor is ready, otherwise we risk a msg sitting
// 		// in the channel without sending heartbits, so by the time a processor start handling it,
// 		// CapiMQ already may consider it dead
// 		if int(dc.activeProcessors.Load()) == dc.maxProcessors {
// 			time.Sleep(Amqp10FullListenerChannelTimeout * time.Millisecond)
// 			continue
// 		}
// 		if !dc.listener.isOpen() {
// 			openCtx, openCancel := context.WithTimeout(context.Background(), Amqp10ListenerOpenTimeout*time.Millisecond)
// 			// dc.useManualFlow: we control listener flow ourselves via IssueCredit, so set link credits to -1
// 			linkCredit := int32(10000000) // Yes, we are desperate
// 			if dc.useManualFlow {
// 				linkCredit = -1
// 			}
// 			openErr := dc.listener.open(openCtx, dc.url, dc.address, linkCredit)
// 			openCancel()
// 			if openErr != nil {
// 				logger.Error("cannot reconnect to %s, address %s, credit %d: %s", dc.url, dc.address, linkCredit, openErr.Error())
// 				time.Sleep(Amqp10ListenerReconnectTimeout * time.Millisecond)
// 			} else {
// 				if dc.useManualFlow {
// 					issueErr := dc.listener.receiver.IssueCredit(dc.maxCreditWindow)
// 					if issueErr != nil {
// 						logger.Error("cannot issue credit %d to listener after open: %s", dc.maxCreditWindow, issueErr.Error())
// 						// We cannot proceed without topping-up the credit, so reconnect
// 						closeCtx, closeCancel := context.WithTimeout(context.Background(), Amqp10ListenerCloseTimeout*time.Millisecond)
// 						if err := dc.listener.close(closeCtx); err != nil {
// 							logger.Error("cannot properly close after failed issueCredit after open: %s", err.Error())
// 						}
// 						closeCancel()
// 					} else {
// 						logger.Info("successfully issued initial listener credit of %d after open", dc.maxCreditWindow)
// 						dc.listenerCreditTracker = dc.maxCreditWindow
// 						dc.retryTracker.Store(0)
// 					}
// 				}
// 			}
// 		}

// 		if dc.useManualFlow && dc.listener.isOpen() && dc.retryTracker.Load() == dc.maxCreditWindow {
// 			// This consumer is stuck processing msgs that are kept being postponed. Message broker somehow decided that.
// 			// Try a "soft reset" by draining credit and getting a new one.
// 			logger.Warn("%d consecutive msgs were retried, assuming msg pollution, will drain...", dc.maxCreditWindow)
// 			drainCtx, drainCancel := context.WithTimeout(context.Background(), Amqp10ListenerDrainTimeout*time.Millisecond)
// 			drainErr := dc.listener.receiver.DrainCredit(drainCtx, nil)
// 			drainCancel()
// 			dc.retryTracker.Store(0)
// 			if drainErr != nil {
// 				logger.Error("cannot drain listener: %s", drainErr.Error())
// 				closeCtx, closeCancel := context.WithTimeout(context.Background(), Amqp10ListenerCloseTimeout*time.Millisecond)
// 				if err := dc.listener.close(closeCtx); err != nil {
// 					logger.Error("cannot properly close after failed drain: %s", err.Error())
// 				}
// 				closeCancel()
// 			} else {
// 				// Drain was successful, issue new credit
// 				issueErr := dc.listener.receiver.IssueCredit(dc.maxCreditWindow)
// 				if issueErr != nil {
// 					logger.Error("cannot issue credit to listener after drain: %s", issueErr.Error())
// 					// We cannot proceed without the credit, so reconnect
// 					closeCtx, closeCancel := context.WithTimeout(context.Background(), Amqp10ListenerCloseTimeout*time.Millisecond)
// 					if err := dc.listener.close(closeCtx); err != nil {
// 						logger.Error("cannot properly close after failed issueCredit after receive: %s", err.Error())
// 					}
// 					closeCancel()
// 				} else {
// 					logger.Info("successfully issued listener credit of %d after drain", dc.maxCreditWindow)
// 					dc.listenerCreditTracker = dc.maxCreditWindow
// 				}
// 			}
// 		}

// 		if dc.listener.isOpen() {
// 			recCtx, recCancel := context.WithTimeout(context.Background(), Amqp10ListenerReceiveTimeout*time.Millisecond)
// 			amqpMsg, recErr := dc.listener.receiver.Receive(recCtx, nil)
// 			recCancel()
// 			if recErr == nil {
// 				var wfmodelMsg wfmodel.Message
// 				if err := json.Unmarshal(slices.Concat(amqpMsg.Data...), &wfmodelMsg); err != nil {
// 					logger.Error("cannot unmarshal wfmodel.Message, will ack this mq message: %s, %v", err.Error(), amqpMsg)
// 					ackCtx, ackCancel := context.WithTimeout(context.Background(), Amqp10ListenerAckTimeout*time.Millisecond)
// 					if err = dc.listener.receiver.AcceptMessage(ackCtx, amqpMsg); err != nil {
// 						logger.Error("cannot ack unmarshaled mq message, will abandon it: %s", err.Error())
// 					}
// 					ackCancel()
// 				} else {
// 					dc.amqpMessagesInHandlingMutex.Lock()
// 					dc.amqpMessagesInHandling[wfmodelMsg.Id] = amqpMsg
// 					dc.amqpMessagesInHandlingMutex.Unlock()
// 					// WARNING: make sure the caller does not close listenerChannel before listenerWorker() completes
// 					listenerChannel <- &wfmodelMsg
// 					dc.activeProcessors.Add(1)
// 				}

// 				if dc.useManualFlow {
// 					// Done with the message (or receive/accept error), check our AMQP1.0 flow
// 					// Top-up early to avoid situation when the credit is low, and the consumer cannot receive because not all of the messages within that low credit  were processed
// 					dc.listenerCreditTracker--
// 					if dc.listenerCreditTracker == dc.minCreditWindow {
// 						issueErr := dc.listener.receiver.IssueCredit(dc.maxCreditWindow)
// 						if issueErr != nil {
// 							logger.Error("cannot issue credit to listener after receive: %s", issueErr.Error())
// 							// We cannot proceed without topping-up the credit, so reconnect
// 							closeCtx, closeCancel := context.WithTimeout(context.Background(), Amqp10ListenerCloseTimeout*time.Millisecond)
// 							if err := dc.listener.close(closeCtx); err != nil {
// 								logger.Error("cannot properly close after failed issueCredit after receive: %s", err.Error())
// 							}
// 							closeCancel()
// 						} else {
// 							logger.Info("successfully issued listener credit of %d after receive", dc.maxCreditWindow)
// 							dc.listenerCreditTracker = dc.maxCreditWindow
// 						}
// 					}
// 				}
// 			} else {
// 				if recErr != context.DeadlineExceeded {
// 					connError := &amqp10.ConnError{}
// 					sessionError := &amqp10.SessionError{}
// 					linkError := &amqp10.LinkError{}
// 					if errors.As(recErr, &connError) || errors.As(recErr, &sessionError) || errors.As(recErr, &linkError) {
// 						// Connectivity error or RabbitMQ complaining about queue not found (linkError)
// 						logger.Error("cannot receive, connectivity error: %s", recErr.Error())
// 					} else {
// 						logger.Error("cannot receive, unknown error: %s", recErr.Error())
// 					}
// 					closeCtx, closeCancel := context.WithTimeout(context.Background(), Amqp10ListenerCloseTimeout*time.Millisecond)
// 					if err := dc.listener.close(closeCtx); err != nil {
// 						logger.Error("cannot properly close after failed receive: %s", err.Error())
// 					}
// 					closeCancel()
// 				}
// 				time.Sleep(Amqp10FullListenerChannelTimeout * time.Millisecond)
// 			}
// 		}
// 	}

// 	// Cleanup on exit
// 	if dc.listener.isOpen() {
// 		closeCtx, closeCancel := context.WithTimeout(context.Background(), Amqp10ListenerCloseTimeout*time.Millisecond)
// 		if err := dc.listener.close(closeCtx); err != nil {
// 			logger.Error("cannot properly close on exit: %s", err.Error())
// 		}
// 		closeCancel()
// 	}

// 	// Signal that listener is done
// 	dc.listener.done <- true
// }

func (dc *Amqp10AsyncConsumer) listenerWorker(logger *l.CapiLogger, listenerChannel chan *wfmodel.Message) {
	logger.PushF("Amqp10AsyncConsumer.listenerWorker")
	defer logger.Close()

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
			openErr := dc.listener.open(openCtx, dc.url, dc.address, -1)
			openCancel()
			if openErr != nil {
				logger.Error("cannot reconnect to %s, address %s, credit %d: %s", dc.url, dc.address, -1, openErr.Error())
				time.Sleep(Amqp10ListenerReconnectTimeout * time.Millisecond)
			}
		}

		if dc.listener.isOpen() {
			issueErr := dc.listener.receiver.IssueCredit(1)
			if issueErr != nil {
				logger.Error("cannot issue credit to listener before receive: %s", issueErr.Error())
				// We cannot proceed without topping-up the credit, so reconnect
				closeCtx, closeCancel := context.WithTimeout(context.Background(), Amqp10ListenerCloseTimeout*time.Millisecond)
				if err := dc.listener.close(closeCtx); err != nil {
					logger.Error("cannot properly close after failed issueCredit before receive: %s", err.Error())
				}
				closeCancel()
			} else {
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
						dc.amqpMessagesInHandlingMutex.Unlock()
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
	defer logger.Close()

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
					ackCtx, ackCancel := context.WithTimeout(context.Background(), Amqp10AcknowledgerAckTimeout*time.Millisecond)
					var ackError error
					switch token.Cmd {
					case AcknowledgerCmdAck:
						// dc.retryTracker.Store(0)
						dc.amqpMessagesInHandlingMutex.Lock()
						delete(dc.amqpMessagesInHandling, token.MsgId)
						dc.amqpMessagesInHandlingMutex.Unlock()
						if ackError = dc.acknowledger.receiver.AcceptMessage(ackCtx, amqpMsg); ackError != nil {
							logger.Error("cannot ack, expect some daemon instance to perform DeleteDataAndUniqueIndexesByBatchIdx for %s: %s", token.MsgId, ackError.Error())
						}
					case AcknowledgerCmdRetry:
						//dc.retryTracker.Add(1)
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
						dc.amqpMessagesInHandlingMutex.Lock()
						delete(dc.amqpMessagesInHandling, token.MsgId)
						dc.amqpMessagesInHandlingMutex.Unlock()
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
				} else {
					logger.Error("aknowledger cannot find message: %s", token.MsgId)
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

	acknowledgerLogger, err := l.NewLoggerFromLogger(logger)
	if err != nil {
		return err
	}
	go dc.acknowledgerWorker(acknowledgerLogger, acknowledgerChannel)

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

func (dc *Amqp10AsyncConsumer) DecrementActiveProcessors() {
	dc.activeProcessors.Add(-1)
}
