package wf

import (
	"fmt"
	"os"
	"time"

	"github.com/capillariesio/capillaries/pkg/env"
	"github.com/capillariesio/capillaries/pkg/l"
	"github.com/capillariesio/capillaries/pkg/wfmodel"
	amqp "github.com/rabbitmq/amqp091-go"
)

const DlxSuffix string = "_dlx"

type DaemonCmdType int8

const (
	DaemonCmdNone                DaemonCmdType = 0 // Should never see this
	DaemonCmdAckSuccess          DaemonCmdType = 2 // Best case
	DaemonCmdRejectAndRetryLater DaemonCmdType = 3 // Node dependencies are not ready, wait with proessing this node
	DaemonCmdReconnectDb         DaemonCmdType = 4 // Db workflow error, try to reconnect
	DaemonCmdQuit                DaemonCmdType = 5 // Shutdown command was received
	DaemonCmdAckWithError        DaemonCmdType = 6 // There was a processing error: either some serious biz logic re-trying will not help, or it was a data table error (we consider it persistent), so ack it
	DaemonCmdReconnectQueue      DaemonCmdType = 7 // Queue error, try to reconnect
)

func (daemonCmd DaemonCmdType) ToString() string {
	switch daemonCmd {
	case DaemonCmdNone:
		return "none"
	case DaemonCmdAckSuccess:
		return "sucess"
	case DaemonCmdRejectAndRetryLater:
		return "reject_and_retry"
	case DaemonCmdReconnectDb:
		return "reconnect_db"
	case DaemonCmdQuit:
		return "quit"
	case DaemonCmdAckWithError:
		return "ack_with_error"
	case DaemonCmdReconnectQueue:
		return "reconnect_queue"
	default:
		return "unknown"
	}
}

/*
amqpDeliveryToString - helper to print the contents of amqp.Delivery object
*/
func amqpDeliveryToString(d amqp.Delivery) string {
	// Do not just do Sprintf("%v", m), it will print the whole Body and it can be very long
	return fmt.Sprintf("Headers:%v, ContentType:%v, ContentEncoding:%v, DeliveryMode:%v, Priority:%v, CorrelationId:%v, ReplyTo:%v, Expiration:%v, MessageId:%v, Timestamp:%v, Type:%v, UserId:%v, AppId:%v, ConsumerTag:%v, MessageCount:%v, DeliveryTag:%v, Redelivered:%v, Exchange:%v, RoutingKey:%v, len(Body):%d",
		d.Headers,
		d.ContentType,
		d.ContentEncoding,
		d.DeliveryMode,
		d.Priority,
		d.CorrelationId,
		d.ReplyTo,
		d.Expiration,
		d.MessageId,
		d.Timestamp,
		d.Type,
		d.UserId,
		d.AppId,
		d.ConsumerTag,
		d.MessageCount,
		d.DeliveryTag,
		d.Redelivered,
		d.Exchange,
		d.RoutingKey,
		len(d.Body))
}

func processDelivery(envConfig *env.EnvConfig, logger *l.Logger, delivery *amqp.Delivery) DaemonCmdType {
	logger.PushF("wf.processDelivery")
	defer logger.PopF()

	// Deserialize incoming message
	var msgIn wfmodel.Message
	errDeserialize := msgIn.Deserialize(delivery.Body)
	if errDeserialize != nil {
		logger.Error("cannot deserialize incoming message: %s. %v", errDeserialize.Error(), delivery.Body)
		return DaemonCmdAckWithError
	}

	switch msgIn.MessageType {
	case wfmodel.MessageTypeDataBatch:
		dataBatchInfo, ok := msgIn.Payload.(wfmodel.MessagePayloadDataBatch)
		if !ok {
			logger.Error("unexpected type of data batch payload: %T", msgIn.Payload)
			return DaemonCmdAckWithError
		}
		return ProcessDataBatchMsg(envConfig, logger, msgIn.Ts, &dataBatchInfo)

	// TODO: other commands like debug level or shutdown go here
	default:
		logger.Error("unexpected message type %d", msgIn.MessageType)
		return DaemonCmdAckWithError
	}
}

func AmqpFullReconnectCycle(envConfig *env.EnvConfig, logger *l.Logger, osSignalChannel chan os.Signal) DaemonCmdType {
	logger.PushF("wf.AmqpFullReconnectCycle")
	defer logger.PopF()

	amqpConnection, err := amqp.Dial(envConfig.Amqp.URL)
	if err != nil {
		logger.Error("cannot dial RabbitMQ at %s, will reconnect: %s", envConfig.Amqp.URL, err.Error())
		return DaemonCmdReconnectQueue
	}

	// Subscribe to errors, this is how we handle queue failures
	chanErrors := amqpConnection.NotifyClose(make(chan *amqp.Error))
	var daemonCmd DaemonCmdType

	amqpChannel, err := amqpConnection.Channel()
	if err != nil {
		logger.Error("cannot create amqp channel, will reconnect: %s", err.Error())
		daemonCmd = DaemonCmdReconnectQueue
	} else {
		daemonCmd = amqpConnectAndSelect(envConfig, logger, osSignalChannel, amqpChannel, chanErrors)
		time.Sleep(1000)
		logger.Info("consuming %d amqp errors to avoid close deadlock...", len(chanErrors))
		for len(chanErrors) > 0 {
			chanErr := <-chanErrors
			logger.Info("consuming amqp error to avoid close deadlock: %v", chanErr)
		}
		logger.Info("consumed amqp errors, closing channel")
		amqpChannel.Close()
		logger.Info("consumed amqp errors, closed channel")
	}
	logger.Info("closing connection")
	amqpConnection.Close()
	logger.Info("closed connection")
	return daemonCmd
}

func amqpConnectAndSelect(envConfig *env.EnvConfig, logger *l.Logger, osSignalChannel chan os.Signal, amqpChannel *amqp.Channel, chanAmqpErrors chan *amqp.Error) DaemonCmdType {
	logger.PushF("wf.amqpConnectAndSelect")
	defer logger.PopF()

	errExchange := amqpChannel.ExchangeDeclare(
		envConfig.Amqp.Exchange, // exchange name
		"direct",                // type, "direct"
		false,                   // durable
		false,                   // auto-deleted
		false,                   // internal
		false,                   // no-wait
		nil)                     // arguments
	if errExchange != nil {
		logger.Error("cannot declare exchange %s, will reconnect: %s", envConfig.Amqp.Exchange, errExchange.Error())
		return DaemonCmdReconnectQueue
	}

	errExchange = amqpChannel.ExchangeDeclare(
		envConfig.Amqp.Exchange+DlxSuffix, // exchange name
		"direct",                          // type
		false,                             // durable
		false,                             // auto-deleted
		false,                             // internal
		false,                             // no-wait
		nil)                               // arguments
	if errExchange != nil {
		logger.Error("cannot declare exchange %s, will reconnect: %s", envConfig.Amqp.Exchange+DlxSuffix, errExchange.Error())
		return DaemonCmdReconnectQueue
	}

	// TODO: declare exchange for non-data signals and handle them in a separate queue

	amqpQueue, err := amqpChannel.QueueDeclare(
		envConfig.HandlerExecutableType, // queue name, matches routing key
		false,                           // durable
		false,                           // delete when unused
		false,                           // exclusive
		false,                           // no-wait
		amqp.Table{"x-dead-letter-exchange": envConfig.Amqp.Exchange + DlxSuffix, "x-dead-letter-routing-key": envConfig.HandlerExecutableType + DlxSuffix}) // arguments
	if err != nil {
		logger.Error("cannot declare queue %s, will reconnect: %s\n", envConfig.HandlerExecutableType, err.Error())
		return DaemonCmdReconnectQueue
	}

	amqpQueueDlx, err := amqpChannel.QueueDeclare(
		envConfig.HandlerExecutableType+DlxSuffix, // queue name, matches routing key
		false, // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		amqp.Table{"x-dead-letter-exchange": envConfig.Amqp.Exchange, "x-dead-letter-routing-key": envConfig.HandlerExecutableType, "x-message-ttl": envConfig.DeadLetterTtl})
	if err != nil {
		logger.Error("cannot declare queue %s, will reconnect: %s\n", envConfig.HandlerExecutableType+DlxSuffix, err.Error())
		return DaemonCmdReconnectQueue
	}

	errBind := amqpChannel.QueueBind(
		amqpQueue.Name,                  // queue name
		envConfig.HandlerExecutableType, // routing key / handler exe type
		envConfig.Amqp.Exchange,         // exchange
		false,                           // nowait
		nil)                             // args
	if errBind != nil {
		logger.Error("cannot bind queue %s with routing key %s, exchange %s , will reconnect: %s", amqpQueue.Name, envConfig.HandlerExecutableType, envConfig.Amqp.Exchange, errBind.Error())
		return DaemonCmdReconnectQueue
	}

	errBind = amqpChannel.QueueBind(
		amqpQueueDlx.Name, // queue name
		envConfig.HandlerExecutableType+DlxSuffix, // routing key / handler exe type
		envConfig.Amqp.Exchange+DlxSuffix,         // exchange
		false,                                     // nowait
		nil)                                       // args
	if errBind != nil {
		logger.Error("cannot bind queue %s with routing key %s, exchange %s , will reconnect: %s", amqpQueueDlx.Name, envConfig.HandlerExecutableType+DlxSuffix, envConfig.Amqp.Exchange+DlxSuffix, errBind.Error())
		return DaemonCmdReconnectQueue
	}

	errQos := amqpChannel.Qos(envConfig.Amqp.PrefetchCount, envConfig.Amqp.PrefetchSize, false)
	if errQos != nil {
		logger.Error("cannot set Qos, will reconnect: %s", errQos.Error())
		return DaemonCmdReconnectQueue
	}

	ampqChannelConsumerTag := logger.ZapMachine.String + "/consumer"
	chanDeliveries, err := amqpChannel.Consume(
		amqpQueue.Name,         // queue
		ampqChannelConsumerTag, // unique consumer tag, default go ampq implementation is os.argv[0] (is it really unique?)
		false,                  // auto-ack
		false,                  // exclusive
		false,                  // no-local - flag not supportd by rabbit
		false,                  // no-wait
		nil)                    // args
	if err != nil {
		logger.Error("cannot register consumer, queue %s, will reconnect: %s", amqpQueue.Name, err.Error())
		return DaemonCmdReconnectQueue
	}
	logger.Info("started consuming queue %s, routing key %s, exchange %s", amqpQueue.Name, envConfig.HandlerExecutableType, envConfig.Amqp.Exchange)

	var sem = make(chan int, envConfig.ThreadPoolSize)

	// daemonCommands len should be > ThreadPoolSize, otherwise on reconnect, we will get a deadlock:
	// "still waiting for all workers to complete" will wait for one or more workers that will try adding
	// "daemonCommands <- DaemonCmdReconnectDb" to the channel. Play safe by multiplying by 2.
	var daemonCommands = make(chan DaemonCmdType, envConfig.ThreadPoolSize*2)

	for {
		select {
		case osSignal := <-osSignalChannel:
			if osSignal == os.Interrupt || osSignal == os.Kill {
				logger.Info("received os signal %v, sending quit...", osSignal)
				daemonCommands <- DaemonCmdQuit
			}

		case chanErr := <-chanAmqpErrors:
			if chanErr != nil {
				logger.Error("detected closed amqp channel, will reconnect: %s", chanErr.Error())
			} else {
				logger.Error("detected closed amqp channel, will reconnect: nil error received")
			}
			daemonCommands <- DaemonCmdReconnectQueue

		case finalDaemonCmd := <-daemonCommands:

			// Here, we expect DaemonCmdReconnectDb, DaemonCmdReconnectQueue, DaemonCmdQuit. All of them require channel.Cancel()

			logger.Info("detected daemon cmd %d(%s), cancelling channel...", finalDaemonCmd, finalDaemonCmd.ToString())
			if err := amqpChannel.Cancel(ampqChannelConsumerTag, false); err != nil {
				logger.Error("cannot cancel amqp channel: %s", err.Error())
			} else {
				logger.Info("channel cancelled successfully")
			}

			logger.Info("handling daemon cmd %d(%s), waiting for all workers to complete (%d items)...", finalDaemonCmd, finalDaemonCmd.ToString(), len(sem))
			for len(sem) > 0 {
				logger.Info("still waiting for all workers to complete (%d items left)...", len(sem))
				time.Sleep(1000 * time.Millisecond)
			}

			logger.Info("handling daemon cmd %d(%s), all workers completed, draining cmd channel (%d items)...", finalDaemonCmd, finalDaemonCmd.ToString(), len(daemonCommands))
			for len(daemonCommands) > 0 {
				daemonCmd := <-daemonCommands
				// Do not ignore quit command, make sure it makes it to the finals
				if daemonCmd == DaemonCmdQuit {
					finalDaemonCmd = DaemonCmdQuit
				}
			}
			logger.Info("final daemon cmd %d(%s), all workers complete, cmd channel drained", finalDaemonCmd, finalDaemonCmd.ToString())
			return finalDaemonCmd

		case amqpDelivery := <-chanDeliveries:

			threadLogger, err := l.NewLoggerFromLogger(logger)
			if err != nil {
				logger.Error("cannot create logger for delivery handler thread: %s", err.Error())
				return DaemonCmdQuit
			}

			logger.PushF("wf.amqpConnectAndSelect_worker")
			defer logger.PopF()

			// Lock one slot in the semaphore
			sem <- 1

			go func(threadLogger *l.Logger, delivery amqp.Delivery, _channel *amqp.Channel) {
				var err error

				// I have spotted cases when m.Body is empty and Aknowledger is nil. Handle them.
				if delivery.Acknowledger == nil {
					threadLogger.Error("processor detected empty Acknowledger, assuming closed amqp channel, will reconnect: %s", amqpDeliveryToString(delivery))
					daemonCommands <- DaemonCmdReconnectQueue
				} else {
					daemonCmd := processDelivery(envConfig, threadLogger, &delivery)

					if daemonCmd == DaemonCmdAckSuccess || daemonCmd == DaemonCmdAckWithError {
						err = delivery.Ack(false)
						if err != nil {
							threadLogger.Error("failed to ack message, will reconnect: %s", err.Error())
							daemonCommands <- DaemonCmdReconnectQueue
						}
					} else if daemonCmd == DaemonCmdRejectAndRetryLater {
						err = delivery.Reject(false)
						if err != nil {
							threadLogger.Error("failed to reject message, will reconnect: %s", err.Error())
							daemonCommands <- DaemonCmdReconnectQueue
						}
					} else if daemonCmd == DaemonCmdReconnectQueue || daemonCmd == DaemonCmdReconnectDb {
						// // Ideally, RabbitMQ should be smart enough to re-deliver a msg that was neither acked nor rejected.
						// // But apparently, sometimes (when the machine goes to sleep, for example) the msg is never re-delivered. To improve our chances, force re-delivery by rejecting the msg.
						// threadLogger.Error("daemonCmd %s detected, will reject(requeue) and reconnect", daemonCmd.ToString())
						// err = delivery.Reject(true)
						// if err != nil {
						// 	threadLogger.Error("failed to reject message, will reconnect: %s", err.Error())
						// 	daemonCommands <- DaemonCmdReconnectQueue
						// } else {
						// 	daemonCommands <- daemonCmd
						// }

						// Verdict: we do not handle machine sleep scenario, amqp091-go goes into deadlock when shutting down.
						daemonCommands <- daemonCmd
					} else if daemonCmd == DaemonCmdQuit {
						daemonCommands <- DaemonCmdQuit
					} else {
						threadLogger.Error("unexpected daemon cmd: %d", daemonCmd)
					}
				}

				// Unlock semaphore slot
				<-sem

			}(threadLogger, amqpDelivery, amqpChannel)
		}
	}
}
