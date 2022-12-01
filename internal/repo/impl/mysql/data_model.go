package mysql

import (
	"fmt"
	"github.com/995933447/dbdriverutil/field"
	"github.com/995933447/easytask/internal/task"
	"gorm.io/plugin/soft_delete"
	"strconv"
)

const (
	execModeNil = iota
	execModeTimeSpec
	execModeCron
)

const (
	statusNil = iota
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
	NextRunAt int64
	TimeCron string
	ExecMode int
	CallbackSrvId uint64
	TriedCnt int
	IsRunInAsync bool
	LastFailedAt int64
	MaxTryCnt int
	SkippedTryCnt int
	MaxRunTimeSec int
}

func (t *TaskModel) toEntity(callbackSrv *task.TaskCallbackSrv) (*task.Task, error) {
	taskEntityStatus, err := t.toEntityStatus()
	if err != nil {
		return nil, err
	}
	return task.NewTask(&task.NewTaskInput{
		Id: t.toEntityId(),
		Name: t.Name,
		Arg: t.Arg,
		IsRunInAsync: t.IsRunInAsync,
		Status: taskEntityStatus,
		TriedCnt: t.TriedCnt,
		LastRunAt: t.LastRunAt,
		MaxTryCnt: t.MaxTryCnt,
		SkippedTryCnt: t.SkippedTryCnt,
		MaxRunTimeSec: t.MaxRunTimeSec,
		CallbackSrv: callbackSrv,
	})
}

func (t *TaskModel) toEntityStatus() (task.Status, error) {
	switch t.Status {
	case statusNil:
		return task.StatusReady, nil
	case statusRunning:
		return task.StatusRunning, nil
	case statusSuccess:
		return task.StatusSuccess, nil
	case statusFailed:
		return task.StatusFailed, nil
	}
	return 0, fmt.Errorf("invalid status:%d", t.Status)
}

func (t *TaskModel) toEntityId() string {
	return toTaskEntityId(t.Id)
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

func (m *TaskCallbackSrvModel) toEntity(routes []*task.TaskCallbackSrvRoute) *task.TaskCallbackSrv {
	return task.NewTaskCallbackSrv(m.Name, routes, m.HasEnableHealthCheck)
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

func toTackCallbackSrvRouteModelId(entityId string) (uint64, error) {
	return strconv.ParseUint(entityId, 10, 64)
}

func toCallbackSrvRouteEntityId(modelId uint64) string {
	return fmt.Sprintf("%d", modelId)
}

func (m *TaskCallbackSrvRouteModel) toEntityId() string {
	return toCallbackSrvRouteEntityId(m.Id)
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