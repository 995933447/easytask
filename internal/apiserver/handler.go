package apiserver

import (
	"context"
	"github.com/995933447/easytask/internal/repo"
	"github.com/995933447/easytask/internal/task"
	"github.com/995933447/easytask/internal/util/logger"
	"github.com/995933447/easytask/pkg/errs"
	"github.com/995933447/easytask/pkg/rpc/proto/httpproto"
	"github.com/995933447/optionstream"
)

func AddTaskHandler(taskRepo repo.TaskRepo, taskCallbackSrvRepo repo.TaskCallbackSrvRepo) func(ctx context.Context, req *httpproto.AddTaskReq) (*httpproto.AddTaskResp, error) {
	return func(ctx context.Context, req *httpproto.AddTaskReq) (*httpproto.AddTaskResp, error) {
		log := logger.MustGetSessLogger()

		srvs, err := taskCallbackSrvRepo.GetSrvs(
			ctx,
			optionstream.NewQueryStream(nil, 1, 0).SetOption(repo.QueryOptKeyEqName, req.SrvName),
			)
		if err != nil {
			log.Error(ctx, err)
			return nil, err
		}

		if len(srvs) == 0 {
			log.Warnf(ctx, "task callback server(name:%s) not found", req.SrvName)
			return nil, NewHandleErr(errs.ErrCodeTaskCallbackSrvNotFound, err)
		}

		oneTask, err := task.NewTask(&task.NewTaskReq{
			CallbackSrv: task.NewTaskCallbackSrv("", req.SrvName, nil, false),
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

func RegisterTaskCallbackSrvHandler(taskCallbackSrvRepo repo.TaskCallbackSrvRepo) func(ctx context.Context, req *httpproto.RegisterTaskCallbackSrvReq) (*httpproto.RegisterTaskCallbackSrvResp, error) {
	return func(ctx context.Context, req *httpproto.RegisterTaskCallbackSrvReq) (*httpproto.RegisterTaskCallbackSrvResp, error) {
		routes := []*task.TaskCallbackSrvRoute{
			task.NewTaskCallbackSrvRoute("", req.Schema, req.Host, req.Port, req.CallbackTimeoutSec, req.IsEnableHealthCheck),
		}
		err := taskCallbackSrvRepo.AddSrvRoutes(ctx, task.NewTaskCallbackSrv("", req.Name, routes, req.IsEnableHealthCheck))
		if err != nil {
			logger.MustGetSessLogger().Error(ctx, err)
			return nil, err
		}
		return &httpproto.RegisterTaskCallbackSrvResp{}, nil
	}
}

func UnregisterTaskCallbackSrvHandler(taskCallbackSrvRepo repo.TaskCallbackSrvRepo) func(ctx context.Context, req *httpproto.UnregisterTaskCallbackSrvReq) (*httpproto.UnregisterTaskCallbackSrvResp, error) {
	return func(ctx context.Context, req *httpproto.UnregisterTaskCallbackSrvReq) (*httpproto.UnregisterTaskCallbackSrvResp, error) {
		routes := []*task.TaskCallbackSrvRoute{
			task.NewTaskCallbackSrvRoute("", req.Schema, req.Host, req.Port, 0, false),
		}
		err := taskCallbackSrvRepo.DelSrvRoutes(ctx, task.NewTaskCallbackSrv("", req.Name, routes, false))
		if err != nil {
			logger.MustGetSessLogger().Error(ctx, err)
			return nil, err
		}
		return &httpproto.UnregisterTaskCallbackSrvResp{}, nil
	}
}