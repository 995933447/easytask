package errs

import (
	"errors"
	"sync"
)

var (
	ErrCurrentNodeNoMaster = errors.New("current node is not master")
	ErrServerStarted = errors.New("server started")
)

type AtomicErr struct {
	mu sync.RWMutex
	err error
}

func (e *AtomicErr) SetErr(err error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.err = err
}

func (e *AtomicErr) GetErr() error {
	e.mu.RLock()
	defer e.mu.Unlock()
	return e.err
}