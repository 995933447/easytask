package errs

import (
	"errors"
)

var (
	ErrCurrentNodeNoMaster = errors.New("current node is not master")
	ErrServerStarted = errors.New("server started")
	ErrTaskCallbackServerNotFound = errors.New("server not found")
)