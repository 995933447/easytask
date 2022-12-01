package task

import (
	"context"
	"github.com/995933447/easytask/internal/callbacksrvexec"
	"github.com/go-playground/validator"
	"math/rand"
	"time"
)

type Status int

const (
	StatusReady Status = iota
	StatusRunning
	StatusSuccess
	StatusFailed
)

type (
	TaskResp struct {
		taskId string
		isRunInAsync bool
		taskStatus Status
		extra any
	}

	InternalErrTaskRespDetail struct {
		Err error
		OccurredAt int64
	}
)

func (r *TaskResp) GetTaskId() string {
	return r.taskId
}

func (r *TaskResp) IsRunInAsync() bool {
	return r.isRunInAsync
}

func (r *TaskResp) GetExtra() any {
	return r.extra
}

func (r *TaskResp) GetTaskStatus() Status {
	return r.taskStatus
}

func newTaskResp(taskId string, isRunInAsync bool, taskStatus Status, extra any) *TaskResp {
	return &TaskResp{
		taskId: taskId,
		isRunInAsync: isRunInAsync,
		taskStatus: taskStatus,
		extra: extra,
	}
}

func newInternalErrTaskResp(taskId string, err error, occurredAt int64) *TaskResp {
	return &TaskResp{
		taskId: taskId,
		taskStatus: StatusFailed,
		extra: InternalErrTaskRespDetail{
			Err: err,
			OccurredAt: occurredAt,
		},
	}
}

type TaskCallbackSrvRoute struct {
	id string
	scheme string
	host string
	port int
	callbackTimeoutSec int
	isEnableHealthCheck bool
}

func (r *TaskCallbackSrvRoute) IsEnableHeathCheck() bool {
	return r.isEnableHealthCheck
}

func (r *TaskCallbackSrvRoute) GetId() string {
	return r.scheme
}

func (r *TaskCallbackSrvRoute) GetSchema() string {
	return r.scheme
}

func (r *TaskCallbackSrvRoute) GetHost() string {
	return r.host
}

func (r *TaskCallbackSrvRoute) GetPort() int {
	return r.port
}

func (r *TaskCallbackSrvRoute) GetCallbackTimeoutSec() int {
	return r.callbackTimeoutSec
}

func NewTaskCallbackSrvRoute(id, schema, host string, port, callbackTimeoutSec int, isEnableHealthCheck bool) *TaskCallbackSrvRoute {
	return &TaskCallbackSrvRoute{
		id: id,
		scheme: schema,
		host: host,
		port: port,
		callbackTimeoutSec: callbackTimeoutSec,
		isEnableHealthCheck: isEnableHealthCheck,
	}
}

type TaskCallbackSrv struct {
	name   string
	routes []*TaskCallbackSrvRoute
	hasEnableHealthCheck bool
}

func (s *TaskCallbackSrv) HasEnableHealthCheckRoute() bool {
	if s.hasEnableHealthCheck {
		return true
	}
	for _, route := range s.routes {
		if route.isEnableHealthCheck {
			return true
		}
	}
	return false
}

func (s *TaskCallbackSrv) GetName() string {
	return s.name
}

func (s *TaskCallbackSrv) GetRoutes() []*TaskCallbackSrvRoute {
	return s.routes
}

func (s *TaskCallbackSrv) GetRandomRoute() *TaskCallbackSrvRoute {
	rand.Seed(time.Now().UnixNano())
	return s.routes[rand.Intn(len(s.routes))]
}

func NewTaskCallbackSrv(name string, routes []*TaskCallbackSrvRoute, hasEnableHealthCheck bool) *TaskCallbackSrv {
	return &TaskCallbackSrv{
		name: name,
		routes: routes,
		hasEnableHealthCheck: hasEnableHealthCheck,
	}
}

type Task struct {
	id string
	callbackSrv *TaskCallbackSrv
	name string
	arg []byte
	isRunInAsync bool
	status Status
	triedCnt int
	lastRunAt int64
	maxTryCnt int
	skippedTryCnt int
	maxRunTimeSec int
}

func (t *Task) GetMaxRuntimeSec() int {
	return t.maxRunTimeSec
}

func (t *Task) GetSkippedTryCnt() int {
	return t.skippedTryCnt
}

func (t *Task) GetMaxTryCnt() int {
	return t.maxTryCnt
}

func (t *Task) GetLastRunAt() int64 {
	return t.lastRunAt
}

func (t *Task) GetId() string {
	return t.id
}

func (t *Task) GetName() string {
	return t.name
}

func (t *Task) GetCallbackSrv() *TaskCallbackSrv {
	return t.callbackSrv
}

func (t *Task) GetArg() []byte {
	return t.arg
}

func (t *Task) IsRunInAsync() bool {
	return t.isRunInAsync
}

func (t *Task) GetStatus() Status {
	return t.status
}

func (t *Task) GetTriedCnt() int {
	return t.triedCnt
}

func (t *Task) IsReady() bool {
	return IsTaskReady(t.status)
}

func (t *Task) IsRunning() bool {
	return IsTaskRunning(t.status)
}

func (t *Task) IsSuccess() bool {
	return IsTaskSuccess(t.status)
}

func (t *Task) IsFailed() bool {
	return IsTaskFailed(t.status)
}

func (t *Task) run(ctx context.Context, callbackExec callbacksrvexec.TaskCallbackSrvExec) (*TaskResp, error) {
	callbackResp, err := callbackExec.CallbackSrv(ctx, t, nil)
	if err != nil {
		return nil, err
	}

	var status Status
	if !callbackResp.IsSuccess() {
		status = StatusFailed
	} else if callbackResp.IsRunInAsync() {
		status = StatusRunning
	} else {
		status = StatusSuccess
	}

	return newTaskResp(t.id, callbackResp.IsRunInAsync(), status, callbackResp.GetExtra()), nil
}

type NewTaskInput struct {
	Id string
	CallbackSrv *TaskCallbackSrv `validate:"required"`
	Name string `validate:"required"`
	Arg []byte `validate:"required"`
	IsRunInAsync bool
	Status Status
	TriedCnt int
	LastRunAt int64
	MaxTryCnt int
	SkippedTryCnt int
	MaxRunTimeSec int
}

func (input *NewTaskInput) Check() error {
	return validator.New().Struct(input)
}

func NewTask(input *NewTaskInput) (*Task, error) {
	// checking if forgot set field
	if err := input.Check(); err != nil {
		return nil, err
	}
	return &Task{
		id: input.Id,
		name: input.Name,
		arg: input.Arg,
		status: input.Status,
		triedCnt: input.TriedCnt,
		callbackSrv: input.CallbackSrv,
		isRunInAsync: input.IsRunInAsync,
	}, nil
}

func IsTaskReady(status Status) bool {
	return status == StatusReady
}

func IsTaskRunning(status Status) bool {
	return status == StatusRunning
}

func IsTaskSuccess(status Status) bool {
	return status == StatusSuccess
}

func IsTaskFailed(status Status) bool {
	return status == StatusFailed
}