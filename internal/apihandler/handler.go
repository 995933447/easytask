package apihandler

import (
	"context"
	"github.com/995933447/easytask/internal/repo"
	"github.com/995933447/easytask/internal/task"
	"github.com/995933447/easytask/internal/util/logger"
	"github.com/995933447/easytask/pkg/rpc/proto/httpproto"
)

func AddTaskHandler(taskRepo repo.TaskRepo) func(ctx context.Context, req *httpproto.AddTaskReq) (*httpproto.AddTaskResp, error) {
	return func(ctx context.Context, req *httpproto.AddTaskReq) (*httpproto.AddTaskResp, error) {
		var resp httpproto.AddTaskResp

		return &resp, nil
	}
}

func AddTaskCallbackSrvHandler(taskCallbackSrvRepo repo.TaskCallbackSrvRepo) func(ctx context.Context, req httpproto.AddTaskCallbackSrvReq) (*httpproto.AddTaskCallbackSrvResp, error) {
	return func(ctx context.Context, req httpproto.AddTaskCallbackSrvReq) (*httpproto.AddTaskCallbackSrvResp, error) {
		var resp httpproto.AddTaskCallbackSrvResp
		routes := []*task.TaskCallbackSrvRoute{
			task.NewTaskCallbackSrvRoute("", req.Schema, req.Host, req.Port, req.CallbackTimeoutSec, req.IsEnableHealthCheck),
		}
		err := taskCallbackSrvRepo.AddSrvRoutes(ctx, task.NewTaskCallbackSrv(req.Name, routes, req.IsEnableHealthCheck))
		if err != nil {
			logger.MustGetSessLogger().Error(ctx, err)
			return nil, err
		}
		return &resp, nil
	}
}