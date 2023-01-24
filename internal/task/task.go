package task

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/995933447/easytask/internal/util/logger"
	"github.com/go-playground/validator"
	"github.com/gorhill/cronexpr"
	"math/rand"
	"time"
)

type Status int

const (
	StatusNil Status = iota
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
		extra string
	}

	InternalErrTaskRespDetail struct {
		Err error `json:"err"`
		OccurredAt int64 `json:"occurred_at"`
	}
)

func (r *TaskResp) GetTaskId() string {
	return r.taskId
}

func (r *TaskResp) IsRunInAsync() bool {
	return r.isRunInAsync
}

func (r *TaskResp) GetExtra() string {
	return r.extra
}

func (r *TaskResp) GetTaskStatus() Status {
	return r.taskStatus
}

func (r *TaskResp) GetTaskRunTimes() int {
	return r.taskRunTimes
}

func NewTaskResp(taskId string, isRunInAsync bool, taskStatus Status, taskRunTimes int, extra string) *TaskResp {
	return &TaskResp{
		taskId: taskId,
		isRunInAsync: isRunInAsync,
		taskStatus: taskStatus,
		extra: extra,
		taskRunTimes: taskRunTimes,
	}
}

func newInternalErrTaskResp(taskId string, err error, occurredAt int64) (*TaskResp, error) {
	detail := InternalErrTaskRespDetail{
		Err: err,
		OccurredAt: occurredAt,
	}

	extra, err := json.Marshal(detail)
	if err != nil {
		return nil, err
	}

	return &TaskResp{
		taskId: taskId,
		taskStatus: StatusFailed,
		extra: string(extra),
	}, nil
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
	return r.id
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
	if len(s.routes) == 0 {
		return nil
	}
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
	arg string
	runTimes int
	lastRunAt int64
	allowMaxRunTimes int
	maxRunTimeSec int
	schedMode SchedMode
	timeCronExpr string
	timeIntervalSec int
	timeSpecAt int64
	bizId string
}

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
	return 0, errors.New("unknown schedule at")
}

func (t *Task) GetBizId() string {
	return t.bizId
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

func (t *Task) GetArg() string {
	return t.arg
}

func (t *Task) GetRunTimes() int {
	return t.runTimes
}

func (t *Task) IncrRunTimes() {
	t.runTimes++
}

func (t *Task) run(ctx context.Context, callbackExec TaskCallbackSrvExec) (*TaskResp, error) {
	callbackResp, err := callbackExec.CallbackSrv(ctx, t, nil)
	if err != nil {
		logger.MustGetTaskLogger().Error(ctx, err)
		return nil, err
	}

	var status Status
	if callbackResp.IsSuccess() {
		status = StatusSuccess
	} else if callbackResp.IsRunInAsync() {
		status = StatusRunning
	} else {
		status = StatusFailed
	}

	return NewTaskResp(t.id, callbackResp.IsRunInAsync(), status, t.runTimes, callbackResp.GetExtra()), nil
}

type NewTaskReq struct {
	Id string
	CallbackSrv *TaskCallbackSrv `validate:"required"`
	CallbackPath string `json:"callback_path"`
	Name string `validate:"required"`
	Arg string
	RunTimes int
	LastRunAt int64
	AllowMaxRunTimes int
	MaxRunTimeSec int
	SchedMode SchedMode `validate:"required"`
	TimeCronExpr string
	TimeIntervalSec int
	TimeSpecAt int64
	BizId string
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
		bizId: req.BizId,
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

type TaskLogType int

const (
	TaskLogTypeNil TaskLogType = iota
	TaskLogTypeStarted
	TaskLogTypeCallback
	TaskLogTypeConfirmed
)

type TaskStartedLogDetail struct {
	task *Task
}

func (d *TaskStartedLogDetail) GetTask() *Task {
	return d.task
}

func NewTaskStartedLogDetail(task *Task) *TaskStartedLogDetail {
	return &TaskStartedLogDetail{
		task: task,
	}
}

type NewTaskCallbackLogDetailReq struct {
	TaskId string `validate:"required"`
	CallbackPath string
	RunTimes int
	Route *TaskCallbackSrvRoute `validate:"required"`
	RespRaw string
	IsRunInAsync bool
	TaskStatus Status
	Err error
}

func (r *NewTaskCallbackLogDetailReq) check() error {
	return validator.New().Struct(r)
}

func NewTaskCallbackLogDetail(req *NewTaskCallbackLogDetailReq) (*TaskCallbackLogDetail, error) {
	if err := req.check(); err != nil {
		return nil, err
	}
	return &TaskCallbackLogDetail{
		taskId: req.TaskId,
		callbackPath: req.CallbackPath,
		runTimes: req.RunTimes,
		route: req.Route,
		respRaw: req.RespRaw,
		isRunInAsync: req.IsRunInAsync,
		taskStatus: req.TaskStatus,
		err: req.Err,
	}, nil
}

type TaskCallbackLogDetail struct {
	taskId string
	callbackPath string
	runTimes int
	route *TaskCallbackSrvRoute
	respRaw string
	isRunInAsync bool
	taskStatus Status
	err error
}

func (d *TaskCallbackLogDetail) GetCallbackPath() string {
	return d.callbackPath
}

func (d *TaskCallbackLogDetail) IsRunInAsync() bool {
	return d.isRunInAsync
}

func (d *TaskCallbackLogDetail) GetTaskStatus() Status {
	return d.taskStatus
}

func (d *TaskCallbackLogDetail) GetTaskId() string {
	return d.taskId
}

func (d *TaskCallbackLogDetail) GetRunTimes() int {
	return d.runTimes
}

func (d *TaskCallbackLogDetail) GetRoute() *TaskCallbackSrvRoute {
	return d.route
}

func (d *TaskCallbackLogDetail) GetRespRaw() string {
	return d.respRaw
}

func (d *TaskCallbackLogDetail) GetErr() error {
	return d.err
}

type TaskConfirmedLogDetail struct {
	taskResp *TaskResp
}

func (d *TaskConfirmedLogDetail) GetTaskResp() *TaskResp {
	return d.taskResp
}

func NewTaskConfirmedLogDetail(taskResp *TaskResp) *TaskConfirmedLogDetail {
	return &TaskConfirmedLogDetail{
		taskResp: taskResp,
	}
}

type TaskLog struct {
	logType TaskLogType
	taskStartedDetail *TaskStartedLogDetail
	taskCallbackDetail *TaskCallbackLogDetail
	taskConfirmedDetail *TaskConfirmedLogDetail
}

func (l *TaskLog) GetType() TaskLogType {
	return l.logType
}

func (l *TaskLog) GetTaskStartedDetail() *TaskStartedLogDetail {
	return l.taskStartedDetail
}

func (l *TaskLog) GetTaskCallbackDetail() *TaskCallbackLogDetail {
	return l.taskCallbackDetail
}

func (l *TaskLog) GetTaskConfirmedDetail() *TaskConfirmedLogDetail {
	return l.taskConfirmedDetail
}

func MustNewTaskLog(logType TaskLogType, detail any) *TaskLog {
	taskLog := &TaskLog{logType: logType}
	switch logType {
	case TaskLogTypeStarted:
		taskLog.taskStartedDetail = detail.(*TaskStartedLogDetail)
	case TaskLogTypeCallback:
		taskLog.taskCallbackDetail = detail.(*TaskCallbackLogDetail)
	case TaskLogTypeConfirmed:
		taskLog.taskConfirmedDetail = detail.(*TaskConfirmedLogDetail)
	default:
		panic(any("unknown log type"))
	}
	return taskLog
}

type TaskLogger struct {
	repo TaskLogRepo
	runtimeLogger logger.Logger
}

func (l *TaskLogger) Log(ctx context.Context, log *TaskLog) error {
	switch log.logType {
	case TaskLogTypeStarted:
		if err := l.repo.SaveTaskStartedLog(ctx, log.taskStartedDetail); err != nil {
			l.runtimeLogger.Error(ctx, err)
			return err
		}
	case TaskLogTypeCallback:
		if err := l.repo.SaveTaskCallbackLog(ctx, log.taskCallbackDetail); err != nil {
			l.runtimeLogger.Error(ctx, err)
			return err
		}
	case TaskLogTypeConfirmed:
		if err := l.repo.SaveTaskConfirmedLog(ctx, log.taskConfirmedDetail); err != nil {
			l.runtimeLogger.Error(ctx, err)
			return err
		}
	default:
		err := errors.New("not support logging task behavior")
		l.runtimeLogger.Error(ctx, err)
		return err
	}
	return nil
}

func NewTaskLogger(repo TaskLogRepo, runtimeLogger logger.Logger) *TaskLogger {
	return &TaskLogger{
		repo: repo,
		runtimeLogger: runtimeLogger,
	}
}