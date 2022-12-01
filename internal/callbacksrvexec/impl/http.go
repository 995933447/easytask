package impl

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/995933447/easytask/internal/callbacksrvexec"
	"github.com/995933447/easytask/internal/task"
	"github.com/995933447/easytask/pkg/rpc/proto/httpproto"
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
			TriedCnt: task.GetTriedCnt(),
		}
		httpResp = &httpproto.TaskCallbackResp{}
	)

	httpReqBytes, err := json.Marshal(httpReq)
	if err != nil {
		return nil, err
	}

	if err := e.doReq(ctx, task.GetCallbackSrv().GetRandomRoute(), httpReqBytes, httpResp); err  != nil {
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
			if err := e.doReq(ctx, route, httpReqBytes, httpResp); err != nil {
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

func (e *HttpExec) doReq(_ context.Context, callbackSrvRoute *task.TaskCallbackSrvRoute, reqBytes []byte, resp interface{}) error {
	httpReq, err := http.NewRequest(
		http.MethodPost,
		fmt.Sprintf("%s://%s:%d", callbackSrvRoute.GetSchema(), callbackSrvRoute.GetHost(), callbackSrvRoute.GetPort()),
		bytes.NewBuffer(reqBytes),
	)
	if err != nil {
		return err
	}

	httpCli := http.Client{}

	if callbackSrvRoute.GetCallbackTimeoutSec() > 0 {
		httpCli.Timeout = time.Duration(callbackSrvRoute.GetCallbackTimeoutSec()) * time.Second
	}

	httpResp, err := httpCli.Do(httpReq)
	if err != nil {
		return err
	}

	httpRespBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return err
	}

	err = json.Unmarshal(httpRespBody, &resp)
	if err != nil {
		return err
	}

	return nil
}
