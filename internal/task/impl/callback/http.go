package callback

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/995933447/easytask/internal/task"
	"github.com/995933447/easytask/internal/util/logger"
	"github.com/995933447/easytask/pkg/contxt"
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

var _ task.TaskCallbackSrvExec = (*HttpExec)(nil)

func (e *HttpExec) CallbackSrv(ctx context.Context, oneTask *task.Task, _ any) (*task.TaskCallbackSrvResp, error) {
	var (
		log = logger.MustGetCallbackLogger()
		httpReq = &httpproto.TaskCallbackReq{
			Cmd: 	  httpproto.HttpCallbackCmdTaskCallback,
			TaskId:   oneTask.GetId(),
			Arg:      oneTask.GetArg(),
			TaskName: oneTask.GetName(),
			RunTimes: oneTask.GetRunTimes(),
		}
		httpResp = &httpproto.TaskCallbackResp{}
	)

	httpReqBytes, err := json.Marshal(httpReq)
	if err != nil {
		log.Error(ctx, err)
		return nil, err
	}

	var (
		route = oneTask.GetCallbackSrv().GetRandomRoute()
		timeoutSec int
	)
	if route.GetCallbackTimeoutSec() > oneTask.GetMaxRunTimeSec() {
		timeoutSec = oneTask.GetMaxRunTimeSec()
	} else {
		timeoutSec = route.GetCallbackTimeoutSec()
	}
	err = e.doReq(contxt.ChildOf(ctx), &doReqInput{
		Path: oneTask.GetCallbackPath(),
		Route: route,
		TimeoutSec: timeoutSec,
		ReqBytes: httpReqBytes,
		Resp: httpResp,
	})
	if err != nil {
		log.Error(ctx, err)
		return nil, err
	}

	return task.NewCallbackSrvResp(httpResp.IsRunInAsync, httpResp.IsSuccess, httpResp.Extra), nil
}

func (e *HttpExec) HeartBeat(ctx context.Context, srv *task.TaskCallbackSrv) (*task.HeartBeatResp, error) {
	var (
		log = logger.MustGetCallbackLogger()
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

			err = e.doReq(contxt.ChildOf(ctx), &doReqInput{
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

	return task.NewHeartBeatResp(replyRoutes, noReplyRoutes), nil
}

type doReqInput struct {
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
		i.Path = "/" + strings.TrimLeft(i.Path, "/")
	}
	return nil
}

func (e *HttpExec) doReq(ctx context.Context, input *doReqInput) error {
	log := logger.MustGetCallbackLogger()

	if err := input.Check(); err != nil {
		log.Error(ctx, err)
		return err
	}

	reqUrl := fmt.Sprintf("%s://%s:%d%s", input.Route.GetSchema(), input.Route.GetHost(), input.Route.GetPort(), input.Path)
	httpReq, err := http.NewRequest(
		http.MethodPost,
		reqUrl,
		bytes.NewBuffer(input.ReqBytes),
	)
	if err != nil {
		log.Error(ctx, err)
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

	log.Infof(ctx, "post:%s param:%s", reqUrl, string(input.ReqBytes))

	httpResp, err := httpCli.Do(httpReq)
	if err != nil {
		log.Error(ctx, err)
		return err
	}

	httpRespBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		log.Error(ctx, err)
		return err
	}

	log.Infof(ctx, "resp:%s", string(httpRespBody))

	err = json.Unmarshal(httpRespBody, &input.Resp)
	if err != nil {
		log.Error(ctx, err)
		return err
	}

	return nil
}
