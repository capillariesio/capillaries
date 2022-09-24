package l

import (
	"fmt"
	"os"
	"sync/atomic"
	"time"

	"github.com/kleineshertz/capillaries/pkg/ctx"
	"github.com/kleineshertz/capillaries/pkg/env"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Logger struct {
	Logger              *zap.Logger
	ZapMachine          zapcore.Field
	ZapThread           zapcore.Field
	SavedZapConfig      zap.Config
	AtomicThreadCounter *int64
	ZapFunction         zapcore.Field
	FunctionStack       []string
}

func (logger *Logger) PushF(functionName string) {
	logger.ZapFunction = zap.String("f", functionName)
	logger.FunctionStack = append(logger.FunctionStack, functionName)
}
func (logger *Logger) PopF() {
	if len(logger.FunctionStack) > 0 {
		logger.FunctionStack = logger.FunctionStack[:len(logger.FunctionStack)-1]
		if len(logger.FunctionStack) > 0 {
			logger.ZapFunction = zap.String("f", logger.FunctionStack[len(logger.FunctionStack)-1])
		} else {
			logger.ZapFunction = zap.String("f", "stack_underflow")
		}
	}
}

func NewLoggerFromEnvConfig(envConfig *env.EnvConfig) (*Logger, error) {
	atomicTreadCounter := int64(0)
	l := Logger{AtomicThreadCounter: &atomicTreadCounter}
	hostName, err := os.Hostname()
	if err != nil {
		return nil, fmt.Errorf("cannot get hostname: %s", err.Error())
	}
	l.ZapMachine = zap.String("i", fmt.Sprintf("%s/%s/%s", hostName, envConfig.HandlerExecutableType, time.Now().Format("01-02T15:04:05.000")))
	l.ZapThread = zap.Int64("t", 0)
	l.ZapFunction = zap.String("f", "")

	l.SavedZapConfig = envConfig.ZapConfig
	l.Logger, err = envConfig.ZapConfig.Build()
	if err != nil {
		return nil, fmt.Errorf("cannot build logger from config: %s", err.Error())
	}
	return &l, nil
}

func NewLoggerFromLogger(srcLogger *Logger) (*Logger, error) {
	l := Logger{
		SavedZapConfig:      srcLogger.SavedZapConfig,
		AtomicThreadCounter: srcLogger.AtomicThreadCounter,
		ZapMachine:          srcLogger.ZapMachine,
		ZapFunction:         zap.String("f", ""),
		ZapThread:           zap.Int64("t", atomic.AddInt64(srcLogger.AtomicThreadCounter, 1))}

	var err error
	l.Logger, err = srcLogger.SavedZapConfig.Build()
	if err != nil {
		return nil, fmt.Errorf("cannot build logger from logger: %s", err.Error())
	}
	return &l, nil
}

func (l *Logger) Close() {
	l.Logger.Sync()
}

func (l *Logger) Debug(format string, a ...interface{}) {
	l.Logger.Debug(fmt.Sprintf(format, a...), l.ZapMachine, l.ZapThread, l.ZapFunction)
}

func (l *Logger) DebugCtx(pCtx *ctx.MessageProcessingContext, format string, a ...interface{}) {
	if pCtx == nil {
		l.Logger.Debug(fmt.Sprintf(format, a...), l.ZapMachine, l.ZapThread, l.ZapFunction)
	} else {
		l.Logger.Debug(fmt.Sprintf(format, a...), l.ZapMachine, l.ZapThread, l.ZapFunction, pCtx.ZapDataKeyspace, pCtx.ZapRun, pCtx.ZapNode, pCtx.ZapBatchIdx, pCtx.ZapMsgAgeMillis)
	}
}

func (l *Logger) Info(format string, a ...interface{}) {
	l.Logger.Info(fmt.Sprintf(format, a...), l.ZapMachine, l.ZapThread, l.ZapFunction)
}

func (l *Logger) InfoCtx(pCtx *ctx.MessageProcessingContext, format string, a ...interface{}) {
	if pCtx == nil {
		l.Logger.Info(fmt.Sprintf(format, a...), l.ZapMachine, l.ZapThread, l.ZapFunction)
	} else {
		l.Logger.Info(fmt.Sprintf(format, a...), l.ZapMachine, l.ZapThread, l.ZapFunction, pCtx.ZapDataKeyspace, pCtx.ZapRun, pCtx.ZapNode, pCtx.ZapBatchIdx, pCtx.ZapMsgAgeMillis)
	}
}

func (l *Logger) Warn(format string, a ...interface{}) {
	l.Logger.Warn(fmt.Sprintf(format, a...), l.ZapMachine, l.ZapThread, l.ZapFunction)
}

func (l *Logger) WarnCtx(pCtx *ctx.MessageProcessingContext, format string, a ...interface{}) {
	if pCtx == nil {
		l.Logger.Warn(fmt.Sprintf(format, a...), l.ZapMachine, l.ZapThread, l.ZapFunction)
	} else {
		l.Logger.Warn(fmt.Sprintf(format, a...), l.ZapMachine, l.ZapThread, l.ZapFunction, pCtx.ZapDataKeyspace, pCtx.ZapRun, pCtx.ZapNode, pCtx.ZapBatchIdx, pCtx.ZapMsgAgeMillis)
	}
}

func (l *Logger) Error(format string, a ...interface{}) {
	l.Logger.Error(fmt.Sprintf(format, a...), l.ZapMachine, l.ZapThread, l.ZapFunction)
}

func (l *Logger) ErrorCtx(pCtx *ctx.MessageProcessingContext, format string, a ...interface{}) {
	if pCtx == nil {
		l.Logger.Error(fmt.Sprintf(format, a...), l.ZapMachine, l.ZapThread, l.ZapFunction)
	} else {
		l.Logger.Error(fmt.Sprintf(format, a...), l.ZapMachine, l.ZapThread, l.ZapFunction, pCtx.ZapDataKeyspace, pCtx.ZapRun, pCtx.ZapNode, pCtx.ZapBatchIdx, pCtx.ZapMsgAgeMillis)
	}
}
