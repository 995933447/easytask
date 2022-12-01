package task

import (
	"context"
	"github.com/995933447/easytask/internal/callbacksrvexec"
	"github.com/995933447/easytask/internal/sched"
	"time"
)

const (
	DefaultWorkerPoolSize = 100
)

type workerEngine struct {
	workerPoolSize      uint
	sched               *sched.Sched
	callbackTaskSrvExec callbacksrvexec.TaskCallbackSrvExec
}

func NewWorkerEngine(workerPoolSize uint, sched *sched.Sched, callbackTaskSrvExec callbacksrvexec.TaskCallbackSrvExec) *workerEngine {
	if workerPoolSize <= 0 {
		workerPoolSize = DefaultWorkerPoolSize
	}

	return &workerEngine{
		workerPoolSize: workerPoolSize,
		sched: sched,
		callbackTaskSrvExec: callbackTaskSrvExec,
	}
}

func (e *workerEngine) Run(ctx context.Context) {
	e.createWorkerPool(ctx)
	e.sched.Run(ctx)
}

func (e *workerEngine) createWorkerPool(ctx context.Context) {
	var i uint
	for ; i < e.workerPoolSize; i++ {
		taskIn := make(chan *Task)
		go e.runWorker(ctx, taskIn)
	}
}

func (e *workerEngine) runWorker(ctx context.Context, taskIn chan *Task) {
	for {
		e.sched.TaskWorkerReady(taskIn)
		task := <-taskIn
		locked, err := e.sched.LockTaskForRun(ctx, task)
		if err != nil {
			return
		}
		if !locked {
			return
		}
		taskResp, err := task.run(ctx, e.callbackTaskSrvExec)
		if err != nil {
			taskResp = newInternalErrTaskResp(task.id, err, time.Now().Unix())
		}
		e.sched.SubmitTaskResp(taskResp)
	}
}
