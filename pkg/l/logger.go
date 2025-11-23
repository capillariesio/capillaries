package l

import (
	"fmt"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/capillariesio/capillaries/pkg/ctx"
	"github.com/capillariesio/capillaries/pkg/env"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

type CapiLogger struct {
	ZapLogger  *zap.Logger
	ZapMachine zapcore.Field
	ZapThread  zapcore.Field
	// SavedZapConfig      zap.Config
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
	atomicLevel, err := zap.ParseAtomicLevel(envConfig.Log.Level)
	if err != nil {
		return nil, fmt.Errorf("cannot parse zap atomic level %s from config: %s", envConfig.Log.Level, err.Error())
	}

	encoderConfig := zapcore.EncoderConfig{
		TimeKey:     "ts",
		EncodeTime:  zapcore.ISO8601TimeEncoder,
		MessageKey:  "m",
		LevelKey:    "l",
		EncodeLevel: zapcore.LowercaseLevelEncoder,
	}

	var core zapcore.Core
	if envConfig.Log.LogFile == "" {
		core = zapcore.NewTee(
			zapcore.NewCore(zapcore.NewJSONEncoder(encoderConfig), zapcore.AddSync(os.Stdout), atomicLevel),
		)
	} else {
		// Lumberjack: rotates by schedule and on SIGHUP
		lj := lumberjack.Logger{
			Filename:   envConfig.Log.LogFile, // /var/log/capillaries/capiwebapi.log
			MaxSize:    1,                     // megabytes
			MaxBackups: 10,
			MaxAge:     1, // days
			Compress:   true,
		}
		ljChan := make(chan os.Signal, 1)
		signal.Notify(ljChan, syscall.SIGHUP)
		go func() {
			for {
				<-ljChan
				if err := lj.Rotate(); err != nil {
					if l.ZapLogger != nil {
						l.ZapLogger.Error(fmt.Sprintf("cannot rotate log file: %s", err.Error()), l.ZapMachine, l.ZapThread, l.ZapFunction)
					}
				}
			}
		}()
		lj_file := zapcore.AddSync(&lj)
		core = zapcore.NewTee(
			zapcore.NewCore(zapcore.NewJSONEncoder(encoderConfig), zapcore.AddSync(os.Stdout), atomicLevel),
			zapcore.NewCore(zapcore.NewJSONEncoder(encoderConfig), lj_file, atomicLevel),
		)
	}
	l.ZapLogger = zap.New(core)

	// // Make it configurable via envConfig.Log if needed
	// l.SavedZapConfig = zap.Config{
	// 	Level:            atomicLevel,
	// 	OutputPaths:      []string{"stdout"},
	// 	ErrorOutputPaths: []string{"stderr"},
	// 	Encoding:         "json",
	// 	EncoderConfig: zapcore.EncoderConfig{
	// 		TimeKey:     "ts",
	// 		EncodeTime:  zapcore.ISO8601TimeEncoder,
	// 		MessageKey:  "m",
	// 		LevelKey:    "l",
	// 		EncodeLevel: zapcore.LowercaseLevelEncoder,
	// 	},
	// }
	// l.ZapLogger, err = l.SavedZapConfig.Build()
	// if err != nil {
	// 	return nil, fmt.Errorf("cannot build l from config: %s", err.Error())
	// }

	return &l, nil
}

func NewLoggerFromLogger(srcLogger *CapiLogger) (*CapiLogger, error) {
	l := CapiLogger{
		// SavedZapConfig:      srcLogger.SavedZapConfig,
		AtomicThreadCounter: srcLogger.AtomicThreadCounter,
		ZapMachine:          srcLogger.ZapMachine,
		ZapFunction:         zap.String("f", ""),
		ZapThread:           zap.Int64("t", atomic.AddInt64(srcLogger.AtomicThreadCounter, 1)),
		ZapLogger:           srcLogger.ZapLogger}

	// var err error
	// l.ZapLogger, err = srcLogger.SavedZapConfig.Build()
	// if err != nil {
	// 	return nil, fmt.Errorf("cannot build l from l: %s", err.Error())
	// }
	return &l, nil
}

func (l *CapiLogger) Close() {
	l.ZapLogger.Sync()
}

func (l *CapiLogger) Debug(format string, a ...any) {
	l.ZapLogger.Debug(fmt.Sprintf(format, a...), l.ZapMachine, l.ZapThread, l.ZapFunction)
}

func (l *CapiLogger) DebugCtx(pCtx *ctx.MessageProcessingContext, format string, a ...any) {
	if pCtx == nil {
		l.ZapLogger.Debug(fmt.Sprintf(format, a...), l.ZapMachine, l.ZapThread, l.ZapFunction)
	} else {
		l.ZapLogger.Debug(fmt.Sprintf(format, a...), l.ZapMachine, l.ZapThread, l.ZapFunction, pCtx.ZapMsgId, pCtx.ZapDataKeyspace, pCtx.ZapRun, pCtx.ZapNode, pCtx.ZapBatchIdx, pCtx.ZapMsgAgeMillis)
	}
}

func (l *CapiLogger) Info(format string, a ...any) {
	l.ZapLogger.Info(fmt.Sprintf(format, a...), l.ZapMachine, l.ZapThread, l.ZapFunction)
}

func (l *CapiLogger) InfoCtx(pCtx *ctx.MessageProcessingContext, format string, a ...any) {
	if pCtx == nil {
		l.ZapLogger.Info(fmt.Sprintf(format, a...), l.ZapMachine, l.ZapThread, l.ZapFunction)
	} else {
		l.ZapLogger.Info(fmt.Sprintf(format, a...), l.ZapMachine, l.ZapThread, l.ZapFunction, pCtx.ZapMsgId, pCtx.ZapDataKeyspace, pCtx.ZapRun, pCtx.ZapNode, pCtx.ZapBatchIdx, pCtx.ZapMsgAgeMillis)
	}
}

func (l *CapiLogger) Warn(format string, a ...any) {
	l.ZapLogger.Warn(fmt.Sprintf(format, a...), l.ZapMachine, l.ZapThread, l.ZapFunction)
}

func (l *CapiLogger) WarnCtx(pCtx *ctx.MessageProcessingContext, format string, a ...any) {
	if pCtx == nil {
		l.ZapLogger.Warn(fmt.Sprintf(format, a...), l.ZapMachine, l.ZapThread, l.ZapFunction)
	} else {
		l.ZapLogger.Warn(fmt.Sprintf(format, a...), l.ZapMachine, l.ZapThread, l.ZapFunction, pCtx.ZapMsgId, pCtx.ZapDataKeyspace, pCtx.ZapRun, pCtx.ZapNode, pCtx.ZapBatchIdx, pCtx.ZapMsgAgeMillis)
	}
}

func (l *CapiLogger) Error(format string, a ...any) {
	l.ZapLogger.Error(fmt.Sprintf(format, a...), l.ZapMachine, l.ZapThread, l.ZapFunction)
}

func (l *CapiLogger) ErrorCtx(pCtx *ctx.MessageProcessingContext, format string, a ...any) {
	if pCtx == nil {
		l.ZapLogger.Error(fmt.Sprintf(format, a...), l.ZapMachine, l.ZapThread, l.ZapFunction)
	} else {
		l.ZapLogger.Error(fmt.Sprintf(format, a...), l.ZapMachine, l.ZapThread, l.ZapFunction, pCtx.ZapMsgId, pCtx.ZapDataKeyspace, pCtx.ZapRun, pCtx.ZapNode, pCtx.ZapBatchIdx, pCtx.ZapMsgAgeMillis)
	}
}
