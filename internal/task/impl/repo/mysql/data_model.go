package mysql

import (
	"errors"
	"fmt"
	"github.com/995933447/dbdriverutil/field"
	"github.com/995933447/easytask/internal/task"
	"gorm.io/plugin/soft_delete"
	"strconv"
)

const (
	schedModeNil = iota
	schedModeTimeSpec
	schedModeTimeCron
	schedModeTimeInterval
)

const (
	statusNil = iota
	statusReady
	statusRunning
	statusSuccess
	statusFailed
)

//go:generate structfieldconstgen -findPkgPath ../mysql -outFile ../mysql/db_field.go -prefix DbField

type BaseModel struct {
	Id uint64 `gorm:"primaryKey"`
	CreatedAt int64 `gorm:"autoCreateTime"`
	UpdatedAt int64 `gorm:"autoUpdateTime"`
	DeletedAt soft_delete.DeletedAt `gorm:"index"`
}

type TaskModel struct {
	BaseModel
	Name string
	Arg field.Json
	Status int
	LastRunAt int64
	PlanSchedNextAt int64
	TimeCronExpr string
	TimeIntervalSec int
	SchedMode int
	CallbackSrvId uint64
	RunTimes int
	IsRunInAsync bool
	LastFailedAt int64
	LastSuccessAt int64
	AllowMaxRunTimes int
	MaxRunTimeSec int
	CallbackPath string
}

func (*TaskModel) TableName() string {
	return "task"
}

func (t *TaskModel) toEntity(callbackSrv *task.TaskCallbackSrv) (*task.Task, error) {
	entitySchedMode, err := t.toEntitySchedMode()
	if err != nil {
		return nil, err
	}
	return task.NewTask(&task.NewTaskReq{
		Id: t.toEntityId(),
		Name: t.Name,
		Arg: t.Arg,
		RunTimes: t.RunTimes,
		LastRunAt: t.LastRunAt,
		AllowMaxRunTimes: t.AllowMaxRunTimes,
		MaxRunTimeSec: t.MaxRunTimeSec,
		CallbackSrv: callbackSrv,
		TimeIntervalSec: t.TimeIntervalSec,
		TimeCronExpr: t.TimeCronExpr,
		SchedMode: entitySchedMode,
	})
}

func (t *TaskModel) toEntitySchedMode() (task.SchedMode, error) {
	return toTaskEntitySchedMode(t.SchedMode)
}

func (t *TaskModel) toEntityId() string {
	return toTaskEntityId(t.Id)
}

func toTaskModelSchedMode(entitySchedMode task.SchedMode) (int, error) {
	switch entitySchedMode {
	case task.SchedModeTimeCron:
		return schedModeTimeCron, nil
	case task.SchedModeTimeSpec:
		return schedModeTimeSpec, nil
	case task.SchedModeTimeInterval:
		return schedModeTimeInterval, nil
	}
	return schedModeNil, errors.New("invalid schedule mode")
}

func toTaskEntitySchedMode(schedMode int) (task.SchedMode, error) {
	switch schedMode {
	case schedModeTimeCron:
		return task.SchedModeTimeCron, nil
	case schedModeTimeSpec:
		return task.SchedModeTimeSpec, nil
	case schedModeTimeInterval:
		return task.SchedModeTimeInterval, nil
	}
	return task.SchedModeNil, errors.New("invalid schedule mode")
}

func toTaskEntityId(modelId uint64) string {
	return fmt.Sprintf("%d", modelId)
}

func toTaskModelId(entityId string) (uint64, error) {
	return strconv.ParseUint(entityId, 10, 64)
}

type TaskCallbackSrvModel struct {
	BaseModel
	Name string
	CheckedHealthAt int64
	HasEnableHealthCheck bool
}

func (*TaskCallbackSrvModel) TableName() string {
	return "task_callback_srv"
}

func (m *TaskCallbackSrvModel) toEntity(routes []*task.TaskCallbackSrvRoute) *task.TaskCallbackSrv {
	return task.NewTaskCallbackSrv(m.toEntityId(), m.Name, routes, m.HasEnableHealthCheck)
}

func (m *TaskCallbackSrvModel) toEntityId() string {
	return toTaskCallbackSrvEntityId(m.Id)
}

func toTaskCallbackSrvEntityId(modelId uint64) string {
	return fmt.Sprintf("%d", modelId)
}

func toTaskCallbackSrvModelId(entity *task.TaskCallbackSrv) (uint64, error) {
	return strconv.ParseUint(entity.GetId(), 10, 64)
}

type TaskCallbackSrvRouteModel struct {
	BaseModel
	Schema string
	Host string
	Port int
	SrvId uint64
	CallbackTimeoutSec int
	CheckedHealthAt int64
	EnableHealthCheck bool
}

func (*TaskCallbackSrvRouteModel) TableName() string {
	return "task_callback_srv_route"
}


func toTackCallbackSrvRouteModelId(entityId string) (uint64, error) {
	return strconv.ParseUint(entityId, 10, 64)
}

func toCallbackSrvRouteEntityId(modelId uint64) string {
	return fmt.Sprintf("%d", modelId)
}

func (m *TaskCallbackSrvRouteModel) toEntityId() string {
	return toCallbackSrvRouteEntityId(m.Id)
}

func toCallbackSrvRouteModelId(entityId string) (uint64, error) {
	return strconv.ParseUint(entityId, 10, 64)
}

func (m *TaskCallbackSrvRouteModel) toEntity() *task.TaskCallbackSrvRoute {
	return task.NewTaskCallbackSrvRoute(
		m.toEntityId(),
		m.Schema,
		m.Host,
		m.Port,
		m.CallbackTimeoutSec,
		m.EnableHealthCheck,
		)
}

type TaskLogModel struct {
	BaseModel
	TaskId uint64 `json:"task_id"`
	StartedAt int64 `json:"started_at"`
	EndedAt int64 `json:"ended_at"`
	TaskStatus int `json:"task_status"`
	IsRunInAsync bool `json:"is_run_in_async"`
	RespExtra field.Json `json:"resp_extra"`
	TryTimes int `json:"try_times"`
}

func (*TaskLogModel) TableName() string {
	return "task_log"
}