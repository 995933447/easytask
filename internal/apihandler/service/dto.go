package service

import (
	"github.com/995933447/easytask/internal/task"
)

type AddTaskReq struct {
	Name string
	SrvName string
	CallbackPath string
	SchedMode task.SchedMode
	TimeCron string
	TimeIntervalSec int
	TimeSpecAt int64
	Arg string
	BizId string
}

type AddTaskResp struct {
	Id string
}

type StopTaskReq struct {
	Id string
}

type StopTaskResp struct {
}

type ConfirmTaskReq struct {
	TaskId string
	IsSuccess bool
	Extra string
	TaskRunTimes int
}

type ConfirmTaskResp struct {
}

type RegisterTaskCallbackSrvReq struct {
	Name string
	Schema string
	Host string
	Port int
	CallbackTimeoutSec int
	IsEnableHealthCheck bool
}

type RegisterTaskCallbackSrvResp struct {
}

type UnregisterTaskCallbackSrvReq struct {
	Name string
	Schema string
	Host string
	Port int
}

type UnregisterTaskCallbackSrvResp struct {
}


