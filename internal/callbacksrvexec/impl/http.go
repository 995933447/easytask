package impl

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/995933447/easytask/internal/callbacksrvexec"
	"github.com/995933447/easytask/internal/task"
	"github.com/995933447/easytask/internal/util/logger"
	"github.com/995933447/easytask/pkg/rpc/proto/httpproto"
	"github.com/995933447/log-go"
	"io"
	"net/http"
	"sync"
	"time"
)

type HttpExec struct {}

func NewHttpExec() *HttpExec {
	return &HttpExec{}
}

var _ callbacksrvexec.TaskCallbackSrvExec = (*HttpExec)(nil)

func (e *HttpExec) CallbackSrv(ctx context.Context, task *task.Task, _ any) (*callbacksrvexec.TaskCallbackSrvResp, error) {
	var (
		httpReq = &httpproto.TaskCallbackReq{
			Cmd: httpproto.HttpCallbackCmdTaskCallback,
			Arg:      task.GetArg(),
			TaskName: task.GetName(),
			RunTimes: task.GetRunTimes(),
		}
		httpResp = &httpproto.TaskCallbackResp{}
	)

	httpReqBytes, err := json.Marshal(httpReq)
	if err != nil {
		logger.MustGetTaskProcLogger().Error(ctx, err)
		return nil, err
	}

	var (
		route = task.GetCallbackSrv().GetRandomRoute()
		timeoutSec int
	)
	if route.GetCallbackTimeoutSec() > task.GetMaxRunTimeSec() {
		timeoutSec = task.GetMaxRunTimeSec()
	} else {
		timeoutSec = route.GetCallbackTimeoutSec()
	}
	if err := e.doReq(ctx, logger.MustGetTaskProcLogger(), route, timeoutSec, httpReqBytes, httpResp); err  != nil {
		logger.MustGetTaskProcLogger().Error(ctx, err)
		return nil, err
	}

	return callbacksrvexec.NewCallbackSrvResp(httpResp.IsRunInAsync, httpResp.IsSuccess, httpResp.Extra), nil
}

func (e *HttpExec) HeartBeat(ctx context.Context, srv *task.TaskCallbackSrv) (*callbacksrvexec.HeartBeatResp, error) {
	var (
		httpReq = &httpproto.HeartBeatReq{
			Cmd: httpproto.HttpCallbackCmdTaskHeartBeat,
		}
		httpResp = &httpproto.HeartBeatResp{}
	)

	httpReqBytes, err := json.Marshal(httpReq)
	if err != nil {
		logger.MustGetMainProcLogger().Error(ctx, err)
		return nil, err
	}

	var (
		wg sync.WaitGroup
		noReplyRoutes []*task.TaskCallbackSrvRoute
		replyRoutes []*task.TaskCallbackSrvRoute
	)
	for _, route := range srv.GetRoutes() {
		wg.Add(1)
		go func(route *task.TaskCallbackSrvRoute) {
			defer wg.Done()
			if err := e.doReq(ctx, logger.MustGetMainProcLogger(), route, route.GetCallbackTimeoutSec(), httpReqBytes, httpResp); err != nil {
				noReplyRoutes = append(noReplyRoutes, route)
				return
			}

			if !httpResp.Pong {
				noReplyRoutes = append(noReplyRoutes, route)
				return
			}

			replyRoutes = append(replyRoutes, route)
		}(route)
	}
	wg.Wait()

	return callbacksrvexec.NewHeartBeatResp(replyRoutes, noReplyRoutes), nil
}

func (e *HttpExec) doReq(ctx context.Context, logger *log.Logger, route *task.TaskCallbackSrvRoute, timeoutSec int, reqBytes []byte, resp interface{}) error {
	httpReq, err := http.NewRequest(
		http.MethodPost,
		fmt.Sprintf("%s://%s:%d", route.GetSchema(), route.GetHost(), route.GetPort()),
		bytes.NewBuffer(reqBytes),
	)
	if err != nil {
		logger.Error(ctx, err)
		return err
	}

	httpCli := http.Client{}

	if timeoutSec > 0 {
		httpCli.Timeout = time.Duration(timeoutSec) * time.Second
	}

	httpResp, err := httpCli.Do(httpReq)
	if err != nil {
		logger.Error(ctx, err)
		return err
	}

	httpRespBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		logger.Error(ctx, err)
		return err
	}

	err = json.Unmarshal(httpRespBody, &resp)
	if err != nil {
		logger.Error(ctx, err)
		return err
	}

	return nil
}
