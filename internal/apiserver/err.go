package apiserver

import (
	"fmt"
	"github.com/995933447/easytask/pkg/errs"
)

type HandleErr struct {
	errCode errs.ErrCode
	err error
}

func NewHandleErr(errCode errs.ErrCode, err error) *HandleErr {
	return &HandleErr{
		errCode: errCode,
		err: err,
	}
}

func (e *HandleErr) GetErrCode() errs.ErrCode {
	return e.errCode
}

func (e *HandleErr) GetErrMsg() string {
	return e.err.Error()
}

func (e *HandleErr) Error() string {
	return fmt.Sprintf("%d: %s", e.errCode, e.err)
}