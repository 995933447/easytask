package errs

type ErrCode int

const (
	ErrCodeInternal = 500
	ErrCodeRouteNotFound = 404
	ErrCodeRouteMethodNotAllow = 405
	ErrCodeArgsInvalid = 10001
)

var errMap = map[ErrCode]string{
	ErrCodeInternal: "internal error",
	ErrCodeRouteNotFound: "route not found",
	ErrCodeArgsInvalid: "request arguments are invalid",
}

func GetErrMsg(code ErrCode) string {
	return errMap[code]
}