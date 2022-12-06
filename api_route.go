package easytask

import (
	"github.com/995933447/easytask/internal/apiserver"
	"github.com/995933447/easytask/internal/repo"
	"github.com/995933447/easytask/pkg/api"
	"net/http"
)

func getApiRoutes(taskRepo repo.TaskRepo, taskCallbackSrvRepo repo.TaskCallbackSrvRepo) []*apiserver.Route {
	return []*apiserver.Route{
		{Path: api.AddTaskCmdPath, Method: http.MethodPost, Handler: apiserver.AddTaskHandler(taskRepo, taskCallbackSrvRepo)},
		{Path: api.RegisterTaskCallbackSrvCmdPath, Method: http.MethodPost, Handler: apiserver.RegisterTaskCallbackSrvHandler(taskCallbackSrvRepo)},
		{Path: api.DelTaskCmdPath, Method: http.MethodPost, Handler: apiserver.DelTaskHandler(taskRepo)},
		{Path: api.UnregisterTaskCallbackSrvCmdPath, Method: http.MethodPost, Handler: apiserver.UnregisterTaskCallbackSrvHandler(taskCallbackSrvRepo)},
	}
}

