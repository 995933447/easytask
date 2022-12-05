package apihandler

import (
	"context"
	"github.com/995933447/easytask/pkg/rpc/proto/httpproto"
)

func AddTask(ctx context.Context, req *httpproto.AddTaskReq) (*httpproto.AddTaskResp, error) {
	var resp httpproto.AddTaskResp
	return &resp, nil
}