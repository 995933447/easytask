package task

import (
	"context"
	"github.com/995933447/autoelect"
	"github.com/995933447/easytask/internal/util/logger"
	"github.com/995933447/easytask/pkg/contxt"
	"time"
)

type Sched struct {
	taskWorkerCh  chan chan *Task
	taskRepo      TaskRepo
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

func (s *Sched) TaskWorkerReady(taskIn chan *Task) {
	s.taskWorkerCh <- taskIn
}

func (s *Sched) Run(ctx context.Context) {
	go s.watchToConfirmTaskRes(ctx)
	s.schedule(ctx)
}

func (s *Sched) schedule(ctx context.Context) {
	for {
		if s.isClusterMode && !s.elect.IsMaster() {
			time.Sleep(time.Second)
			continue
		}

		tasks, err := s.taskRepo.TimeoutTasks(contxt.ChildOf(ctx), 1000)
		if err != nil {
			logger.MustGetSysLogger().Error(ctx, err)
			continue
		}

		if len(tasks) == 0 {
			time.Sleep(time.Second)
			continue
		}

		for _, theTask := range tasks {
			taskIn := <- s.taskWorkerCh
			taskIn <- theTask
		}
	}
}

func (s *Sched) watchToConfirmTaskRes(ctx context.Context) {
	for {
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
		taskWorkerCh: make(chan chan *Task),
		taskRepo: taskRepo,
		taskRespCh: make(chan *TaskResp),
		elect: elect,
		isClusterMode: isClusterMode,
	}
}