package errs

type ErrCode int

const (
	ErrCodeInternal = 500
	ErrCodeRouteNotFound = 404
	ErrCodeRouteMethodNotAllow = 405
	ErrCodeArgsInvalid = 10001
	ErrCodeTaskNotFound = 10002
	ErrCodeTaskCallbackSrvNotFound = 10003
)

var errMap = map[ErrCode]string{
	ErrCodeInternal: "internal error",
	ErrCodeRouteNotFound: "route not found",
	ErrCodeArgsInvalid: "request arguments are invalid",
	ErrCodeTaskNotFound: "task not found",
	ErrCodeTaskCallbackSrvNotFound: "task callback server not found",
}

func GetErrMsg(code ErrCode) string {
	return errMap[code]
}