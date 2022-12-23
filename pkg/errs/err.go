package errs

import "fmt"

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

type BizError struct {
	code ErrCode
	msg string
}

func (e *BizError) Error() string {
	return fmt.Sprintf("%d: %s", e.code, e.msg)
}

func NewBizErr(code ErrCode) *BizError {
	return NewBizErrWithMsg(code, GetErrMsg(code))
}

func NewBizErrWithMsg(code ErrCode, msg string) *BizError {
	return &BizError{
		code: code,
		msg: msg,
	}
}