package task

import (
	"context"
	"github.com/995933447/easytask/internal/util/logger"
	"github.com/995933447/easytask/pkg/contxt"
	"time"
)

const (
	DefaultWorkerPoolSize = 100
)

type workerEngine struct {
	workerPoolSize      uint
	sched               *Sched
	callbackTaskSrvExec TaskCallbackSrvExec
}

func NewWorkerEngine(workerPoolSize uint, sched *Sched, callbackTaskSrvExec TaskCallbackSrvExec) *workerEngine {
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
		go e.runWorker(ctx, taskIn, i)
	}
}

func (e *workerEngine) runWorker(ctx context.Context, taskIn chan *Task, workerId uint) {
	log := logger.MustGetTaskLogger()
	for {
		e.sched.TaskWorkerReady(taskIn)
		ctx = contxt.ChildOf(ctx)
		log.Infof(ctx, "worker:%d is ready", workerId)
		task := <- taskIn
		locked, err := e.sched.LockTaskForRun(contxt.ChildOf(ctx), task)
		if err != nil {
			log.Error(ctx, err)
			return
		}
		if !locked {
			log.Warnf(ctx, "lock task(id:%s) failed", task.id)
			return
		}
		taskResp, err := task.run(contxt.ChildOf(ctx), e.callbackTaskSrvExec)
		if err != nil {
			log.Error(ctx, err)
			taskResp = newInternalErrTaskResp(task.id, err, time.Now().Unix())
		}
		e.sched.SubmitTaskResp(taskResp)
	}
}
