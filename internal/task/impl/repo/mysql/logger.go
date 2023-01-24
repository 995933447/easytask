package mysql

import (
	"context"
	"errors"
	"fmt"
	log "github.com/995933447/easytask/internal/util/logger"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/utils"
	"time"
)

type Logger struct {
	baseLogger log.Logger
	slowThreshold time.Duration
}

func NewLogger(baseLogger log.Logger, slowThreshold time.Duration) *Logger {
	return &Logger{
		baseLogger: baseLogger,
		slowThreshold: slowThreshold,
	}
}

const (
	traceStr     = "%s [%.3fms] [rows:%v] %s"
	traceWarnStr = "%s %s [%.3fms] [rows:%v] %s"
	traceErrStr  = "%s %s [%.3fms] [rows:%v] %s"
)

func (l *Logger) LogMode(_ logger.LogLevel) logger.Interface  {
	return l
}

func (l *Logger) Info(ctx context.Context, format string,  args ...interface{}) {
	l.baseLogger.Infof(ctx, format, args...)
}

func (l *Logger) Warn(ctx context.Context, format string,  args ...interface{}) {
	l.baseLogger.Warnf(ctx, format, args...)
}

func (l *Logger) Error(ctx context.Context, format string,  args ...interface{}) {
	l.baseLogger.Errorf(ctx, format, args...)
}

func (l *Logger) Trace(ctx context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
	elapsed := time.Since(begin)
	switch {
	case err != nil && (!errors.Is(err, gorm.ErrRecordNotFound)):
		sql, rows := fc()
		if rows == -1 {
			l.Error(ctx, traceErrStr, utils.FileWithLineNum(), err, float64(elapsed.Nanoseconds())/1e6, "-", sql)
		} else {
			l.Error(ctx, traceErrStr, utils.FileWithLineNum(), err, float64(elapsed.Nanoseconds())/1e6, rows, sql)
		}
	case elapsed > l.slowThreshold && l.slowThreshold != 0:
		sql, rows := fc()
		slowLog := fmt.Sprintf("SLOW SQL >= %v", l.slowThreshold)
		if rows == -1 {
			l.Warn(ctx, traceWarnStr, utils.FileWithLineNum(), slowLog, float64(elapsed.Nanoseconds())/1e6, "-", sql)
		} else {
			l.Warn(ctx, traceWarnStr, utils.FileWithLineNum(), slowLog, float64(elapsed.Nanoseconds())/1e6, rows, sql)
		}
	default:
		sql, rows := fc()
		if rows == -1 {
			l.baseLogger.Infof(ctx, traceStr, utils.FileWithLineNum(), float64(elapsed.Nanoseconds())/1e6, "-", sql)
		} else {
			l.baseLogger.Infof(ctx, traceStr, utils.FileWithLineNum(), float64(elapsed.Nanoseconds())/1e6, rows, sql)
		}
	}
}
