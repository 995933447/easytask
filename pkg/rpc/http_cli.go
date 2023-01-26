package rpc

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/995933447/easytask/pkg/contxt"
	"github.com/995933447/easytask/pkg/errs"
	"github.com/995933447/easytask/pkg/rpc/proto/httpproto"
	simpletracectx "github.com/995933447/simpletrace/context"
	"io"
	"net/http"
	"strings"
	"time"
)

type HttpReqOpt func(req *http.Request, client *http.Client) error

func WithTimeoutHttpOpt(timeout time.Duration) HttpReqOpt {
	return func(_ *http.Request, client *http.Client) error {
		client.Timeout = timeout
		return nil
	}
}

type HttpCli struct {
	apiSrvAddr string
}

func NewHttpCli(apiSrvAddr string) *HttpCli {
	return &HttpCli{
		apiSrvAddr: strings.TrimRight(apiSrvAddr, "/"),
	}
}

func (c *HttpCli) AddTask(ctx context.Context, req *httpproto.AddTaskReq, opts ...HttpReqOpt) (*httpproto.AddTaskResp, error) {
	var resp httpproto.AddTaskResp
	err := c.post(contxt.New("api", ctx), httpproto.AddTaskCmdPath, req, &resp, opts...)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *HttpCli) StopTask(ctx context.Context, req *httpproto.StopTaskReq, opts ...HttpReqOpt) (*httpproto.StopTaskResp, error) {
	var resp httpproto.StopTaskResp
	err := c.post(contxt.New("api", ctx), httpproto.StopTaskCmdPath, req, &resp, opts...)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *HttpCli) ConfirmTask(ctx context.Context, req *httpproto.ConfirmTaskReq, opts ...HttpReqOpt) (*httpproto.ConfirmTaskResp, error) {
	var resp httpproto.ConfirmTaskResp
	err := c.post(contxt.New("api", ctx), httpproto.ConfirmTaskCmdPath, req, &resp, opts...)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *HttpCli) RegisterTaskCallbackSrv(ctx context.Context, req *httpproto.RegisterTaskCallbackSrvReq, opts ...HttpReqOpt) (*httpproto.RegisterTaskCallbackSrvResp, error) {
	var resp httpproto.RegisterTaskCallbackSrvResp
	err := c.post(contxt.New("api", ctx), httpproto.RegisterTaskCallbackSrvCmdPath, req, &resp, opts...)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *HttpCli) UnregisterTaskCallbackSrv(ctx context.Context, req *httpproto.UnregisterTaskCallbackSrvReq, opts ...HttpReqOpt) (*httpproto.UnregisterTaskCallbackSrvResp, error) {
	var resp httpproto.UnregisterTaskCallbackSrvResp
	err := c.post(contxt.New("api", ctx), httpproto.UnregisterTaskCallbackSrvCmdPath, req, &resp, opts...)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *HttpCli) post(ctx context.Context, path string, req, resp any, opts ...HttpReqOpt) error {
	httpReqBody, err := json.Marshal(req)
	if err != nil {
		return err
	}

	httpReq, err := http.NewRequest(http.MethodPost, c.apiSrvAddr + path, bytes.NewBuffer(httpReqBody))
	if err != nil {
		return err
	}

	if traceCtx, ok := ctx.(*simpletracectx.Context); ok {
		httpReq.Header.Add(httpproto.HeaderSimpleTraceId, traceCtx.GetTraceId())
		httpReq.Header.Add(httpproto.HeaderSimpleTraceSpanId, traceCtx.GetSpanId())
		httpReq.Header.Add(httpproto.HeaderSimpleTraceParentSpanId, traceCtx.GetParentSpanId())
	}

	httpCli := http.Client{}

	for _, opt := range opts {
		if err = opt(httpReq, &httpCli); err != nil {
			return err
		}
	}

	httpResp, err := httpCli.Do(httpReq)
	if err != nil {
		return err
	}

	httpRespBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return err
	}

	var formattedResp httpproto.FinalStdoutResp
	formattedResp.Data = resp
	err = json.Unmarshal(httpRespBody, &formattedResp)
	if err != nil {
		return err
	}

	if formattedResp.Code > 0 {
		return errs.NewBizErrWithMsg(formattedResp.Code, formattedResp.Msg)
	}

	return nil
}