package httpproto

import "github.com/995933447/easytask/internal/task"

type AddTaskReq struct {
	Name string `json:"name"`
	SrvName string 	`json:"srv_name"`
	CallbackPath string `json:"callback_path"`
	SchedMode task.SchedMode `json:"sched_mode"`
	TimeCron string `json:"time_cron"`
	TimeIntervalSec string `json:"time_interval_sec"`
	TimeSpec string `json:"time_spec"`
	Arg []byte `json:"arg"`
}

type AddTaskResp struct {
	Id string `json:"id"`
}

type DelTaskReq struct {
	Id string `json:"id"`
}

type DelTaskResp struct {
}

type ConfirmTaskReq struct {
	Id string `json:"id"`
	IsSuccess bool `json:"is_success"`
	Extra []byte `json:"extra"`
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