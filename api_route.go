package main

import (
	"github.com/995933447/easytask/internal/apihandler"
	"github.com/995933447/easytask/internal/apiserver"
	"github.com/995933447/easytask/internal/registry"
	"github.com/995933447/easytask/internal/task"
	"github.com/995933447/easytask/pkg/rpc/proto/httpproto"
	"net/http"
)

func getHttpApiRoutes(taskRepo task.TaskRepo, reg *registry.Registry) []*apiserver.HttpRoute {
	var (
		taskSrv = apihandler.NewTaskService(taskRepo, reg)
		registrySrv = apihandler.NewRegistryService(reg)
	)
	return []*apiserver.HttpRoute{
		{Path: httpproto.AddTaskCmdPath, Method: http.MethodPost, Handler: taskSrv.AddTask},
		{Path: httpproto.StopTaskCmdPath, Method: http.MethodPost, Handler: taskSrv.StopTask},
		{Path: httpproto.ConfirmTaskCmdPath, Method: http.MethodPost, Handler: taskSrv.ConfirmTask},
		{Path: httpproto.RegisterTaskCallbackSrvCmdPath, Method: http.MethodPost, Handler: registrySrv.RegisterTaskCallbackSrv},
		{Path: httpproto.UnregisterTaskCallbackSrvCmdPath, Method: http.MethodPost, Handler: registrySrv.UnregisterTaskCallbackSrv},
	}
}

