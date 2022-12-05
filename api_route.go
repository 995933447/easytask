package easytask

import (
	"github.com/995933447/easytask/internal/apihandler"
	"github.com/995933447/easytask/internal/repo"
	"net/http"
)

func getApiRoutes(taskRepo repo.TaskRepo, taskCallbackSrvRepo repo.TaskCallbackSrvRepo) []*apihandler.Route {
	return []*apihandler.Route{
		{Path: "/task", Method: http.MethodPost, Handler: apihandler.AddTaskHandler(taskRepo)},
		{Path: "/task/server", Method: http.MethodPost, Handler: apihandler.AddTaskCallbackSrvHandler(taskCallbackSrvRepo)},
	}
}

