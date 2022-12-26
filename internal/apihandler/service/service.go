package service

import (
	"context"
	"github.com/995933447/easytask/internal/registry"
	"github.com/995933447/easytask/internal/task"
	"github.com/995933447/easytask/internal/util/logger"
)

func NewTaskService(taskRepo task.TaskRepo, reg *registry.Registry) *TaskService {
	return &TaskService{
		taskRepo: taskRepo,
		reg: reg,
	}
}

type TaskService struct {
	taskRepo task.TaskRepo
	reg      *registry.Registry
}

func (s *TaskService) AddTask(ctx context.Context, req *AddTaskReq) (*AddTaskResp, error) {
	srv, err := s.reg.Discover(ctx, req.SrvName)
	if err != nil {
		logger.MustGetSessLogger().Error(ctx, err)
		return nil, err
	}

	oneTask, err := task.NewTask(&task.NewTaskReq{
		CallbackSrv: srv,
		CallbackPath: req.CallbackPath,
		Name: req.Name,
		Arg: req.Arg,
		SchedMode: req.SchedMode,
		TimeSpecAt: req.TimeSpecAt,
		TimeCronExpr: req.TimeCron,
		TimeIntervalSec: req.TimeIntervalSec,
		BizId: req.BizId,
	})
	if err != nil {
		logger.MustGetSessLogger().Error(ctx, err)
		return nil, err
	}

	taskId, err := s.taskRepo.AddTask(ctx, oneTask)
	if err != nil {
		logger.MustGetSessLogger().Error(ctx, err)
		return nil, err
	}

	return &AddTaskResp{TaskId: taskId}, nil
}

func (s *TaskService) StopTask(ctx context.Context, req *StopTaskReq) (*StopTaskResp, error) {
	if err := s.taskRepo.DelTaskById(ctx, req.TaskId); err != nil {
		logger.MustGetSessLogger().Error(ctx, err)
		return nil, err
	}
	return &StopTaskResp{}, nil
}

func (s *TaskService) ConfirmTask(ctx context.Context, req *ConfirmTaskReq) (*ConfirmTaskResp, error) {
	var taskStatus task.Status
	if req.IsSuccess {
		taskStatus = task.StatusSuccess
	} else {
		taskStatus = task.StatusFailed
	}
	err := s.taskRepo.ConfirmTask(ctx, task.NewTaskResp(req.TaskId, true, taskStatus, req.TaskRunTimes, req.Extra))
	if err != nil {
		logger.MustGetSessLogger().Error(ctx, err)
		return nil, err
	}
	return &ConfirmTaskResp{}, nil
}

func NewRegistryService(reg *registry.Registry) *RegistryService {
	return &RegistryService{
		reg: reg,
	}
}

type RegistryService struct {
	reg *registry.Registry
}

func (s *RegistryService) RegisterTaskCallbackSrv(ctx context.Context, req *RegisterTaskCallbackSrvReq) (*RegisterTaskCallbackSrvResp, error) {
	routes := []*task.TaskCallbackSrvRoute{
		task.NewTaskCallbackSrvRoute("", req.Schema, req.Host, req.Port, req.CallbackTimeoutSec, req.IsEnableHealthCheck),
	}
	err := s.reg.Register(ctx, task.NewTaskCallbackSrv("", req.Name, routes, req.IsEnableHealthCheck))
	if err != nil {
		logger.MustGetSessLogger().Error(ctx, err)
		return nil, err
	}
	return &RegisterTaskCallbackSrvResp{}, nil
}

func (s *RegistryService) UnregisterTaskCallbackSrv(ctx context.Context, req *UnregisterTaskCallbackSrvReq) (*UnregisterTaskCallbackSrvResp, error) {
	srv, err := s.reg.Discover(ctx, req.Name)
	if err != nil {
		logger.MustGetSessLogger().Error(ctx, err)
		return nil, err
	}

	var readyDelRoutes []*task.TaskCallbackSrvRoute
	for _, route := range srv.GetRoutes() {
		if route.GetHost() == req.Host && route.GetSchema() == req.Schema && route.GetPort() == req.Port {
			readyDelRoutes = append(readyDelRoutes, route)
		}
	}

	err = s.reg.Unregister(ctx, task.NewTaskCallbackSrv(srv.GetId(), srv.GetName(), readyDelRoutes, srv.HasEnableHealthCheckRoute()))
	if err != nil {
		logger.MustGetSessLogger().Error(ctx, err)
		return nil, err
	}
	return &UnregisterTaskCallbackSrvResp{}, nil
}