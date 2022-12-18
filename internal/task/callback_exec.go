package task

import (
	"context"
)

type TaskCallbackSrvExec interface {
	CallbackSrv(ctx context.Context, task *Task, extra interface{}) (*TaskCallbackSrvResp, error)
	HeartBeat(context.Context, *TaskCallbackSrv) (*HeartBeatResp, error)
}

func NewCallbackSrvResp(isRunInAsync, isSuccess bool, extra any) *TaskCallbackSrvResp {
	return &TaskCallbackSrvResp{
		isRunInAsync: isRunInAsync,
		isSuccess: isSuccess,
		extra: extra,
	}
}

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

type HeartBeatResp struct {
	noReplyRoutes []*TaskCallbackSrvRoute
	replyRoutes []*TaskCallbackSrvRoute
}

func (r *HeartBeatResp) GetNoReplyRoutes() []*TaskCallbackSrvRoute {
	return r.noReplyRoutes
}

func (r *HeartBeatResp) GetReplyRoutes() []*TaskCallbackSrvRoute {
	return r.replyRoutes
}

func NewHeartBeatResp(replyRoutes, noReplyRoutes []*TaskCallbackSrvRoute) *HeartBeatResp {
	 return &HeartBeatResp{
		 noReplyRoutes: replyRoutes,
		 replyRoutes: noReplyRoutes,
	 }
}
