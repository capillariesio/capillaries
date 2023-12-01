package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/capillariesio/capillaries/pkg/custom/py_calc"
	"github.com/capillariesio/capillaries/pkg/custom/tag_and_denormalize"
	"github.com/capillariesio/capillaries/pkg/env"
	"github.com/capillariesio/capillaries/pkg/l"
	"github.com/capillariesio/capillaries/pkg/sc"
	"github.com/capillariesio/capillaries/pkg/wf"
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

func main() {

	envConfig, err := env.ReadEnvConfigFile("capidaemon.json")
	if err != nil {
		log.Fatalf(err.Error())
		os.Exit(1)
	}
	envConfig.CustomProcessorDefFactoryInstance = &StandardDaemonProcessorDefFactory{}

	logger, err := l.NewLoggerFromEnvConfig(envConfig)
	if err != nil {
		log.Fatalf(err.Error())
		os.Exit(1)
	}
	defer logger.Close()

	logger.PushF("daemon.main")
	defer logger.PopF()

	osSignalChannel := make(chan os.Signal, 1)
	signal.Notify(osSignalChannel, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

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
			// Break from select
			break
		}
	}
}
