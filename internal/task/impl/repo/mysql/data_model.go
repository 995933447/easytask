package mysql

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
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
	Name string `gorm:"index:task_biz,unique;comment:'任务名称'"`
	Arg string `gorm:"comment:'参数'"`
	LastRunAt int64 `gorm:"comment:'上次运行时间'"`
	PlanSchedNextAt int64 `gorm:"comment:'下次计划执行时间'"`
	TimeCronExpr string `gorm:"comment:'定时cron表达式'"`
	TimeIntervalSec int `gorm:"comment:'执行间隔'"`
	SchedMode int `gorm:"comment:'执行模式.1.指定时间,2.cron表达式,3.间隔执行'"`
	CallbackSrvId uint64 `gorm:"comment:'回调服务id'"`
	RunTimes int `gorm:"comment:'已执行次数'"`
	AllowMaxRunTimes int `gorm:"comment:'最大可执行次数'"`
	MaxRunTimeSec int `gorm:"comment:'任务最长运行时间'"`
	CallbackPath string `gorm:"comment:'回调路径'"`
	BizId string `gorm:"index:task_biz,unique,comment:'用于指定任务唯一业务id'"`
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
		BizId: t.BizId,
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
	Name string `gorm:"index:server_name,unique;comment:'服务名称'"`
	CheckedHealthAt int64 `gorm:"comment:'上次健康检查时间'"`
	HasEnableHealthCheck bool `gorm:"comment:'是否有开启健康检查的路由'"`
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
	SrvSchema string `gorm:"index:route_addr,unique;comment:'协议'"`
	Host string `gorm:"index:route_addr,unique;comment:'host'"`
	Port int `gorm:"index:route_addr,unique;comment:'端口'"`
	SrvId uint64 `gorm:"index:route_addr,unique;comment:'服务id'"`
	CallbackTimeoutSec int `gorm:"comment:'回调超时时间'"`
	CheckedHealthAt int64 `gorm:"comment:'上次健康检查时间'"`
	EnableHealthCheck bool `gorm:"comment:'是否开启健康检查'"`
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
		m.SrvSchema,
		m.Host,
		m.Port,
		m.CallbackTimeoutSec,
		m.EnableHealthCheck,
		)
}

type TaskLogCallbackReqSnapshot struct {
	SrvSchema string `gorm:"index:route_addr,unique" json:"srv_schema"`
	Host string `gorm:"index:route_addr,unique" json:"host"`
	Port int `gorm:"index:route_addr,unique" json:"port"`
	TimeoutSec int `json:"timeout_sec"`
	CallbackAt int64 `json:"callback_at"`
	CallbackPath string `json:"callback_path"`
}

func (s TaskLogCallbackReqSnapshot) Value() (driver.Value, error) {
	j, err := json.Marshal(s)
	if err != nil {
		return nil, err
	}
	return j, nil
}

func (s *TaskLogCallbackReqSnapshot) Scan(src interface{}) error {
	err := json.Unmarshal(src.([]byte), s)
	if err != nil {
		return err
	}
	return nil
}

type TaskLogCallbackRespSnapshot struct {
	RespRaw string `json:"resp_raw"`
}

func (s TaskLogCallbackRespSnapshot) Value() (driver.Value, error) {
	j, err := json.Marshal(s)
	if err != nil {
		return nil, err
	}
	return string(j), nil
}

func (s *TaskLogCallbackRespSnapshot) Scan(src interface{}) error {
	err := json.Unmarshal([]byte(src.(string)), s)
	if err != nil {
		return err
	}
	return nil
}

type TaskLogModel struct {
	BaseModel
	TaskId uint64 `json:"task_id" gorm:"index:task_run_times,unique;comment:'任务id'"`
	StartedAt int64 `json:"started_at" gorm:"comment:'任务开始时间'"`
	EndedAt int64 `json:"ended_at" gorm:"comment:'任务结束时间'"`
	TaskStatus int `json:"task_status" gorm:"comment:'任务状态:2.进行中,3.成功,4.失败'"`
	IsRunInAsync bool `json:"is_run_in_async" gorm:"comment:'是否异步模式'"`
	RespExtra string `json:"resp_extra" gorm:"comment:'响应额外信息'"`
	RunTimes int `json:"try_times" gorm:"index:task_run_times,unique;comment:'任务是第几次执行'"`
	SrvId uint64 `json:"srv_id" gorm:"comment:'回调服务id'"`
	ReqSnapshot *TaskLogCallbackReqSnapshot `json:"req_snapshot" gorm:"comment:'请求快照'"`
	RespSnapshot *TaskLogCallbackRespSnapshot `json:"resp_snapshot" gorm:"comment:'响应快照'"`
	CallbackErr string `gorm:"comment:'回调错误'"`
}

func (*TaskLogModel) TableName() string {
	return "task_log"
}