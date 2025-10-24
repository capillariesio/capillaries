package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hashicorp/golang-lru/v2/expirable"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/capillariesio/capillaries/pkg/custom/py_calc"
	"github.com/capillariesio/capillaries/pkg/custom/tag_and_denormalize"
	"github.com/capillariesio/capillaries/pkg/env"
	"github.com/capillariesio/capillaries/pkg/l"
	"github.com/capillariesio/capillaries/pkg/mq"
	"github.com/capillariesio/capillaries/pkg/sc"
	"github.com/capillariesio/capillaries/pkg/wf"
	"github.com/capillariesio/capillaries/pkg/wfmodel"
	"github.com/capillariesio/capillaries/pkg/xfer"
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
		msg, err := amqpConsumerReceiver.Receive(ctxReceive)
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

		// Do not ack or release, just ignore!
		// ctxAck, cancelAck := context.WithTimeout(context.Background(), 1*time.Second)
		// if m.BatchIdx == 0 {
		// 	err = amqpConsumerAcknowledger.ReleaseForRetry(ctxAck, msg)
		// } else {
		// 	err = amqpConsumerAcknowledger.Ack(ctxAck, msg)
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

func main1() {

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
		msg, err := amqpConsumerReceiver.Receive(ctxReceive)
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
		ctxAck, cancelAck := context.WithTimeout(context.Background(), 1*time.Second)
		if m.BatchIdx == 0 {
			err = amqpConsumerAcknowledger.ReleaseForRetry(ctxAck, msg)
		} else {
			err = amqpConsumerAcknowledger.Ack(ctxAck, msg)
		}
		cancelAck()
		if err == context.DeadlineExceeded {
			fmt.Printf("ack timeout\n")
		}
		if err != nil {
			panic(err.Error())
		}
	}
}

func mainOld() {
	// defer profile.Start(profile.MemProfile).Stop()
	// go func() {
	// 	http.ListenAndServe("localhost:8081", nil)
	// }()

	// curl http://localhost:8081/debug/pprof/heap > heap.01.pprof
	// aws s3 cp heap.01.pprof s3://capillaries-testbucket/log/

	initCtx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	envConfig, err := env.ReadEnvConfigFile(initCtx, "capidaemon.json")
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

	for {
		daemonCmd := wf.AmqpFullReconnectCycle(envConfig, logger, osSignalChannel)
		if daemonCmd == wf.DaemonCmdQuit {
			logger.Info("got quit cmd, shut down is supposed to be complete by now")
			os.Exit(0)
		}
		logger.Info("got %d, waiting before reconnect...", daemonCmd)

		// Read from osSignalChannel with timeout
		timeoutChannel := make(chan bool, 1)
		go func() {
			time.Sleep(10 * time.Second)
			timeoutChannel <- true
		}()
		select {
		case osSignal := <-osSignalChannel:
			if osSignal == os.Interrupt || osSignal == os.Kill {
				logger.Info("received os signal %v while reconnecting to mq, quitting...", osSignal)
				os.Exit(0)
			}
		case <-timeoutChannel:
			logger.Info("timeout while reconnecting to mq, will try to reconnect again")
			continue
		}
	}
}
