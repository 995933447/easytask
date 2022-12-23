package httpproto

import (
	"github.com/995933447/easytask/pkg/errs"
	"github.com/995933447/easytask/pkg/rpc/proto"
)

type FinalStdoutResp struct {
	Code errs.ErrCode `json:"code"`
	Msg string `json:"msg"`
	Data any `json:"data"`
	Hint string `json:"hint"`
}

type AddTaskReq struct {
	Name string `json:"name"`
	SrvName string 	`json:"srv_name"`
	CallbackPath string `json:"callback_path"`
	SchedMode proto.SchedMode `json:"sched_mode"`
	TimeCron string `json:"time_cron"`
	TimeIntervalSec int `json:"time_interval_sec"`
	TimeSpecAt int64 `json:"time_spec_at"`
	Arg string `json:"arg"`
	BizId string `json:"biz_id"`
}

type AddTaskResp struct {
	TaskId string `json:"id"`
}

type StopTaskReq struct {
	TaskId string `json:"id"`
}

type StopTaskResp struct {
}

type ConfirmTaskReq struct {
	TaskId string `json:"id"`
	IsSuccess bool `json:"is_success"`
	Extra string `json:"extra"`
	TaskRunTimes int `json:"task_run_times"`
}

type ConfirmTaskResp struct {
}

type RegisterTaskCallbackSrvReq struct {
	Name string `json:"name"`
	Schema string `json:"schema"`
	Host string `json:"host"`
	Port int `json:"port"`
	CallbackTimeoutSec int `json:"callback_timeout_sec"`
	IsEnableHealthCheck bool `json:"is_enable_health_check"`
}

type RegisterTaskCallbackSrvResp struct {
}

type UnregisterTaskCallbackSrvReq struct {
	Name string `json:"name"`
	Schema string `json:"schema"`
	Host string `json:"host"`
	Port int `json:"port"`
}

type UnregisterTaskCallbackSrvResp struct {
}