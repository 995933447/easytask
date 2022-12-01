package callbacksrvexec

import (
	"context"
	"github.com/995933447/easytask/internal/task"
)

type TaskCallbackSrvResp struct {
	isRunInAsync bool
	isSuccess bool
	extra any
}

func (r *TaskCallbackSrvResp) IsRunInAsync() bool {
	return r.isRunInAsync
}

func (r *TaskCallbackSrvResp) IsSuccess() bool {
	return r.isSuccess
}

func (r *TaskCallbackSrvResp) GetExtra() any {
	return r.extra
}

func NewCallbackSrvResp(isRunInAsync, isSuccess bool, extra any) *TaskCallbackSrvResp {
	return &TaskCallbackSrvResp{
		isRunInAsync: isRunInAsync,
		isSuccess: isSuccess,
		extra: extra,
	}
}

type HeartBeatResp struct {
	noReplyRoutes []*task.TaskCallbackSrvRoute
	replyRoutes []*task.TaskCallbackSrvRoute
}

func (r *HeartBeatResp) GetNoReplyRoutes() []*task.TaskCallbackSrvRoute {
	return r.noReplyRoutes
}

func (r *HeartBeatResp) GetReplyRoutes() []*task.TaskCallbackSrvRoute {
	return r.replyRoutes
}

func NewHeartBeatResp(replyRoutes, noReplyRoutes []*task.TaskCallbackSrvRoute) *HeartBeatResp {
	 return &HeartBeatResp{
		 noReplyRoutes: replyRoutes,
		 replyRoutes: noReplyRoutes,
	 }
}

type TaskCallbackSrvExec interface {
	CallbackSrv(ctx context.Context, task *task.Task, extra interface{}) (*TaskCallbackSrvResp, error)
	HeartBeat(context.Context, *task.TaskCallbackSrv) (*HeartBeatResp, error)
}
