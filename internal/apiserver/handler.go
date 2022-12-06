package apiserver

import (
	"context"
	"github.com/995933447/easytask/internal/registry"
	"github.com/995933447/easytask/internal/repo"
	"github.com/995933447/easytask/internal/task"
	"github.com/995933447/easytask/internal/util/logger"
	"github.com/995933447/easytask/pkg/rpc/proto/httpproto"
)

func AddTaskHandler(taskRepo repo.TaskRepo, reg *registry.Registry) func(ctx context.Context, req *httpproto.AddTaskReq) (*httpproto.AddTaskResp, error) {
	return func(ctx context.Context, req *httpproto.AddTaskReq) (*httpproto.AddTaskResp, error) {
		log := logger.MustGetSessLogger()

		srv, err := reg.Discover(ctx, req.SrvName)
		if err != nil {
			log.Error(ctx, err)
			return nil, err
		}

		oneTask, err := task.NewTask(&task.NewTaskReq{
			CallbackSrv: srv,
			CallbackPath: req.CallbackPath,
			Name: req.Name,
		})
		if err != nil {
			log.Error(ctx, err)
			return nil, err
		}

		taskId, err := taskRepo.AddTask(ctx, oneTask)
		if err != nil {
			log.Error(ctx, err)
			return nil, err
		}

		return &httpproto.AddTaskResp{Id: taskId}, nil
	}
}

func DelTaskHandler(taskRepo repo.TaskRepo) func(ctx context.Context, req *httpproto.DelTaskReq) (*httpproto.DelTaskResp, error) {
	return func(ctx context.Context, req *httpproto.DelTaskReq) (*httpproto.DelTaskResp, error) {
		if err := taskRepo.DelTaskById(ctx, req.Id); err != nil {
			logger.MustGetSessLogger().Error(ctx, err)
			return nil, err
		}
		return &httpproto.DelTaskResp{}, nil
	}
}

func RegisterTaskCallbackSrvHandler(reg *registry.Registry) func(ctx context.Context, req *httpproto.RegisterTaskCallbackSrvReq) (*httpproto.RegisterTaskCallbackSrvResp, error) {
	return func(ctx context.Context, req *httpproto.RegisterTaskCallbackSrvReq) (*httpproto.RegisterTaskCallbackSrvResp, error) {
		routes := []*task.TaskCallbackSrvRoute{
			task.NewTaskCallbackSrvRoute("", req.Schema, req.Host, req.Port, req.CallbackTimeoutSec, req.IsEnableHealthCheck),
		}
		err := reg.Register(ctx, task.NewTaskCallbackSrv("", req.Name, routes, req.IsEnableHealthCheck))
		if err != nil {
			logger.MustGetSessLogger().Error(ctx, err)
			return nil, err
		}
		return &httpproto.RegisterTaskCallbackSrvResp{}, nil
	}
}

func UnregisterTaskCallbackSrvHandler(reg *registry.Registry) func(ctx context.Context, req *httpproto.UnregisterTaskCallbackSrvReq) (*httpproto.UnregisterTaskCallbackSrvResp, error) {
	return func(ctx context.Context, req *httpproto.UnregisterTaskCallbackSrvReq) (*httpproto.UnregisterTaskCallbackSrvResp, error) {
		routes := []*task.TaskCallbackSrvRoute{
			task.NewTaskCallbackSrvRoute("", req.Schema, req.Host, req.Port, 0, false),
		}
		err := reg.Unregister(ctx, task.NewTaskCallbackSrv("", req.Name, routes, false))
		if err != nil {
			logger.MustGetSessLogger().Error(ctx, err)
			return nil, err
		}
		return &httpproto.UnregisterTaskCallbackSrvResp{}, nil
	}
}