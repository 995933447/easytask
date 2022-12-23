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
	Name string `json:"name" validate:"required"`
	SrvName string 	`json:"srv_name" validate:"required"`
	CallbackPath string `json:"callback_path"`
	SchedMode proto.SchedMode `json:"sched_mode" validate:"required"`
	TimeCron string `json:"time_cron"`
	TimeIntervalSec int `json:"time_interval_sec"`
	TimeSpecAt int64 `json:"time_spec_at"`
	Arg string `json:"arg"`
	BizId string `json:"biz_id"`
}

type AddTaskResp struct {
	TaskId string `json:"task_id"`
}

type StopTaskReq struct {
	TaskId string `json:"task_id" validate:"required"`
}

type StopTaskResp struct {
}

type ConfirmTaskReq struct {
	TaskId string `json:"task_id" validate:"required"`
	IsSuccess bool `json:"is_success"`
	Extra string `json:"extra"`
	TaskRunTimes int `json:"task_run_times" validate:"required"`
}

type ConfirmTaskResp struct {
}

type RegisterTaskCallbackSrvReq struct {
	Name string `json:"name" validate:"required"`
	Schema string `json:"schema" validate:"required"`
	Host string `json:"host" validate:"required"`
	Port int `json:"port" validate:"required"`
	CallbackTimeoutSec int `json:"callback_timeout_sec"`
	IsEnableHealthCheck bool `json:"is_enable_health_check"`
}

type RegisterTaskCallbackSrvResp struct {
}

type UnregisterTaskCallbackSrvReq struct {
	Name string `json:"name" validate:"required"`
	Schema string `json:"schema" validate:"required"`
	Host string `json:"host" validate:"required"`
	Port int `json:"port" validate:"required"`
}

type UnregisterTaskCallbackSrvResp struct {
}