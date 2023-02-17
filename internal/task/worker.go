package task

import (
	"context"
	"github.com/995933447/easytask/internal/util/logger"
	"github.com/995933447/easytask/pkg/contxt"
	"github.com/995933447/simpletrace"
	simpletracectx "github.com/995933447/simpletrace/context"
	"sync"
	"sync/atomic"
	"time"
)

const (
	DefaultWorkerPoolSize = 100
)

type WorkerEngine struct {
	workerPoolSize      uint
	sched               *Sched
	callbackTaskSrvExec TaskCallbackSrvExec
	isPaused            atomic.Bool
	exitWorkerWait      sync.WaitGroup
}

func NewWorkerEngine(workerPoolSize uint, sched *Sched, callbackTaskSrvExec TaskCallbackSrvExec) *WorkerEngine {
	if workerPoolSize <= 0 {
		workerPoolSize = DefaultWorkerPoolSize
	}

	return &WorkerEngine{
		workerPoolSize: workerPoolSize,
		sched: sched,
		callbackTaskSrvExec: callbackTaskSrvExec,
	}
}

func (e *WorkerEngine) Run(ctx context.Context) {
	e.createWorkerPool(ctx)
	e.sched.run(ctx)
}

func (e *WorkerEngine) Stop() {
	e.sched.stop()
	e.isPaused.Store(true)
	e.exitWorkerWait.Wait()
}

func (e *WorkerEngine) createWorkerPool(ctx context.Context) {
	logger.MustGetSysLogger().Info(ctx, "start create worker pool")
	var i uint
	for ; i < e.workerPoolSize; i++ {
		go e.runWorker(ctx, i)
		e.exitWorkerWait.Add(1)
		logger.MustGetSysLogger().Infof(ctx, "task worker(id:%d) running", i)
	}
	logger.MustGetSysLogger().Info(ctx, "finish creating worker pool")
}

func (e *WorkerEngine) runWorker(ctx context.Context, workerId uint) {
	var (
		traceModule = "task_worker"
		origCtxTraceId string
	)
	if traceCtx, ok := ctx.(*simpletracectx.Context); ok {
		origCtxTraceId = traceCtx.GetTraceId()
	}

	checkPausedTk := time.NewTicker(time.Second * 2)
	for {
		ctx = contxt.NewWithTrace(traceModule, context.TODO(), traceModule + "_" + origCtxTraceId + "." + simpletrace.NewTraceId(), "")

		logger.MustGetSysLogger().Infof(ctx, "worker(id:%d) is ready", workerId)

		var (
			task *Task
			isPaused bool
		)
		select {
		case task = <- e.sched.taskCh:
		case <- checkPausedTk.C:
			if isPaused = e.isPaused.Load(); isPaused {
				e.exitWorkerWait.Done()
				logger.MustGetSysLogger().Debugf(ctx, "worker(id:%d) exited", workerId)
			}
		}

		if isPaused {
			break
		}

		if task == nil {
			continue
		}

		if len(task.callbackSrv.GetRoutes()) == 0 {
			logger.MustGetSysLogger().Warnf(
				ctx,
				"task(id:%s, name:%s) callback server(name:%s) has no routes",
				task.id, task.name, task.callbackSrv.name,
			)
			return
		}

		logger.MustGetSysLogger().Infof(ctx, "worker(id:%d) run task(id:%s name:%s)", workerId, task.id, task.name)

		now := time.Now()

		locked, err := e.sched.lockTaskForRun(ctx, task)
		if err != nil {
			logger.MustGetSysLogger().Error(ctx, err)
			continue
		}

		if !locked {
			logger.MustGetSysLogger().Warnf(ctx, "worker(id:%d) lock task(id:%s) failed", workerId, task.id)
			continue
		}

		taskResp, err := task.run(ctx, e.callbackTaskSrvExec)
		if err != nil {
			logger.MustGetSysLogger().Error(ctx, err)
			taskResp, err = newInternalErrTaskResp(task.id, err, now.Unix())
			if err != nil {
				logger.MustGetSysLogger().Error(ctx, err)
				continue
			}
		}

		err = e.sched.submitTaskResp(ctx, taskResp)
		if err != nil {
			logger.MustGetSysLogger().Error(ctx, err)
		}

		logger.MustGetSysLogger().Infof(
			ctx,
			"worker(id:%d) finish task(id:%s name:%s), exec time:%d ms",
			workerId, task.id, task.name, time.Now().Sub(now) / time.Millisecond,
		)
	}
}
