package wf

// import (
// 	"fmt"
// 	"os"
// 	"sync/atomic"
// 	"time"

// 	"github.com/capillariesio/capillaries/pkg/env"
// 	"github.com/capillariesio/capillaries/pkg/l"
// 	"github.com/capillariesio/capillaries/pkg/wfmodel"
// 	amqp "github.com/rabbitmq/amqp091-go"
// )

// const DlxSuffix string = "_dlx"

// type DaemonCmdType int8

// const (
// 	DaemonCmdReconnectDb    DaemonCmdType = 4 // Db workflow error, try to reconnect
// 	DaemonCmdQuit           DaemonCmdType = 5 // Shutdown command was received
// 	DaemonCmdReconnectQueue DaemonCmdType = 7 // Queue error, try to reconnect
// )

// func (daemonCmd DaemonCmdType) ToString() string {
// 	switch daemonCmd {
// 	case DaemonCmdReconnectDb:
// 		return "cmd_reconnect_db"
// 	case DaemonCmdQuit:
// 		return "cmd_quit"
// 	case DaemonCmdReconnectQueue:
// 		return "cmd_reconnect_queue"
// 	default:
// 		return "cmd_unknown"
// 	}
// }

// /*
// amqpDeliveryToString - helper to print the contents of amqp.Delivery object
// */
// func amqpDeliveryToString(d amqp.Delivery) string {
// 	// Do not just do Sprintf("%v", m), it will print the whole Body and it can be very long
// 	return fmt.Sprintf("Headers:%v, ContentType:%v, ContentEncoding:%v, DeliveryMode:%v, Priority:%v, CorrelationId:%v, ReplyTo:%v, Expiration:%v, MessageId:%v, Timestamp:%v, Type:%v, UserId:%v, AppId:%v, ConsumerTag:%v, MessageCount:%v, DeliveryTag:%v, Redelivered:%v, Exchange:%v, RoutingKey:%v, len(Body):%d",
// 		d.Headers,
// 		d.ContentType,
// 		d.ContentEncoding,
// 		d.DeliveryMode,
// 		d.Priority,
// 		d.CorrelationId,
// 		d.ReplyTo,
// 		d.Expiration,
// 		d.MessageId,
// 		d.Timestamp,
// 		d.Type,
// 		d.UserId,
// 		d.AppId,
// 		d.ConsumerTag,
// 		d.MessageCount,
// 		d.DeliveryTag,
// 		d.Redelivered,
// 		d.Exchange,
// 		d.RoutingKey,
// 		len(d.Body))
// }

// func processDelivery(envConfig *env.EnvConfig, logger *l.CapiLogger, delivery *amqp.Delivery) ProcessDeliveryResultType {
// 	logger.PushF("wf.processDelivery")
// 	defer logger.PopF()

// 	// Deserialize incoming message
// 	var msgIn wfmodel.Message
// 	errDeserialize := msgIn.Deserialize(delivery.Body)
// 	if errDeserialize != nil {
// 		logger.Error("cannot deserialize incoming message: %s. %v", errDeserialize.Error(), delivery.Body)
// 		return ProcessDeliveryAckWithError
// 	}

// 	return ProcessDataBatchMsg(envConfig, logger, &msgIn)
// }

// func AmqpFullReconnectCycle(envConfig *env.EnvConfig, logger *l.CapiLogger, osSignalChannel chan os.Signal) DaemonCmdType {
// 	logger.PushF("wf.AmqpFullReconnectCycle")
// 	defer logger.PopF()

// 	amqpConnection, err := amqp.Dial(envConfig.Amqp.URL)
// 	if err != nil {
// 		logger.Error("cannot dial RabbitMQ at %s, will reconnect: %s", envConfig.Amqp.URL, err.Error())
// 		return DaemonCmdReconnectQueue
// 	}

// 	// Subscribe to errors, this is how we handle queue failures
// 	chanErrors := amqpConnection.NotifyClose(make(chan *amqp.Error))
// 	var daemonCmd DaemonCmdType

// 	amqpChannel, err := amqpConnection.Channel()
// 	if err != nil {
// 		logger.Error("cannot create amqp channel, will reconnect: %s", err.Error())
// 		daemonCmd = DaemonCmdReconnectQueue
// 	} else {
// 		daemonCmd = amqpConnectAndSelect(envConfig, logger, osSignalChannel, amqpChannel, chanErrors)
// 		time.Sleep(1000 * time.Millisecond)
// 		logger.Info("consuming %d amqp errors to avoid close deadlock...", len(chanErrors))
// 		for len(chanErrors) > 0 {
// 			chanErr := <-chanErrors
// 			logger.Info("consuming amqp error to avoid close deadlock: %v", chanErr)
// 		}
// 		logger.Info("consumed amqp errors, closing channel")
// 		amqpChannel.Close() // TODO: this hangs sometimes
// 		logger.Info("consumed amqp errors, closed channel")
// 	}
// 	logger.Info("closing connection")
// 	amqpConnection.Close()
// 	logger.Info("closed connection")
// 	return daemonCmd
// }

// func initAmqpDeliveryChannel(envConfig *env.EnvConfig, logger *l.CapiLogger, amqpChannel *amqp.Channel, ampqChannelConsumerTag string) (<-chan amqp.Delivery, error) {
// 	errExchange := amqpChannel.ExchangeDeclare(
// 		envConfig.Amqp.Exchange, // exchange name
// 		"direct",                // type, "direct"
// 		false,                   // durable
// 		false,                   // auto-deleted
// 		false,                   // internal
// 		false,                   // no-wait
// 		nil)                     // arguments
// 	if errExchange != nil {
// 		logger.Error("cannot declare exchange %s, will reconnect: %s", envConfig.Amqp.Exchange, errExchange.Error())
// 		return nil, errExchange
// 	}

// 	errExchange = amqpChannel.ExchangeDeclare(
// 		envConfig.Amqp.Exchange+DlxSuffix, // exchange name
// 		"direct",                          // type
// 		false,                             // durable
// 		false,                             // auto-deleted
// 		false,                             // internal
// 		false,                             // no-wait
// 		nil)                               // arguments
// 	if errExchange != nil {
// 		logger.Error("cannot declare exchange %s, will reconnect: %s", envConfig.Amqp.Exchange+DlxSuffix, errExchange.Error())
// 		return nil, errExchange
// 	}

// 	// TODO: declare exchange for non-data signals and handle them in a separate queue

// 	amqpQueue, err := amqpChannel.QueueDeclare(
// 		envConfig.HandlerExecutableType, // queue name, matches routing key
// 		false,                           // durable
// 		false,                           // delete when unused
// 		false,                           // exclusive
// 		false,                           // no-wait
// 		amqp.Table{"x-dead-letter-exchange": envConfig.Amqp.Exchange + DlxSuffix, "x-dead-letter-routing-key": envConfig.HandlerExecutableType + DlxSuffix}) // arguments
// 	if err != nil {
// 		logger.Error("cannot declare queue %s, will reconnect: %s\n", envConfig.HandlerExecutableType, err.Error())
// 		return nil, err
// 	}

// 	amqpQueueDlx, err := amqpChannel.QueueDeclare(
// 		envConfig.HandlerExecutableType+DlxSuffix, // queue name, matches routing key
// 		false, // durable
// 		false, // delete when unused
// 		false, // exclusive
// 		false, // no-wait
// 		amqp.Table{"x-dead-letter-exchange": envConfig.Amqp.Exchange, "x-dead-letter-routing-key": envConfig.HandlerExecutableType, "x-message-ttl": envConfig.Daemon.DeadLetterTtl})
// 	if err != nil {
// 		logger.Error("cannot declare queue %s, will reconnect: %s\n", envConfig.HandlerExecutableType+DlxSuffix, err.Error())
// 		return nil, err
// 	}

// 	errBind := amqpChannel.QueueBind(
// 		amqpQueue.Name,                  // queue name
// 		envConfig.HandlerExecutableType, // routing key / handler exe type
// 		envConfig.Amqp.Exchange,         // exchange
// 		false,                           // nowait
// 		nil)                             // args
// 	if errBind != nil {
// 		logger.Error("cannot bind queue %s with routing key %s, exchange %s , will reconnect: %s", amqpQueue.Name, envConfig.HandlerExecutableType, envConfig.Amqp.Exchange, errBind.Error())
// 		return nil, errBind
// 	}

// 	errBind = amqpChannel.QueueBind(
// 		amqpQueueDlx.Name, // queue name
// 		envConfig.HandlerExecutableType+DlxSuffix, // routing key / handler exe type
// 		envConfig.Amqp.Exchange+DlxSuffix,         // exchange
// 		false,                                     // nowait
// 		nil)                                       // args
// 	if errBind != nil {
// 		logger.Error("cannot bind queue %s with routing key %s, exchange %s , will reconnect: %s", amqpQueueDlx.Name, envConfig.HandlerExecutableType+DlxSuffix, envConfig.Amqp.Exchange+DlxSuffix, errBind.Error())
// 		return nil, errBind
// 	}

// 	errQos := amqpChannel.Qos(envConfig.Amqp.PrefetchCount, envConfig.Amqp.PrefetchSize, false)
// 	if errQos != nil {
// 		logger.Error("cannot set Qos, will reconnect: %s", errQos.Error())
// 		return nil, errQos
// 	}

// 	chanDeliveries, err := amqpChannel.Consume(
// 		amqpQueue.Name,         // queue
// 		ampqChannelConsumerTag, // unique consumer tag, default go ampq implementation is os.argv[0] (is it really unique?)
// 		false,                  // auto-ack
// 		false,                  // exclusive
// 		false,                  // no-local - flag not supportd by rabbit
// 		false,                  // no-wait
// 		nil)                    // args
// 	if err != nil {
// 		logger.Error("cannot register consumer, queue %s, will reconnect: %s", amqpQueue.Name, err.Error())
// 		return nil, err
// 	}

// 	return chanDeliveries, nil
// }

// func writeSingletonDaemonCmd(daemonCommands chan DaemonCmdType, writeCount *int64, cmd DaemonCmdType) {
// 	if atomic.AddInt64(writeCount, 1) == 1 {
// 		daemonCommands <- cmd
// 	}
// }

// func amqpConnectAndSelect(envConfig *env.EnvConfig, logger *l.CapiLogger, osSignalChannel chan os.Signal, amqpChannel *amqp.Channel, chanAmqpErrors chan *amqp.Error) DaemonCmdType {
// 	logger.PushF("wf.amqpConnectAndSelect")
// 	defer logger.PopF()

// 	ampqChannelConsumerTag := logger.ZapMachine.String + "/consumer"

// 	chanDeliveries, initErr := initAmqpDeliveryChannel(envConfig, logger, amqpChannel, ampqChannelConsumerTag)
// 	if initErr != nil {
// 		return DaemonCmdReconnectQueue
// 	}

// 	logger.Info("started consuming queue %s, routing key %s, exchange %s", envConfig.HandlerExecutableType, envConfig.HandlerExecutableType, envConfig.Amqp.Exchange)

// 	var sem = make(chan int, envConfig.Daemon.ThreadPoolSize)
// 	var daemonCommands = make(chan DaemonCmdType, 1)
// 	daemonCommandCount := int64(0)
// 	for {
// 		select {
// 		case osSignal := <-osSignalChannel:
// 			if osSignal == os.Interrupt || osSignal == os.Kill {
// 				logger.Info("received os signal %v, sending quit...", osSignal)
// 				writeSingletonDaemonCmd(daemonCommands, &daemonCommandCount, DaemonCmdQuit)
// 			}

// 		case chanErr := <-chanAmqpErrors:
// 			if chanErr != nil {
// 				logger.Error("detected closed amqp channel, will reconnect: %s", chanErr.Error())
// 			} else {
// 				logger.Error("detected closed amqp channel, will reconnect: nil error received")
// 			}
// 			writeSingletonDaemonCmd(daemonCommands, &daemonCommandCount, DaemonCmdReconnectQueue)

// 		case finalDaemonCmd := <-daemonCommands:

// 			// Here, we expect DaemonCmdReconnectDb, DaemonCmdReconnectQueue, DaemonCmdQuit. All of them require channel.Cancel()

// 			logger.Info("detected daemon cmd %d(%s), cancelling channel...", finalDaemonCmd, finalDaemonCmd.ToString())
// 			if err := amqpChannel.Cancel(ampqChannelConsumerTag, false); err != nil {
// 				logger.Error("cannot cancel amqp channel: %s", err.Error())
// 			} else {
// 				logger.Info("channel cancelled successfully")
// 			}

// 			logger.Info("handling daemon cmd %d(%s), waiting for all workers to complete (%d items)...", finalDaemonCmd, finalDaemonCmd.ToString(), len(sem))
// 			for len(sem) > 0 {
// 				logger.Info("handling daemon cmd %d(%s), still waiting for all workers to complete (%d items left)...", finalDaemonCmd, finalDaemonCmd.ToString(), len(sem))
// 				time.Sleep(1000 * time.Millisecond)
// 			}

// 			logger.Info("final daemon cmd %d(%s), all workers complete", finalDaemonCmd, finalDaemonCmd.ToString())
// 			return finalDaemonCmd

// 		case amqpDelivery := <-chanDeliveries:

// 			threadLogger, err := l.NewLoggerFromLogger(logger)
// 			if err != nil {
// 				logger.Error("cannot create logger for delivery handler thread: %s", err.Error())
// 				return DaemonCmdQuit
// 			}

// 			// Lock one slot in the semaphore
// 			sem <- 1

// 			// envConfig.ThreadPoolSize goroutines run simultaneously
// 			go func(threadLogger *l.CapiLogger, delivery amqp.Delivery, _ *amqp.Channel) {
// 				var err error

// 				// I have spotted cases when m.Body is empty and Aknowledger is nil. Handle them.
// 				if delivery.Acknowledger == nil {
// 					threadLogger.Error("processor detected empty Acknowledger, assuming closed amqp channel, will reconnect: %s", amqpDeliveryToString(delivery))
// 					writeSingletonDaemonCmd(daemonCommands, &daemonCommandCount, DaemonCmdReconnectQueue)
// 				} else {
// 					// The main processing call
// 					processDeliveryResult := processDelivery(envConfig, threadLogger, &delivery)
// 					switch processDeliveryResult {
// 					case ProcessDeliveryAckSuccess, ProcessDeliveryAckWithError:
// 						err = delivery.Ack(false)
// 						if err != nil {
// 							threadLogger.Error("failed to ack message, will reconnect, sending daemonCommands <- DaemonCmdReconnectQueue: %s", err.Error())
// 							writeSingletonDaemonCmd(daemonCommands, &daemonCommandCount, DaemonCmdReconnectQueue)
// 						}
// 					case ProcessDeliveryRejectAndRetryLater:
// 						err = delivery.Reject(false)
// 						if err != nil {
// 							threadLogger.Error("failed to reject message, will reconnect: %s", err.Error())
// 							writeSingletonDaemonCmd(daemonCommands, &daemonCommandCount, DaemonCmdReconnectQueue)
// 						}
// 					case ProcessDeliveryReconnectQueueNotUsed:
// 						// Ideally, RabbitMQ should be smart enough to re-deliver a msg that was neither acked nor rejected.
// 						// But apparently, sometimes (when the machine goes to sleep, for example) the msg is never re-delivered. To improve our chances, force re-delivery by rejecting the msg.
// 						// threadLogger.Error("ProcessDeliveryReconnectQueue detected, will reject(requeue) and reconnect")
// 						// err = delivery.Reject(true)
// 						// if err != nil {
// 						// 	threadLogger.Error("failed to reject message, will reconnect: %s", err.Error())
// 						// 	daemonCommands <- DaemonCmdReconnectQueue
// 						// } else {
// 						// 	daemonCommands <- whatever the result was
// 						// }
// 						// Verdict: we do not handle machine sleep scenario, amqp091-go goes into deadlock when shutting down the box as of 2022
// 						writeSingletonDaemonCmd(daemonCommands, &daemonCommandCount, DaemonCmdReconnectQueue)
// 					case ProcessDeliveryReconnectDb:
// 						writeSingletonDaemonCmd(daemonCommands, &daemonCommandCount, DaemonCmdReconnectDb)
// 					default:
// 						threadLogger.Error("unexpected process delivery result: %d", processDeliveryResult)
// 					}
// 				}

// 				// Unlock semaphore slot
// 				<-sem

// 			}(threadLogger, amqpDelivery, amqpChannel)
// 		}
// 	}
// }
