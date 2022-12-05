package logger

import (
	"context"
	"github.com/995933447/log-go"
	"github.com/995933447/log-go/impls/loggerwriters"
	"sync"
)

type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
	LevelPanic
	LevelFatal
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
	SysProcLogDir string `json:"sys_proc_log_dir"`
	SessionProcLogDir string `json:"session_proc_log_dir"`
	FileSize int64 `json:"file_size"`
}

var conf *Conf

func Init(cfg *Conf) {
	conf = cfg
}

func hasInit() bool {
	return conf == nil
}

var (
	sysLogger Logger
	newSysLoggerMu sync.Mutex

	sessionLogger Logger
	newSessionLoggerMu sync.Mutex
)

func MustGetSysLogger() Logger {
	if !hasInit() {
		panic(any("init not finish"))
	}
	if sysLogger != nil {
		return sysLogger
	}
	newSysLoggerMu.Lock()
	defer newSessionLoggerMu.Unlock()
	if sysLogger != nil {
		return sysLogger
	}
	sysLogger = NewLogger(conf.SysProcLogDir, conf.FileSize)
	return sysLogger
}

func MustGetSessLogger() Logger {
	if !hasInit() {
		panic(any("init not finish"))
	}
	if sessionLogger != nil {
		return sessionLogger
	}
	newSessionLoggerMu.Lock()
	defer newSessionLoggerMu.Unlock()
	if sessionLogger != nil {
		return sessionLogger
	}
	sessionLogger = NewLogger(conf.SessionProcLogDir, conf.FileSize)
	return sessionLogger
}

func NewLogger(baseDir string, maxFileSize int64) Logger {
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
	return log.NewLogger(writer)
}
