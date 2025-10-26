package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/capillariesio/capillaries/pkg/custom/py_calc"
	"github.com/capillariesio/capillaries/pkg/custom/tag_and_denormalize"
	"github.com/capillariesio/capillaries/pkg/env"
	"github.com/capillariesio/capillaries/pkg/l"
	"github.com/capillariesio/capillaries/pkg/mq"
	"github.com/capillariesio/capillaries/pkg/sc"
	"github.com/capillariesio/capillaries/pkg/wf"
	"github.com/capillariesio/capillaries/pkg/wfmodel"
	"github.com/capillariesio/capillaries/pkg/xfer"
	"github.com/hashicorp/golang-lru/v2/expirable"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// https://stackoverflow.com/questions/25927660/how-to-get-the-current-function-name
// func trc() string {
// 	pc := make([]uintptr, 15)
// 	n := runtime.Callers(2, pc)
// 	frames := runtime.CallersFrames(pc[:n])
// 	frame, _ := frames.Next()
// 	return fmt.Sprintf("%s:%d %s\n", frame.File, frame.Line, frame.Function)
// }

type StandardDaemonProcessorDefFactory struct {
}

func (f *StandardDaemonProcessorDefFactory) Create(processorType string) (sc.CustomProcessorDef, bool) {
	// All processors to be supported by this 'stock' binary (daemon/toolbelt).
	// If you develop your own processor(s), use your own ProcessorDefFactory that lists all processors,
	// they all must implement CustomProcessorRunner interface
	switch processorType {
	case py_calc.ProcessorPyCalcName:
		return &py_calc.PyCalcProcessorDef{}, true
	case tag_and_denormalize.ProcessorTagAndDenormalizeName:
		return &tag_and_denormalize.TagAndDenormalizeProcessorDef{}, true
	default:
		return nil, false
	}
}

var version string

func main() {
	// defer profile.Start(profile.MemProfile).Stop()
	// go func() {
	// 	http.ListenAndServe("localhost:8081", nil)
	// }()

	// curl http://localhost:8081/debug/pprof/heap > heap.01.pprof
	// aws s3 cp heap.01.pprof s3://capillaries-testbucket/log/

	initCtx, initCancel := context.WithTimeout(context.Background(), 1*time.Second)
	envConfig, err := env.ReadEnvConfigFile(initCtx, "capidaemon.json")
	initCancel()
	if err != nil {
		log.Fatalf("%s", err.Error())
	}
	envConfig.CustomProcessorDefFactoryInstance = &StandardDaemonProcessorDefFactory{}

	if envConfig.Log.PrometheusExporterPort > 0 {
		prometheus.MustRegister(xfer.SftpFileGetGetDuration, xfer.HttpFileGetGetDuration, xfer.S3FileGetGetDuration)
		prometheus.MustRegister(sc.ScriptDefCacheHitCounter, sc.ScriptDefCacheMissCounter)
		prometheus.MustRegister(wf.NodeDependencyReadynessHitCounter, wf.NodeDependencyReadynessMissCounter, wf.NodeDependencyReadynessGetDuration, wf.NodeDependencyNoneCounter, wf.NodeDependencyWaitCounter, wf.NodeDependencyGoCounter, wf.NodeDependencyNogoCounter, wf.ReceivedMsgCounter)
		go func() {
			http.Handle("/metrics", promhttp.Handler())
			if err := http.ListenAndServe(fmt.Sprintf(":%d", envConfig.Log.PrometheusExporterPort), nil); err != nil {
				log.Fatalf("%s", err.Error())
			}
		}()
	}

	logger, err := l.NewLoggerFromEnvConfig(envConfig)
	if err != nil {
		log.Fatalf("%s", err.Error())
	}
	defer logger.Close()

	logger.PushF("daemon.main")
	defer logger.PopF()

	logger.Info("Capillaries daemon %s", version)
	logger.Info("env config: %s", envConfig.String())
	logger.Info("S3 config status: %s", xfer.GetS3ConfigStatus(initCtx).String())

	osSignalChannel := make(chan os.Signal, 1)
	signal.Notify(osSignalChannel, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	sc.ScriptDefCache = expirable.NewLRU[string, sc.ScriptInitResult](50, nil, time.Minute*1)
	wf.NodeDependencyReadynessCache = expirable.NewLRU[string, string](1000, nil, time.Second*2)

	var heartbeatInterval int64
	var asyncConsumer mq.MqAsyncConsumer
	if envConfig.Amqp10.URL != "" && envConfig.Amqp10.Address != "" {
		asyncConsumer = mq.NewAmqp10Consumer(envConfig.Amqp10.URL, envConfig.Amqp10.Address, int32(envConfig.Daemon.ThreadPoolSize))
	} else if envConfig.CapiMqDaemon.URL != "" {
		asyncConsumer = mq.NewCapimqConsumer(envConfig.CapiMqDaemon.URL)
		heartbeatInterval = envConfig.CapiMqDaemon.HeartbeatInterval
	} else {
		log.Fatalf("%s", "no mq broker configured")
	}

	listenerChannel := make(chan *wfmodel.Message, envConfig.Daemon.ThreadPoolSize)
	acknowledgerChannel := make(chan mq.AknowledgerToken, max(1, envConfig.Daemon.ThreadPoolSize/2)) // empirical
	var sem = make(chan int, envConfig.Daemon.ThreadPoolSize)

	if err := asyncConsumer.Start(logger, listenerChannel, acknowledgerChannel); err != nil {
		log.Fatalf("%s", err.Error())
	}

	for {
		select {
		case osSignal := <-osSignalChannel:
			if osSignal == os.Interrupt || osSignal == os.Kill {
				logger.Info("received os signal %v , quitting...", osSignal)
				if err = asyncConsumer.StopListener(logger); err != nil {
					logger.Error("cannot stop listener gracefully, brace for impact: %s", err.Error())
				}
				// This can make listenerWorker panic if there was an error above
				close(listenerChannel)

				logger.Info("started waiting for all workers to complete (%d items)", len(sem))
				for len(sem) > 0 {
					logger.Info("still waiting for all workers to complete (%d items left)...", len(sem))
					time.Sleep(1000 * time.Millisecond)
				}

				if err = asyncConsumer.StopAcknowledger(logger); err != nil {
					logger.Error("cannot stop acknowledger gracefully, brace for impact: %s", err.Error())
				}
				// This can make acknowledgerWorker panic if there was an error above
				close(acknowledgerChannel)
				os.Exit(0)
			}
		case wfmodelMsg := <-listenerChannel:
			threadLogger, err := l.NewLoggerFromLogger(logger)
			if err != nil {
				logger.Error("cannot create logger for delivery handler thread: %s", err.Error())
				log.Fatalf("%s", err.Error())
			}

			// Lock one slot in the semaphore
			sem <- 1

			// envConfig.ThreadPoolSize goroutines run simultaneously
			go func(threadLogger *l.CapiLogger, wfmodelMsg *wfmodel.Message, acknowledgerChannel chan mq.AknowledgerToken) {
				var heartbeatCallback func(wfmodelMsgId string)
				if asyncConsumer.SupportsHearbeat() {
					heartbeatCallback = func(wfmodelMsgId string) {
						acknowledgerChannel <- mq.AknowledgerToken{MsgId: wfmodelMsgId, Cmd: mq.AcknowledgerCmdHeartbeat}
					}
				}
				acknowledgerChannel <- mq.AknowledgerToken{
					MsgId: wfmodelMsg.Id,
					Cmd:   wf.ProcessDataBatchMsgNew(envConfig, logger, wfmodelMsg, heartbeatInterval, heartbeatCallback)}

				// Unlock semaphore slot
				<-sem

			}(threadLogger, wfmodelMsg, acknowledgerChannel)

		}

	}
}

/*
func main() {

	amqpConsumerReceiver := mq.Amqp10Consumer{}
	amqpConsumerAcknowledger := mq.Amqp10Consumer{}

	initCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	if err := amqpConsumerReceiver.Open(initCtx, "amqp://artemis:artemis@127.0.0.1:5672/", "capillaries"); err != nil {
		panic(err.Error())
	}
	cancel()

	initCtx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	if err := amqpConsumerAcknowledger.Open(initCtx, "amqp://artemis:artemis@127.0.0.1:5672/", "capillaries"); err != nil {
		panic(err.Error())
	}
	cancel()

	for {
		ctxReceive, cancelReceive := context.WithTimeout(context.Background(), 1*time.Second)
		msg, err := amqpConsumerReceiver.Receiver().Receive(ctxReceive, nil)
		cancelReceive()
		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				fmt.Printf("nothing to receive\n")
				continue
				//} else if errors.Is(err, amqp10.ConnError) || errors.Is(err, amqp10.SessionError) {
			} else {
				panic(err.Error())
			}
		}
		var m wfmodel.Message
		err = json.Unmarshal(msg.Data[0], &m)
		if err != nil {
			panic(err.Error())
		}
		fmt.Printf("%s\n", m.FullBatchId())
		// ctxAck, cancelAck := context.WithTimeout(context.Background(), 1*time.Second)
		// if m.BatchIdx > 100 {
		// 	err = amqpConsumerAcknowledger.Receiver().ReleaseMessage(ctxAck, msg)
		// } else {
		// 	err = amqpConsumerAcknowledger.Receiver().AcceptMessage(ctxAck, msg)
		// }
		// cancelAck()
		// if err == context.DeadlineExceeded {
		// 	fmt.Printf("ack timeout\n")
		// }
		// if err != nil {
		// 	panic(err.Error())
		// }
	}
}
*/

// func mainOld() {
// 	// defer profile.Start(profile.MemProfile).Stop()
// 	// go func() {
// 	// 	http.ListenAndServe("localhost:8081", nil)
// 	// }()

// 	// curl http://localhost:8081/debug/pprof/heap > heap.01.pprof
// 	// aws s3 cp heap.01.pprof s3://capillaries-testbucket/log/

// 	initCtx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
// 	defer cancel()

// 	envConfig, err := env.ReadEnvConfigFile(initCtx, "capidaemon.json")
// 	if err != nil {
// 		log.Fatalf("%s", err.Error())
// 	}
// 	envConfig.CustomProcessorDefFactoryInstance = &StandardDaemonProcessorDefFactory{}

// 	if envConfig.Log.PrometheusExporterPort > 0 {
// 		prometheus.MustRegister(xfer.SftpFileGetGetDuration, xfer.HttpFileGetGetDuration, xfer.S3FileGetGetDuration)
// 		prometheus.MustRegister(sc.ScriptDefCacheHitCounter, sc.ScriptDefCacheMissCounter)
// 		prometheus.MustRegister(wf.NodeDependencyReadynessHitCounter, wf.NodeDependencyReadynessMissCounter, wf.NodeDependencyReadynessGetDuration, wf.NodeDependencyNoneCounter, wf.NodeDependencyWaitCounter, wf.NodeDependencyGoCounter, wf.NodeDependencyNogoCounter, wf.ReceivedMsgCounter)
// 		go func() {
// 			http.Handle("/metrics", promhttp.Handler())
// 			if err := http.ListenAndServe(fmt.Sprintf(":%d", envConfig.Log.PrometheusExporterPort), nil); err != nil {
// 				log.Fatalf("%s", err.Error())
// 			}
// 		}()
// 	}

// 	logger, err := l.NewLoggerFromEnvConfig(envConfig)
// 	if err != nil {
// 		log.Fatalf("%s", err.Error())
// 	}
// 	defer logger.Close()

// 	logger.PushF("daemon.main")
// 	defer logger.PopF()

// 	logger.Info("Capillaries daemon %s", version)
// 	logger.Info("env config: %s", envConfig.String())
// 	logger.Info("S3 config status: %s", xfer.GetS3ConfigStatus(initCtx).String())

// 	osSignalChannel := make(chan os.Signal, 1)
// 	signal.Notify(osSignalChannel, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

// 	sc.ScriptDefCache = expirable.NewLRU[string, sc.ScriptInitResult](50, nil, time.Minute*1)
// 	wf.NodeDependencyReadynessCache = expirable.NewLRU[string, string](1000, nil, time.Second*2)

// 	for {
// 		daemonCmd := wf.AmqpFullReconnectCycle(envConfig, logger, osSignalChannel)
// 		if daemonCmd == wf.DaemonCmdQuit {
// 			logger.Info("got quit cmd, shut down is supposed to be complete by now")
// 			os.Exit(0)
// 		}
// 		logger.Info("got %d, waiting before reconnect...", daemonCmd)

// 		// Read from osSignalChannel with timeout
// 		timeoutChannel := make(chan bool, 1)
// 		go func() {
// 			time.Sleep(10 * time.Second)
// 			timeoutChannel <- true
// 		}()
// 		select {
// 		case osSignal := <-osSignalChannel:
// 			if osSignal == os.Interrupt || osSignal == os.Kill {
// 				logger.Info("received os signal %v while reconnecting to mq, quitting...", osSignal)
// 				os.Exit(0)
// 			}
// 		case <-timeoutChannel:
// 			logger.Info("timeout while reconnecting to mq, will try to reconnect again")
// 			continue
// 		}
// 	}
// }
