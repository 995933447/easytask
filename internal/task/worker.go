package task

import (
	"context"
	"github.com/995933447/easytask/internal/util/logger"
	"github.com/995933447/easytask/pkg/contxt"
	"github.com/995933447/simpletrace"
	simpletracectx "github.com/995933447/simpletrace/context"
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
	e.sched.run(ctx)
}

func (e *workerEngine) createWorkerPool(ctx context.Context) {
	logger.MustGetSysLogger().Info(ctx, "start create worker pool")
	var i uint
	for ; i < e.workerPoolSize; i++ {
		go e.runWorker(ctx, i)
		logger.MustGetSysLogger().Infof(ctx, "task worker(id:%d) running", i)
	}
	logger.MustGetSysLogger().Info(ctx, "finish creating worker pool")
}

func (e *workerEngine) runWorker(ctx context.Context, workerId uint) {
	var (
		traceModule = "task_worker"
		origCtxTraceId string
	)
	if traceCtx, ok := ctx.(*simpletracectx.Context); ok {
		origCtxTraceId = traceCtx.GetTraceId()
	}
	for {
		ctx = contxt.NewWithTrace(traceModule, ctx, traceModule + "_" + origCtxTraceId + "." + simpletrace.NewTraceId(), "")

		logger.MustGetSysLogger().Infof(ctx, "worker(id:%d) is ready", workerId)

		task := e.sched.nextTask()

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
