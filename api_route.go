package easytask

import (
	"github.com/995933447/easytask/internal/apihandler"
	"github.com/995933447/easytask/internal/apiserver"
	"github.com/995933447/easytask/internal/registry"
	"github.com/995933447/easytask/internal/repo"
	"github.com/995933447/easytask/pkg/api"
	"net/http"
)

func getHttpApiRoutes(taskRepo repo.TaskRepo, reg *registry.Registry) []*apiserver.HttpRoute {
	var (
		taskSrv = apihandler.NewTaskService(taskRepo, reg)
		registrySrv = apihandler.NewRegistryService(reg)
	)
	return []*apiserver.HttpRoute{
		{Path: api.AddTaskCmdPath, Method: http.MethodPost, Handler: taskSrv.AddTask},
		{Path: api.DelTaskCmdPath, Method: http.MethodPost, Handler: taskSrv.DelTask},
		{Path: api.RegisterTaskCallbackSrvCmdPath, Method: http.MethodPost, Handler: registrySrv.RegisterTaskCallbackSrv},
		{Path: api.UnregisterTaskCallbackSrvCmdPath, Method: http.MethodPost, Handler: registrySrv.UnregisterTaskCallbackSrv},
	}
}

