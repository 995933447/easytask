package mysql

import (
	"context"
	"github.com/995933447/easytask/internal/task"
	"github.com/995933447/easytask/internal/util/logger"
	"github.com/995933447/optionstream"
	"sync/atomic"
	"time"
)

var migratedTaskLogRepoDB atomic.Bool

type TaskLogRepo struct {
	repoConnector
}

func (r *TaskLogRepo) SaveTaskStartedLog(ctx context.Context, detail *task.TaskStartedLogDetail) error {
	taskModelId, err := toTaskModelId(detail.GetTask().GetId())
	if err != nil {
		logger.MustGetRepoLogger().Error(ctx, err)
		return err
	}
	srvModelId, err := toCallbackSrvRouteModelId(detail.GetTask().GetCallbackSrv().GetId())
	if err != nil {
		logger.MustGetRepoLogger().Error(ctx, err)
		return err
	}
	err = r.mustGetConn(ctx).Create(&TaskLogModel{
		TaskId: taskModelId,
		StartedAt: time.Now().Unix(),
		TaskStatus: statusRunning,
		RunTimes: detail.GetTask().GetRunTimes(),
		SrvId: srvModelId,
	}).Error
	if err != nil {
		logger.MustGetRepoLogger().Error(ctx, err)
		return err
	}
	return nil
}

func (r *TaskLogRepo) SaveTaskCallbackLog(ctx context.Context, detail *task.TaskCallbackLogDetail) error {
	updateMap := map[string]interface{}{
		DbFieldIsRunInAsync: detail.IsRunInAsync(),
		DbFieldReqSnapshot: &TaskLogCallbackReqSnapshot{
			SrvSchema: detail.GetRoute().GetSchema(),
			Host: detail.GetRoute().GetHost(),
			Port: detail.GetRoute().GetPort(),
			TimeoutSec: detail.GetRoute().GetCallbackTimeoutSec(),
			CallbackAt: time.Now().Unix(),
			CallbackPath: detail.GetCallbackPath(),
		},
		DbFieldRespSnapshot: &TaskLogCallbackRespSnapshot{
			RespRaw: detail.GetRespRaw(),
		},
	}
	if task.IsTaskSuccess(detail.GetTaskStatus()) {
		updateMap[DbFieldTaskStatus] = statusSuccess
		updateMap[DbFieldEndedAt] = time.Now().Unix()
	} else if task.IsTaskFailed(detail.GetTaskStatus()) {
		updateMap[DbFieldTaskStatus] = statusFailed
		updateMap[DbFieldEndedAt] = time.Now().Unix()
	}
	if detail.GetErr() != nil {
		updateMap[DbFieldCallbackErr] = detail.GetErr()
	}
	err := r.mustGetConn(ctx).
		Model(&TaskLogModel{}).
		Where(DbFieldTaskId, detail.GetTaskId()).
		Where(DbFieldRunTimes, detail.GetRunTimes()).
		Updates(updateMap).Error
	if err != nil {
		logger.MustGetRepoLogger().Error(ctx, err)
		return err
	}
	return nil
}

func (r *TaskLogRepo) SaveTaskConfirmedLog(ctx context.Context, detail *task.TaskConfirmedLogDetail) error {
	updateMap := map[string]interface{}{
		DbFieldEndedAt: time.Now().Unix(),
		DbFieldRespExtra: detail.GetTaskResp().GetExtra(),
	}

	if task.IsTaskSuccess(detail.GetTaskResp().GetTaskStatus()) {
		updateMap[DbFieldTaskStatus] = statusSuccess
	} else if task.IsTaskFailed(detail.GetTaskResp().GetTaskStatus()) {
		updateMap[DbFieldTaskStatus] = statusFailed
	}

	err := r.mustGetConn(ctx).
		Model(&TaskLogModel{}).
		Where(DbFieldTaskId, detail.GetTaskResp().GetTaskId()).
		Where(DbFieldRunTimes, detail.GetTaskResp().GetTaskRunTimes()).
		Updates(updateMap).
		Error
	if err != nil {
		logger.MustGetRepoLogger().Error(ctx, err)
		return err
	}

	return nil
}

func (r *TaskLogRepo) DelLogs(ctx context.Context, stream *optionstream.Stream) error {
	conn :=  r.mustGetConn(ctx)
	err := optionstream.NewStreamProcessor(stream).
		OnInt64(task.QueryOptKeyCreatedExceed, func(val int64) error {
			conn = conn.Where(DbFieldCreatedAt + " > ?", val)
			return nil
		}).
		Process()
	if err != nil {
		logger.MustGetRepoLogger().Error(ctx, err)
		return err
	}

	if err = conn.Unscoped().Delete(&TaskModel{}).Error; err != nil {
		logger.MustGetRepoLogger().Error(ctx, err)
		return err
	}

	return nil
}

func NewTaskLogRepo(ctx context.Context, connDsn string) (task.TaskLogRepo, error) {
	repo := &TaskLogRepo{
		repoConnector{
			connDsn: connDsn,
		},
	}
	if !migratedTaskLogRepoDB.Load() {
		if err := repo.mustGetConn(ctx).AutoMigrate(&TaskLogModel{}); err != nil {
			logger.MustGetRepoLogger().Error(ctx, err)
			return nil, err
		}
	}
	return repo, nil
}
