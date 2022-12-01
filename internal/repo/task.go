package repo

import (
	"context"
	"github.com/995933447/easytask/internal/task"
	"github.com/995933447/optionstream"
)

type TaskRepo interface {
	TimeoutTasks(ctx context.Context, size int) ([]*task.Task, error)
	LockTask(context.Context, *task.Task) (bool, error)
	ConfirmTasks(context.Context, []*task.TaskResp) error
}

type TaskCallbackSrvRepo interface {
	AddSrvRoutes(context.Context, *task.TaskCallbackSrv) error
	DelSrvRoutes(context.Context, *task.TaskCallbackSrv) error
	SetSrvRoutesPassHealthCheck(context.Context, *task.TaskCallbackSrv) error
	GetSrvsByIds(context.Context, []string) ([]*task.TaskCallbackSrv, error)
	GetSrvs(context.Context, *optionstream.QueryStream) ([]*task.TaskCallbackSrv, error)
}