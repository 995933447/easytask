package httpproto

const (
	HttpCallbackCmdTaskCallback = iota
	HttpCallbackCmdTaskHeartBeat
)

type TaskCallbackResp struct {
	IsRunInAsync bool `json:"is_run_in_async"`
	IsSuccess bool `json:"is_success"`
	Extra string `json:"extra"`
}

type TaskCallbackReq struct {
	Cmd int `json:"cmd"`
	TaskId string `json:"id"`
	TaskName string `json:"task_name"`
	Arg      string `json:"arg"`
	RunTimes int    `json:"run_times"`
}

type HeartBeatResp struct {
	Pong bool `json:"pong"`
}

type HeartBeatReq struct {
	Cmd int `json:"cmd"`
}