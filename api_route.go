package main

import (
	"github.com/995933447/easytask/internal/apihandler"
	"github.com/995933447/easytask/internal/apihandler/service"
	"github.com/995933447/easytask/internal/apiserver"
	"github.com/995933447/easytask/internal/registry"
	"github.com/995933447/easytask/internal/task"
	"github.com/995933447/easytask/pkg/rpc/proto/httpproto"
	"net/http"
)

func getHttpApiRoutes(taskRepo task.TaskRepo, reg *registry.Registry) []*apiserver.HttpRoute {
	httpApi := apihandler.NewHttpApi(service.NewTaskService(taskRepo, reg), service.NewRegistryService(reg))
	return []*apiserver.HttpRoute{
		{Path: httpproto.AddTaskCmdPath, Method: http.MethodPost, Handler: httpApi.AddTask},
		{Path: httpproto.StopTaskCmdPath, Method: http.MethodPost, Handler: httpApi.StopTask},
		{Path: httpproto.ConfirmTaskCmdPath, Method: http.MethodPost, Handler: httpApi.ConfirmTask},
		{Path: httpproto.RegisterTaskCallbackSrvCmdPath, Method: http.MethodPost, Handler: httpApi.RegisterTaskCallbackSrv},
		{Path: httpproto.UnregisterTaskCallbackSrvCmdPath, Method: http.MethodPost, Handler: httpApi.UnregisterTaskCallbackSrv},
	}
}

