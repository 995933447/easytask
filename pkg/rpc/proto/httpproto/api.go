package httpproto

type AddTaskReq struct {

}

type AddTaskResp struct {

}

type AddTaskCallbackSrvReq struct {
	Name string `json:"name"`
	Schema string `json:"schema"`
	Host string `json:"host"`
	Port int `json:"port"`
	CallbackTimeoutSec int `json:"callback_timeout_sec"`
	IsEnableHealthCheck bool `json:"is_enable_health_check"`
}

type AddTaskCallbackSrvResp struct {

}