package logger

import (
	"context"
	"github.com/995933447/log-go"
	"github.com/995933447/log-go/impls/loggerwriters"
	"strings"
	"sync"
	"sync/atomic"
)

type Logger interface {
	Debug(ctx context.Context, content interface{})

	Info(ctx context.Context, content interface{})

	Warn(ctx context.Context, content interface{})

	Error(ctx context.Context, content interface{})

	Fatal(ctx context.Context, content interface{})

	Panic(ctx context.Context, content interface{})

	Debugf(ctx context.Context, format string, args ...interface{})

	Infof(ctx context.Context, format string, args ...interface{})

	Warnf(ctx context.Context, format string, args ...interface{})

	Errorf(ctx context.Context, format string, args ...interface{})

	Fatalf(ctx context.Context, format string, args ...interface{})

	Panicf(ctx context.Context, format string, args ...interface{})
}

type Conf struct{
	LogDir string `json:"log_dir"`
	FileSize int64 `json:"file_size"`
	Level string `json:"level"`
}

var (
	hasInit atomic.Bool
	initMu sync.Mutex
)

var (
	sysLogger Logger
	sessionLogger Logger
	electLogger Logger
	repoLogger Logger
	taskLogger Logger
	registryLogger Logger
	callbackLogger Logger
)

func Init(conf *Conf) {
	if hasInit.Load() {
		return
	}
	initMu.Lock()
	defer initMu.Unlock()
	if hasInit.Load() {
		return
	}
	conf.LogDir = strings.TrimRight(conf.LogDir, "/")
	sysLogger = NewLogger(conf.LogDir + "/sys", conf.FileSize, conf.Level)
	sessionLogger = NewLogger(conf.LogDir + "/session", conf.FileSize, conf.Level)
	electLogger = NewLogger(conf.LogDir + "/elect", conf.FileSize, conf.Level)
	repoLogger = NewLogger(conf.LogDir + "/repo", conf.FileSize, conf.Level)
	taskLogger = NewLogger(conf.LogDir + "/task", conf.FileSize, conf.Level)
	registryLogger = NewLogger(conf.LogDir + "/registry", conf.FileSize, conf.Level)
	callbackLogger = NewLogger(conf.LogDir + "/callback", conf.FileSize, conf.Level)
	hasInit.Store(true)
}

func MustGetRegistryLogger() Logger {
	if !hasInit.Load() {
		panic(any("init not finish"))
	}
	return registryLogger
}

func MustGetTaskLogger() Logger {
	if !hasInit.Load() {
		panic(any("init not finish"))
	}
	return taskLogger
}

func MustGetElectLogger() Logger {
	if !hasInit.Load() {
		panic(any("init not finish"))
	}
	return electLogger
}

func MustGetRepoLogger() Logger {
	if !hasInit.Load() {
		panic(any("init not finish"))
	}
	return repoLogger
}

func MustGetSysLogger() Logger {
	if !hasInit.Load() {
		panic(any("init not finish"))
	}
	return sysLogger
}

func MustGetSessLogger() Logger {
	if !hasInit.Load() {
		panic(any("init not finish"))
	}
	return sessionLogger
}

func MustGetCallbackLogger() Logger {
	if !hasInit.Load() {
		panic(any("init not finish"))
	}
	return callbackLogger
}

func NewLogger(baseDir string, maxFileSize int64, level string) Logger {
	writer := loggerwriters.NewFileLoggerWriter(
		baseDir,
		maxFileSize,
		5,
		loggerwriters.OpenNewFileByByDateHour,
		100000,
		)
	go func() {
		err := writer.Loop()
		if err != nil {
			panic(any(err))
		}
	}()
	logger := log.NewLogger(writer)
	switch strings.ToLower(level) {
	case "debug":
		logger.SetLogLevel(log.LevelDebug)
	case "info":
		logger.SetLogLevel(log.LevelInfo)
	case "warn":
		logger.SetLogLevel(log.LevelWarn)
	case "error":
		logger.SetLogLevel(log.LevelError)
	case "panic":
		logger.SetLogLevel(log.LevelPanic)
	case "fatal":
		logger.SetLogLevel(log.LevelFatal)
	}
	return logger
}
