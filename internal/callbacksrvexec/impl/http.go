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
	simpletracectx "github.com/995933447/simpletrace/context"
	"github.com/go-playground/validator"
	"io"
	"net/http"
	"strings"
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
		log = logger.MustGetSysLogger()
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
		log.Error(ctx, err)
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
	err = e.doReq(ctx, &doReqInput{
		Log: log,
		Path: task.GetCallbackPath(),
		Route: route,
		TimeoutSec: timeoutSec,
		ReqBytes: httpReqBytes,
		Resp: httpResp,
	})
	if err != nil {
		log.Error(ctx, err)
		return nil, err
	}

	return callbacksrvexec.NewCallbackSrvResp(httpResp.IsRunInAsync, httpResp.IsSuccess, httpResp.Extra), nil
}

func (e *HttpExec) HeartBeat(ctx context.Context, srv *task.TaskCallbackSrv) (*callbacksrvexec.HeartBeatResp, error) {
	var (
		log = logger.MustGetSysLogger()
		httpReq = &httpproto.HeartBeatReq{
			Cmd: httpproto.HttpCallbackCmdTaskHeartBeat,
		}
		httpResp = &httpproto.HeartBeatResp{}
	)

	httpReqBytes, err := json.Marshal(httpReq)
	if err != nil {
		log.Error(ctx, err)
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

			err = e.doReq(ctx, &doReqInput{
				Log: log,
				Route: route,
				TimeoutSec: route.GetCallbackTimeoutSec(),
				ReqBytes: httpReqBytes,
				Resp: httpResp,
			})
			if err != nil {
				log.Error(ctx, err)
				noReplyRoutes = append(noReplyRoutes, route)
				return
			}

			if !httpResp.Pong {
				log.Warnf(ctx, "route(id:%s) heart beat resp.Pong is false", route.GetId())
				noReplyRoutes = append(noReplyRoutes, route)
				return
			}

			replyRoutes = append(replyRoutes, route)
		}(route)
	}
	wg.Wait()

	return callbacksrvexec.NewHeartBeatResp(replyRoutes, noReplyRoutes), nil
}

type doReqInput struct {
	Log logger.Logger `validate:"required"`
	Path string
	Route *task.TaskCallbackSrvRoute `validate:"required"`
	TimeoutSec int
	ReqBytes []byte `validate:"required"`
	Resp interface{} `validate:"required"`
}

func (i *doReqInput) Check() error {
	if err := validator.New().Struct(i); err != nil {
		return err
	}
	i.Path = strings.TrimSpace(i.Path)
	if i.Path != "" {
		i.Path += "/"
	}
	return nil
}

func (e *HttpExec) doReq(ctx context.Context, input *doReqInput) error {
	if err := input.Check(); err != nil {
		return err
	}

	httpReq, err := http.NewRequest(
		http.MethodPost,
		fmt.Sprintf("%s://%s:%d%s", input.Route.GetSchema(), input.Route.GetHost(), input.Route.GetPort(), input.Path),
		bytes.NewBuffer(input.ReqBytes),
	)
	if err != nil {
		input.Log.Error(ctx, err)
		return err
	}

	httpCli := http.Client{}

	if input.TimeoutSec > 0 {
		httpCli.Timeout = time.Duration(input.TimeoutSec) * time.Second
	}

	if traceCtx, ok := ctx.(*simpletracectx.Context); ok {
		httpReq.Header.Add(httpproto.HeaderSimpleTraceId, traceCtx.GetTraceId())
		httpReq.Header.Add(httpproto.HeaderSimpleTraceSpanId, traceCtx.GetSpanId())
		httpReq.Header.Add(httpproto.HeaderSimpleTraceParentSpanId, traceCtx.GetParentSpanId())
	}

	httpResp, err := httpCli.Do(httpReq)
	if err != nil {
		input.Log.Error(ctx, err)
		return err
	}

	httpRespBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		input.Log.Error(ctx, err)
		return err
	}

	err = json.Unmarshal(httpRespBody, &input.Resp)
	if err != nil {
		input.Log.Error(ctx, err)
		return err
	}

	return nil
}
