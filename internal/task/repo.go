package task

import (
	"context"
	"github.com/995933447/optionstream"
)

type TaskRepo interface {
	TimeoutTasks(ctx context.Context, size int, cursor string) (tasks []*Task, nextCursor string, err error)
	LockTask(context.Context, *Task) (bool, error)
	ConfirmTask(context.Context, *TaskResp) error
	AddTask(context.Context, *Task) (string, error)
	GetTaskById(context.Context, string) (*Task, error)
	DelTaskById(context.Context, string) error
	DelTasks(context.Context, *optionstream.Stream) error
}

type TaskCallbackSrvRepo interface {
	AddSrvRoutes(context.Context, *TaskCallbackSrv) error
	DelSrvRoutes(context.Context, *TaskCallbackSrv) error
	SetSrvRoutesPassHealthCheck(context.Context, *TaskCallbackSrv) error
	GetSrvsByIds(context.Context, []string) ([]*TaskCallbackSrv, error)
	GetSrvs(context.Context, *optionstream.QueryStream) ([]*TaskCallbackSrv, error)
}

type TaskLogRepo interface {
	SaveTaskStartedLog(context.Context, *TaskStartedLogDetail) error
	SaveTaskCallbackLog(context.Context, *TaskCallbackLogDetail) error
	SaveTaskConfirmedLog(context.Context, *TaskConfirmedLogDetail) error
	DelLogs(context.Context, *optionstream.Stream) error
}