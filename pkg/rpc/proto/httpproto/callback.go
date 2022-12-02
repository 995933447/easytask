package httpproto

const (
	HttpCallbackCmdTaskCallback = iota
	HttpCallbackCmdTaskHeartBeat
)

type TaskCallbackResp struct {
	IsRunInAsync bool `json:"is_run_in_async"`
	IsSuccess bool `json:"is_success"`
	Extra []byte `json:"extra"`
}

type TaskCallbackReq struct {
	Cmd int `json:"cmd"`
	TaskName string `json:"task_name"`
	Arg      []byte `json:"arg"`
	RunTimes int    `json:"tried_cnt"`
}

type HeartBeatResp struct {
	Pong bool `json:"pong"`
}

type HeartBeatReq struct {
	Cmd int `json:"cmd"`
}