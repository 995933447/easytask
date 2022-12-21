package task

import (
	"context"
	"github.com/995933447/autoelect"
	"github.com/995933447/easytask/internal/util/logger"
	"github.com/995933447/easytask/pkg/contxt"
	"github.com/995933447/simpletrace"
	simpletracectx "github.com/995933447/simpletrace/context"
	"time"
)

type Sched struct {
	taskRepo      TaskRepo
	taskCh        chan *Task
	taskRespCh    chan *TaskResp
	elect         autoelect.AutoElection
	isClusterMode bool
}

func (s *Sched) LockTaskForRun(ctx context.Context, task *Task) (bool, error) {
	locked, err := s.taskRepo.LockTask(contxt.ChildOf(ctx), task)
	if err != nil {
		logger.MustGetTaskLogger().Error(ctx, err)
		return false, err
	}
	return locked, nil
}

func (s *Sched) NextTask() *Task {
	return <- s.taskCh
}

func (s *Sched) Run(ctx context.Context) {
	go s.watchToConfirmTaskRes(ctx)
	s.schedule(ctx)
}

func (s *Sched) schedule(ctx context.Context) {
	var (
		traceModule = "task_sched"
		origCtxTraceId string
	)
	if traceCtx, ok := ctx.(*simpletracectx.Context); ok {
		origCtxTraceId = traceCtx.GetTraceId()
	}

	var (
		cursor string
		size = 1000
	)
	for {
		ctx = contxt.NewWithTrace(traceModule, ctx, traceModule + "_" + origCtxTraceId + "." + simpletrace.NewTraceId(), "")
		if s.isClusterMode && !s.elect.IsMaster() {
			logger.MustGetSysLogger().Debugf(ctx, "not master")
			time.Sleep(time.Second)
			continue
		}

		tasks, nextCursor, err := s.taskRepo.TimeoutTasks(contxt.ChildOf(ctx), size, cursor)
		if err != nil {
			logger.MustGetSysLogger().Error(ctx, err)
			continue
		}

		logger.MustGetSysLogger().Debugf(ctx, "scheduler tasks(len:%d)", len(tasks))

		if len(tasks) == 0 {
			logger.MustGetSysLogger().Debugf(ctx, "no more tasks, sleep 1 s, last cursor is %s", cursor)
			time.Sleep(time.Second)
			cursor = ""
			continue
		}

		cursor = nextCursor

		for _, oneTask := range tasks {
			s.taskCh <- oneTask
		}
	}
}

func (s *Sched) watchToConfirmTaskRes(ctx context.Context) {
	var (
		traceModule = "task_confirm"
		origCtxTraceId string
	)
	if traceCtx, ok := ctx.(*simpletracectx.Context); ok {
		origCtxTraceId = traceCtx.GetTraceId()
	}

	for {
		ctx = contxt.NewWithTrace(traceModule, ctx, traceModule + "_" + origCtxTraceId + "." + simpletrace.NewTraceId(), "")

		var taskResps []*TaskResp
		taskResp := <-s.taskRespCh
		taskResps = append(taskResps, taskResp)
		var noMoreTaskResp bool
		for {
			select {
			case taskResp := <- s.taskRespCh:
				taskResps = append(taskResps, taskResp)
			default:
				noMoreTaskResp = true
			}

			if noMoreTaskResp {
				break
			}
		}

		err := s.taskRepo.ConfirmTasks(contxt.ChildOf(ctx), taskResps)
		if err != nil {
			logger.MustGetSysLogger().Error(ctx, err)
		}
	}
}

func (s *Sched) SubmitTaskResp(resp *TaskResp) {
	s.taskRespCh <- resp
}

func NewSched(isClusterMode bool, taskRepo TaskRepo, elect autoelect.AutoElection) *Sched {
	return &Sched{
		taskCh: make(chan *Task, 10000),
		taskRepo: taskRepo,
		taskRespCh: make(chan *TaskResp),
		elect: elect,
		isClusterMode: isClusterMode,
	}
}