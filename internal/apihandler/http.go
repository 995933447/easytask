package apihandler

import (
	"context"
	"github.com/995933447/easytask/internal/apihandler/service"
	"github.com/995933447/easytask/internal/task"
	"github.com/995933447/easytask/internal/util/logger"
	"github.com/995933447/easytask/pkg/rpc/proto"
	"github.com/995933447/easytask/pkg/rpc/proto/httpproto"
	"github.com/995933447/reflectutil"
)

type HttpApi struct {
	taskSrv *service.TaskService
	registrySrv *service.RegistryService
}

func NewHttpApi(taskSrv *service.TaskService, registrySrv *service.RegistryService) *HttpApi {
	return &HttpApi{
		taskSrv: taskSrv,
		registrySrv: registrySrv,
	}
}

func (a *HttpApi) AddTask(ctx context.Context, req *httpproto.AddTaskReq) (*httpproto.AddTaskResp, error) {
	var schedMode task.SchedMode
	switch req.SchedMode {
	case proto.SchedModeTimeCron:
		schedMode = task.SchedModeTimeCron
	case proto.SchedModeTimeSpec:
		schedMode = task.SchedModeTimeSpec
	case proto.SchedModeTimeInterval:
		schedMode = task.SchedModeTimeInterval
	}

	addTaskResp, err := a.taskSrv.AddTask(ctx, &service.AddTaskReq{
		Name: req.Name,
		SrvName: req.SrvName,
		CallbackPath: req.CallbackPath,
		SchedMode: schedMode,
		TimeSpecAt: req.TimeSpecAt,
		TimeIntervalSec: req.TimeIntervalSec,
		TimeCron: req.TimeCron,
		Arg: req.Arg,
		BizId: req.BizId,
	})
	if err != nil {
		logger.MustGetSessLogger().Error(ctx, err)
		return nil, err
	}

	return &httpproto.AddTaskResp{
		TaskId: addTaskResp.TaskId,
	}, nil
}

func (a *HttpApi) StopTask(ctx context.Context, req *httpproto.StopTaskReq) (*httpproto.StopTaskResp, error) {
	stopTaskReq := &service.StopTaskReq{}
	err := reflectutil.CopySameFields(req, stopTaskReq)
	if err != nil {
		logger.MustGetSessLogger().Error(ctx, err)
		return nil, err
	}

	_, err = a.taskSrv.StopTask(ctx, stopTaskReq)
	if err != nil {
		logger.MustGetSessLogger().Error(ctx, err)
		return nil, err
	}

	return &httpproto.StopTaskResp{}, nil
}

func (a *HttpApi) ConfirmTask(ctx context.Context, req *httpproto.ConfirmTaskReq) (*httpproto.ConfirmTaskResp, error) {
	confirmTaskReq := &service.ConfirmTaskReq{}
	err := reflectutil.CopySameFields(req, confirmTaskReq)
	if err != nil {
		logger.MustGetSessLogger().Error(ctx, err)
		return nil, err
	}

	_, err = a.taskSrv.ConfirmTask(ctx, confirmTaskReq)
	if err != nil {
		logger.MustGetSessLogger().Error(ctx, err)
		return nil, err
	}

	return &httpproto.ConfirmTaskResp{}, nil
}

func (a *HttpApi) RegisterTaskCallbackSrv(ctx context.Context, req *httpproto.RegisterTaskCallbackSrvReq) (*httpproto.RegisterTaskCallbackSrvResp, error) {
	registerSrvReq := &service.RegisterTaskCallbackSrvReq{}
	err := reflectutil.CopySameFields(req, registerSrvReq)
	if err != nil {
		logger.MustGetSessLogger().Error(ctx, err)
		return nil, err
	}

	_, err = a.registrySrv.RegisterTaskCallbackSrv(ctx, registerSrvReq)
	if err != nil {
		logger.MustGetSessLogger().Error(ctx, err)
		return nil, err
	}

	return &httpproto.RegisterTaskCallbackSrvResp{}, nil
}

func (a *HttpApi) UnregisterTaskCallbackSrv(ctx context.Context, req *httpproto.UnregisterTaskCallbackSrvReq) (*httpproto.UnregisterTaskCallbackSrvResp, error) {
	unregisterSrvReq := &service.UnregisterTaskCallbackSrvReq{}
	err := reflectutil.CopySameFields(req, unregisterSrvReq)
	if err != nil {
		logger.MustGetSessLogger().Error(ctx, err)
		return nil, err
	}

	_, err = a.registrySrv.UnregisterTaskCallbackSrv(ctx, unregisterSrvReq)
	if err != nil {
		logger.MustGetSessLogger().Error(ctx, err)
		return nil, err
	}

	return &httpproto.UnregisterTaskCallbackSrvResp{}, nil
}