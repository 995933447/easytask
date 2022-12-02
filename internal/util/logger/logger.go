package logger

import (
	"github.com/995933447/log-go"
	"github.com/995933447/log-go/impls/loggerwriters"
	"sync"
)

type Conf struct{
	MainProcLogDir string `json:"main_proc_log_dir"`
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
	mainProcLogger *log.Logger
	newMainProcLoggerMu sync.Mutex

	taskProcLogger *log.Logger
	newTaskProcLoggerMu sync.Mutex
)

func MustGetMainProcLogger() *log.Logger {
	if !hasInit() {
		panic(any("init not finish"))
	}
	if mainProcLogger != nil {
		return mainProcLogger
	}
	newMainProcLoggerMu.Lock()
	defer newMainProcLoggerMu.Unlock()
	if mainProcLogger != nil {
		return mainProcLogger
	}
	mainProcLogger = NewLogger(conf.MainProcLogDir, conf.FileSize)
	return mainProcLogger
}

func MustGetTaskProcLogger() *log.Logger {
	if !hasInit() {
		panic(any("init not finish"))
	}
	if taskProcLogger != nil {
		return taskProcLogger
	}
	newTaskProcLoggerMu.Lock()
	defer newTaskProcLoggerMu.Unlock()
	if taskProcLogger != nil {
		return taskProcLogger
	}
	taskProcLogger = NewLogger(conf.TaskProcLogDir, conf.FileSize)
	return taskProcLogger
}

func NewLogger(baseDir string, maxFileSize int64) *log.Logger {
	writer := loggerwriters.NewFileLoggerWriter(baseDir, maxFileSize, 5, loggerwriters.OpenNewFileByByDateHour, 100000)
	go func() {
		err := writer.Loop()
		if err != nil {
			panic(any(err))
		}
	}()
	return log.NewLogger(writer)
}
