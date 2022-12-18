package apihandler

import (
	"context"
	"github.com/995933447/easytask/internal/registry"
	"github.com/995933447/easytask/internal/task"
	"github.com/995933447/easytask/internal/util/logger"
	"github.com/995933447/easytask/pkg/rpc/proto/httpproto"
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

func (s *TaskService) AddTask(ctx context.Context, req *httpproto.AddTaskReq) (*httpproto.AddTaskResp, error) {
	log := logger.MustGetSessLogger()

	srv, err := s.reg.Discover(ctx, req.SrvName)
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

	taskId, err := s.taskRepo.AddTask(ctx, oneTask)
	if err != nil {
		log.Error(ctx, err)
		return nil, err
	}

	return &httpproto.AddTaskResp{Id: taskId}, nil
}

func (s *TaskService) DelTask(ctx context.Context, req *httpproto.DelTaskReq) (*httpproto.DelTaskResp, error) {
	if err := s.taskRepo.DelTaskById(ctx, req.Id); err != nil {
		logger.MustGetSessLogger().Error(ctx, err)
		return nil, err
	}
	return &httpproto.DelTaskResp{}, nil
}

func (s *TaskService) ConfirmTask(ctx context.Context, req *httpproto.ConfirmTaskReq) (*httpproto.ConfirmTaskResp, error) {
	var taskStatus task.Status
	if req.IsSuccess {
		taskStatus = task.StatusSuccess
	} else {
		taskStatus = task.StatusFailed
	}
	err := s.taskRepo.ConfirmTask(ctx, task.NewTaskResp(req.Id, true, taskStatus, req.TaskRunTimes, nil))
	if err != nil {
		logger.MustGetSessLogger().Error(ctx, err)
		return nil, err
	}
	return &httpproto.ConfirmTaskResp{}, nil
}

func NewRegistryService(reg *registry.Registry) *RegistryService {
	return &RegistryService{
		reg: reg,
	}
}

type RegistryService struct {
	reg *registry.Registry
}

func (s *RegistryService) RegisterTaskCallbackSrv(ctx context.Context, req *httpproto.RegisterTaskCallbackSrvReq) (*httpproto.RegisterTaskCallbackSrvResp, error) {
	routes := []*task.TaskCallbackSrvRoute{
		task.NewTaskCallbackSrvRoute("", req.Schema, req.Host, req.Port, req.CallbackTimeoutSec, req.IsEnableHealthCheck),
	}
	err := s.reg.Register(ctx, task.NewTaskCallbackSrv("", req.Name, routes, req.IsEnableHealthCheck))
	if err != nil {
		logger.MustGetSessLogger().Error(ctx, err)
		return nil, err
	}
	return &httpproto.RegisterTaskCallbackSrvResp{}, nil
}

func (s *RegistryService) UnregisterTaskCallbackSrv(ctx context.Context, req *httpproto.UnregisterTaskCallbackSrvReq) (*httpproto.UnregisterTaskCallbackSrvResp, error) {
	routes := []*task.TaskCallbackSrvRoute{
		task.NewTaskCallbackSrvRoute("", req.Schema, req.Host, req.Port, 0, false),
	}
	err := s.reg.Unregister(ctx, task.NewTaskCallbackSrv("", req.Name, routes, false))
	if err != nil {
		logger.MustGetSessLogger().Error(ctx, err)
		return nil, err
	}
	return &httpproto.UnregisterTaskCallbackSrvResp{}, nil
}