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
	e.sched.Run(ctx)
}

func (e *workerEngine) createWorkerPool(ctx context.Context) {
	logger.MustGetSysLogger().Info(ctx, "start create worker pool")
	var i uint
	for ; i < e.workerPoolSize; i++ {
		go e.runWorker(ctx, i)
		logger.MustGetSysLogger().Infof(ctx, "task worker(id:%d) running", i)
	}
	logger.MustGetSysLogger().Info(ctx, "created worker pool")
}

func (e *workerEngine) runWorker(ctx context.Context, workerId uint) {
	var (
		traceModule = "task_worker"
		origCtxTraceId string
		log = logger.MustGetTaskLogger()
	)
	if traceCtx, ok := ctx.(*simpletracectx.Context); ok {
		origCtxTraceId = traceCtx.GetTraceId()
	}
	for {
		ctx = contxt.NewWithTrace(traceModule, ctx, traceModule + "_" + origCtxTraceId + "." + simpletrace.NewTraceId(), "")

		log.Infof(ctx, "worker(id:%d) is ready", workerId)

		task := e.sched.NextTask()

		if len(task.callbackSrv.GetRoutes()) == 0 {
			log.Warnf(
				ctx,
				"task(id:%s, name:%s) callback server(name:%s) has no routes",
				task.id, task.name, task.callbackSrv.name,
			)
			return
		}

		log.Infof(ctx, "worker(id:%d) run task(id:%s name:%s)", workerId, task.id, task.name)

		now := time.Now()

		locked, err := e.sched.LockTaskForRun(contxt.ChildOf(ctx), task)
		if err != nil {
			log.Error(ctx, err)
			continue
		}

		if !locked {
			log.Warnf(ctx, "worker(id:%d) lock task(id:%s) failed", workerId, task.id)
			continue
		}

		taskResp, err := task.run(contxt.ChildOf(ctx), e.callbackTaskSrvExec)
		if err != nil {
			log.Error(ctx, err)
			taskResp, err = newInternalErrTaskResp(task.id, err, now.Unix())
			if err != nil {
				log.Error(ctx, err)
				continue
			}
		}

		err = e.sched.SubmitTaskResp(ctx, taskResp)
		if err != nil {
			log.Error(ctx, err)
		}

		log.Infof(
			ctx,
			"worker(id:%d) finish task(id:%s name:%s), exec time:%d ms",
			workerId, task.id, task.name, time.Now().Sub(now) / time.Millisecond,
		)
	}
}
