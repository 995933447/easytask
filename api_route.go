package easytask

import (
	"github.com/995933447/easytask/internal/apiserver"
	"github.com/995933447/easytask/internal/registry"
	"github.com/995933447/easytask/internal/repo"
	"github.com/995933447/easytask/pkg/api"
	"net/http"
)

func getHttpApiRoutes(taskRepo repo.TaskRepo, reg *registry.Registry) []*apiserver.HttpRoute {
	return []*apiserver.HttpRoute{
		{Path: api.AddTaskCmdPath, Method: http.MethodPost, Handler: apiserver.AddTaskHandler(taskRepo, reg)},
		{Path: api.RegisterTaskCallbackSrvCmdPath, Method: http.MethodPost, Handler: apiserver.RegisterTaskCallbackSrvHandler(reg)},
		{Path: api.DelTaskCmdPath, Method: http.MethodPost, Handler: apiserver.DelTaskHandler(taskRepo)},
		{Path: api.UnregisterTaskCallbackSrvCmdPath, Method: http.MethodPost, Handler: apiserver.UnregisterTaskCallbackSrvHandler(reg)},
	}
}

