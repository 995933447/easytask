package sched

import (
	"context"
	"github.com/995933447/autoelect"
	"github.com/995933447/easytask/internal/repo"
	"github.com/995933447/easytask/internal/task"
	"time"
)

type Sched struct {
	taskWorkerCh chan chan *task.Task
	taskRepo repo.TaskRepo
	taskRespCh chan *task.TaskResp
	elect autoelect.AutoElection
	isClusterMode 		bool
}

func (s *Sched) LockTaskForRun(ctx context.Context, task *task.Task) (bool, error) {
	return s.taskRepo.LockTask(ctx, task)
}

func (s *Sched) TaskWorkerReady(taskIn chan *task.Task) {
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

		tasks, err := s.taskRepo.TimeoutTasks(ctx, 1000)
		if err != nil {
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
		var taskResps []*task.TaskResp
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

		err := s.taskRepo.ConfirmTasks(ctx, taskResps)
		if err != nil {

		}
	}
}

func (s *Sched) SubmitTaskResp(resp *task.TaskResp) {
	s.taskRespCh <- resp
}

func NewSched(isClusterMode bool, taskRepo repo.TaskRepo, elect autoelect.AutoElection) *Sched {
	return &Sched{
		taskWorkerCh: make(chan chan *task.Task),
		taskRepo: taskRepo,
		taskRespCh: make(chan *task.TaskResp),
		elect: elect,
		isClusterMode: isClusterMode,
	}
}