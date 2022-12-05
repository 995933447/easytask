package easytask

import (
	"github.com/995933447/easytask/internal/apihandler"
	"net/http"
)

var apiRoutes = []*apihandler.Route{
	{Path: "/task", Method: http.MethodPost, Handler: apihandler.AddTask},
}
