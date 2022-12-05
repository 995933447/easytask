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
	TaskProcLogDir string `json:"task_proc_log_dir"`
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
	sysProcLogger Logger
	newSysProcLoggerMu sync.Mutex

	sessionProcLogger Logger
	newSessionProcLoggerMu sync.Mutex
)

func MustGetSysProcLogger() Logger {
	if !hasInit() {
		panic(any("init not finish"))
	}
	if sysProcLogger != nil {
		return sysProcLogger
	}
	newSysProcLoggerMu.Lock()
	defer newSessionProcLoggerMu.Unlock()
	if sysProcLogger != nil {
		return sysProcLogger
	}
	sysProcLogger = NewLogger(conf.SysProcLogDir, conf.FileSize)
	return sysProcLogger
}

func MustGetSessProcLogger() Logger {
	if !hasInit() {
		panic(any("init not finish"))
	}
	if sessionProcLogger != nil {
		return sessionProcLogger
	}
	newSessionProcLoggerMu.Lock()
	defer newSessionProcLoggerMu.Unlock()
	if sessionProcLogger != nil {
		return sessionProcLogger
	}
	sessionProcLogger = NewLogger(conf.TaskProcLogDir, conf.FileSize)
	return sessionProcLogger
}

func NewLogger(baseDir string, maxFileSize int64) Logger {
	writer := loggerwriters.NewFileLoggerWriter(baseDir, maxFileSize, 5, loggerwriters.OpenNewFileByByDateHour, 100000)
	go func() {
		err := writer.Loop()
		if err != nil {
			panic(any(err))
		}
	}()
	return log.NewLogger(writer)
}
