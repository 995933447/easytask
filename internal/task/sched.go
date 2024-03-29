package task

import (
	"context"
	"github.com/995933447/autoelect"
	"github.com/995933447/easytask/internal/util/errs"
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
	exitSignCh    chan struct{}
}

func (s *Sched) lockTaskForRun(ctx context.Context, task *Task) (bool, error) {
	locked, err := s.taskRepo.LockTask(ctx, task)
	if err != nil {
		logger.MustGetTaskLogger().Error(ctx, err)
		return false, err
	}
	return locked, nil
}

func (s *Sched) nextTask() *Task {
	return <- s.taskCh
}

func (s *Sched) run(ctx context.Context) {
	s.schedule(ctx)
}

func (s *Sched) stop() {
	s.exitSignCh <- struct{}{}
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
		var isExitingSched bool
		select {
		case _ = <- s.exitSignCh:
			isExitingSched = true
		default:
		}

		if isExitingSched {
			break
		}

		ctx = contxt.NewWithTrace(traceModule, context.TODO(), traceModule + "_" + origCtxTraceId + "." + simpletrace.NewTraceId(), "")
		if !s.elect.IsMaster() {
			err := errs.ErrCurrentNodeNoMaster
			logger.MustGetRegistryLogger().Error(ctx, err)
			time.Sleep(time.Second)
			continue
		}

		tasks, nextCursor, err := s.taskRepo.TimeoutTasks(ctx, size, cursor)
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


func (s *Sched) submitTaskResp(ctx context.Context, resp *TaskResp) error {
	if err := s.taskRepo.ConfirmTask(ctx, resp); err != nil {
		logger.MustGetSysLogger().Error(ctx, err)
		return err
	}
	return nil
}

func NewSched(taskRepo TaskRepo, elect autoelect.AutoElection) *Sched {
	return &Sched{
		taskCh: make(chan *Task),
		taskRepo: taskRepo,
		taskRespCh: make(chan *TaskResp),
		elect: elect,
		exitSignCh: make(chan struct{}),
	}
}