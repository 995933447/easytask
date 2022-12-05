package task

import (
	"context"
	"github.com/995933447/easytask/internal/callbacksrvexec"
	"github.com/995933447/easytask/internal/sched"
	"github.com/995933447/easytask/internal/util/logger"
	"github.com/995933447/easytask/pkg/contx"
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
	e.createWorkerPool()
	e.sched.Run(ctx)
}

func (e *workerEngine) createWorkerPool() {
	var i uint
	for ; i < e.workerPoolSize; i++ {
		taskIn := make(chan *Task)
		go e.runWorker(taskIn)
	}
}

func (e *workerEngine) runWorker(taskIn chan *Task) {
	for {
		e.sched.TaskWorkerReady(taskIn)
		task := <-taskIn
		ctx := contx.New("task", context.TODO())
		log := logger.MustGetSysProcLogger()
		locked, err := e.sched.LockTaskForRun(ctx, task)
		if err != nil {
			log.Error(ctx, err)
			return
		}
		if !locked {
			log.Warnf(ctx, "lock task(id:%s) failed", task.id)
			return
		}
		taskResp, err := task.run(ctx, e.callbackTaskSrvExec)
		if err != nil {
			log.Error(ctx, err)
			taskResp = newInternalErrTaskResp(task.id, err, time.Now().Unix())
		}
		e.sched.SubmitTaskResp(taskResp)
	}
}
