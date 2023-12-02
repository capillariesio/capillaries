package l

import (
	"fmt"
	"os"
	"sync/atomic"
	"time"

	"github.com/capillariesio/capillaries/pkg/ctx"
	"github.com/capillariesio/capillaries/pkg/env"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type CapiLogger struct {
	ZapLogger           *zap.Logger
	ZapMachine          zapcore.Field
	ZapThread           zapcore.Field
	SavedZapConfig      zap.Config
	AtomicThreadCounter *int64
	ZapFunction         zapcore.Field
	FunctionStack       []string
}

func (l *CapiLogger) PushF(functionName string) {
	l.ZapFunction = zap.String("f", functionName)
	l.FunctionStack = append(l.FunctionStack, functionName)
}
func (l *CapiLogger) PopF() {
	if len(l.FunctionStack) > 0 {
		l.FunctionStack = l.FunctionStack[:len(l.FunctionStack)-1]
		if len(l.FunctionStack) > 0 {
			l.ZapFunction = zap.String("f", l.FunctionStack[len(l.FunctionStack)-1])
		} else {
			l.ZapFunction = zap.String("f", "stack_underflow")
		}
	}
}

func NewLoggerFromEnvConfig(envConfig *env.EnvConfig) (*CapiLogger, error) {
	atomicTreadCounter := int64(0)
	l := CapiLogger{AtomicThreadCounter: &atomicTreadCounter}
	hostName, err := os.Hostname()
	if err != nil {
		return nil, fmt.Errorf("cannot get hostname: %s", err.Error())
	}
	l.ZapMachine = zap.String("i", fmt.Sprintf("%s/%s/%s", hostName, envConfig.HandlerExecutableType, time.Now().Format("01-02T15:04:05.000")))
	l.ZapThread = zap.Int64("t", 0)
	l.ZapFunction = zap.String("f", "")

	// TODO: this solution writes everything to stdout. Potentially, there is a way to write Debug/Info/Warn to stdout and
	// errors to std err: https://stackoverflow.com/questions/68472667/how-to-log-to-stdout-or-stderr-based-on-log-level-using-uber-go-zap
	// Do some research to see if this can be added to our ZapConfig.Build() scenario.
	l.SavedZapConfig = envConfig.ZapConfig
	l.ZapLogger, err = envConfig.ZapConfig.Build()
	if err != nil {
		return nil, fmt.Errorf("cannot build l from config: %s", err.Error())
	}
	return &l, nil
}

func NewLoggerFromLogger(srcLogger *CapiLogger) (*CapiLogger, error) {
	l := CapiLogger{
		SavedZapConfig:      srcLogger.SavedZapConfig,
		AtomicThreadCounter: srcLogger.AtomicThreadCounter,
		ZapMachine:          srcLogger.ZapMachine,
		ZapFunction:         zap.String("f", ""),
		ZapThread:           zap.Int64("t", atomic.AddInt64(srcLogger.AtomicThreadCounter, 1))}

	var err error
	l.ZapLogger, err = srcLogger.SavedZapConfig.Build()
	if err != nil {
		return nil, fmt.Errorf("cannot build l from l: %s", err.Error())
	}
	return &l, nil
}

func (l *CapiLogger) Close() {
	l.ZapLogger.Sync() //nolint:all
}

func (l *CapiLogger) Debug(format string, a ...any) {
	l.ZapLogger.Debug(fmt.Sprintf(format, a...), l.ZapMachine, l.ZapThread, l.ZapFunction)
}

func (l *CapiLogger) DebugCtx(pCtx *ctx.MessageProcessingContext, format string, a ...any) {
	if pCtx == nil {
		l.ZapLogger.Debug(fmt.Sprintf(format, a...), l.ZapMachine, l.ZapThread, l.ZapFunction)
	} else {
		l.ZapLogger.Debug(fmt.Sprintf(format, a...), l.ZapMachine, l.ZapThread, l.ZapFunction, pCtx.ZapDataKeyspace, pCtx.ZapRun, pCtx.ZapNode, pCtx.ZapBatchIdx, pCtx.ZapMsgAgeMillis)
	}
}

func (l *CapiLogger) Info(format string, a ...any) {
	l.ZapLogger.Info(fmt.Sprintf(format, a...), l.ZapMachine, l.ZapThread, l.ZapFunction)
}

func (l *CapiLogger) InfoCtx(pCtx *ctx.MessageProcessingContext, format string, a ...any) {
	if pCtx == nil {
		l.ZapLogger.Info(fmt.Sprintf(format, a...), l.ZapMachine, l.ZapThread, l.ZapFunction)
	} else {
		l.ZapLogger.Info(fmt.Sprintf(format, a...), l.ZapMachine, l.ZapThread, l.ZapFunction, pCtx.ZapDataKeyspace, pCtx.ZapRun, pCtx.ZapNode, pCtx.ZapBatchIdx, pCtx.ZapMsgAgeMillis)
	}
}

func (l *CapiLogger) Warn(format string, a ...any) {
	l.ZapLogger.Warn(fmt.Sprintf(format, a...), l.ZapMachine, l.ZapThread, l.ZapFunction)
}

func (l *CapiLogger) WarnCtx(pCtx *ctx.MessageProcessingContext, format string, a ...any) {
	if pCtx == nil {
		l.ZapLogger.Warn(fmt.Sprintf(format, a...), l.ZapMachine, l.ZapThread, l.ZapFunction)
	} else {
		l.ZapLogger.Warn(fmt.Sprintf(format, a...), l.ZapMachine, l.ZapThread, l.ZapFunction, pCtx.ZapDataKeyspace, pCtx.ZapRun, pCtx.ZapNode, pCtx.ZapBatchIdx, pCtx.ZapMsgAgeMillis)
	}
}

func (l *CapiLogger) Error(format string, a ...any) {
	l.ZapLogger.Error(fmt.Sprintf(format, a...), l.ZapMachine, l.ZapThread, l.ZapFunction)
}

func (l *CapiLogger) ErrorCtx(pCtx *ctx.MessageProcessingContext, format string, a ...any) {
	if pCtx == nil {
		l.ZapLogger.Error(fmt.Sprintf(format, a...), l.ZapMachine, l.ZapThread, l.ZapFunction)
	} else {
		l.ZapLogger.Error(fmt.Sprintf(format, a...), l.ZapMachine, l.ZapThread, l.ZapFunction, pCtx.ZapDataKeyspace, pCtx.ZapRun, pCtx.ZapNode, pCtx.ZapBatchIdx, pCtx.ZapMsgAgeMillis)
	}
}
