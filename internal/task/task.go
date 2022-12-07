package task

import (
	"context"
	"github.com/995933447/easytask/internal/callbacksrvexec"
	"github.com/995933447/easytask/internal/util/logger"
	"errors"
	"github.com/go-playground/validator"
	"github.com/gorhill/cronexpr"
	"math/rand"
	"time"
)

type Status int

const (
	StatusNil = iota
	StatusReady
	StatusRunning
	StatusSuccess
	StatusFailed
)

type SchedMode int

const (
	SchedModeNil SchedMode = iota
	SchedModeTimeCron
	SchedModeTimeSpec
	SchedModeTimeInterval
)

type (
	TaskResp struct {
		taskId string
		isRunInAsync bool
		taskStatus Status
		taskRunTimes int
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

func (r *TaskResp) GetTaskRunTimes() int {
	return r.taskRunTimes
}

func NewTaskResp(taskId string, isRunInAsync bool, taskStatus Status, taskRunTimes int, extra any) *TaskResp {
	return &TaskResp{
		taskId: taskId,
		isRunInAsync: isRunInAsync,
		taskStatus: taskStatus,
		extra: extra,
		taskRunTimes: taskRunTimes,
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
	id string
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

func (s *TaskCallbackSrv) GetId() string {
	return s.id
}

func (s *TaskCallbackSrv) GetRandomRoute() *TaskCallbackSrvRoute {
	rand.Seed(time.Now().UnixNano())
	return s.routes[rand.Intn(len(s.routes))]
}

func NewTaskCallbackSrv(id, name string, routes []*TaskCallbackSrvRoute, hasEnableHealthCheck bool) *TaskCallbackSrv {
	return &TaskCallbackSrv{
		id: id,
		name: name,
		routes: routes,
		hasEnableHealthCheck: hasEnableHealthCheck,
	}
}

type Task struct {
	id string
	callbackSrv *TaskCallbackSrv
	callbackPath string
	name string
	arg []byte
	runTimes int
	lastRunAt int64
	allowMaxRunTimes int
	maxRunTimeSec int
	schedMode SchedMode
	timeCronExpr string
	timeIntervalSec int
	timeSpecAt int64
}

var ErrUnknownSchedAt = errors.New("unknown schedule at")

func (t *Task) GetSchedNextAt() (int64, error) {
	now := time.Now()
	switch t.GetSchedMode() {
	case SchedModeTimeInterval:
		return now.Unix() + int64(t.GetTimeIntervalSec()), nil
	case SchedModeTimeCron:
		expr, err := cronexpr.Parse(t.GetTimeCronExpr())
		if err != nil {
			return 0, err
		}
		return expr.Next(now).Unix(), nil
	case SchedModeTimeSpec:
		return t.timeSpecAt, nil
	}
	return 0, ErrUnknownSchedAt
}

func (t *Task) GetTimeIntervalSec() int {
	return t.timeIntervalSec
}

func (t *Task) GetSchedMode() SchedMode {
	return t.schedMode
}

func (t *Task) GetTimeCronExpr() string {
	return t.timeCronExpr
}

func (t *Task) GetTimeSpecAt() int64 {
	return t.timeSpecAt
}

func (t *Task) GetMaxRunTimeSec() int {
	return t.maxRunTimeSec
}

func (t *Task) GetAllowMaxRunTimes() int {
	return t.allowMaxRunTimes
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

func (t *Task) GetCallbackPath() string {
	return t.callbackPath
}

func (t *Task) GetArg() []byte {
	return t.arg
}

func (t *Task) GetRunTimes() int {
	return t.runTimes
}

func (t *Task) IncrRunTimes() {
	t.runTimes++
}

func (t *Task) run(ctx context.Context, callbackExec callbacksrvexec.TaskCallbackSrvExec) (*TaskResp, error) {
	callbackResp, err := callbackExec.CallbackSrv(ctx, t, nil)
	if err != nil {
		logger.MustGetSysLogger().Error(ctx, err)
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

	return NewTaskResp(t.id, callbackResp.IsRunInAsync(), status, t.runTimes, callbackResp.GetExtra()), nil
}

type NewTaskReq struct {
	Id string
	CallbackSrv *TaskCallbackSrv `validate:"required"`
	CallbackPath string `json:"callback_path"`
	Name string `validate:"required"`
	Arg []byte `validate:"required"`
	RunTimes int
	LastRunAt int64
	AllowMaxRunTimes int
	MaxRunTimeSec int
	SchedMode SchedMode `validate:"required"`
	TimeCronExpr string
	TimeIntervalSec int
	TimeSpecAt int64
}

func (r *NewTaskReq) Check() error {
	return validator.New().Struct(r)
}

func NewTask(req *NewTaskReq) (*Task, error) {
	// checking if forgot set field
	if err := req.Check(); err != nil {
		return nil, err
	}
	return &Task{
		id: req.Id,
		name: req.Name,
		arg: req.Arg,
		runTimes: req.RunTimes,
		callbackSrv: req.CallbackSrv,
		callbackPath: req.CallbackPath,
		allowMaxRunTimes: req.AllowMaxRunTimes,
		lastRunAt: req.LastRunAt,
		maxRunTimeSec: req.MaxRunTimeSec,
		schedMode: req.SchedMode,
		timeCronExpr: req.TimeCronExpr,
		timeIntervalSec: req.TimeIntervalSec,
		timeSpecAt: req.TimeSpecAt,
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