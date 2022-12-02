package mysql

import (
	"context"
	"github.com/995933447/easytask/internal/repo"
	"github.com/995933447/easytask/internal/task"
	"github.com/995933447/easytask/internal/util/logger"
	"github.com/995933447/reflectutil"
	"github.com/gorhill/cronexpr"
	"gorm.io/gorm"
	"time"
)

type TaskRepo struct {
	srvRepo repo.TaskCallbackSrvRepo
	repoConnector
	defaultTaskMaxRunTimeSec int64
}

func (r *TaskRepo) TimeoutTasks(ctx context.Context, size int) ([]*task.Task, error) {
	var taskModels []*TaskModel

	err := r.mustGetConn().
		WithContext(ctx).
		Where(DbFieldRunTimes + " < " + DbFieldAllowMaxRunTimes).
		Where(DbFieldPlanSchedNextAt + " <= ?", time.Now().Unix()).
		Limit(size).
		Scan(&taskModels).
		Error
	if err != nil {
		logger.MustGetMainProcLogger().Error(ctx, err)
		return nil, err
	}

	callbackSrvModelIds := reflectutil.PluckUint64(taskModels, FieldCallbackSrvId)
	var callbackSrvIds []string
	for _, callbackSrvModelId := range callbackSrvModelIds {
		callbackSrvIds = append(callbackSrvIds, toCallbackSrvRouteEntityId(callbackSrvModelId))
	}
	callbackSrvs, err := r.srvRepo.GetSrvsByIds(ctx, callbackSrvIds)
	if err != nil {
		logger.MustGetMainProcLogger().Error(ctx, err)
		return nil, err
	}

	callbackSrvMap := reflectutil.MapByKey(callbackSrvs, FieldId).(map[uint64]*task.Task)

	var tasks []*task.Task
	for _, taskModel := range taskModels {
		callbackSrv, ok := callbackSrvMap[taskModel.CallbackSrvId]
		if !ok {
			continue
		}
		tasks = append(tasks, callbackSrv)
	}

	return nil, nil
}

func (r *TaskRepo) LockTask(ctx context.Context, oneTask *task.Task) (bool, error) {
	var (
		now = time.Now().Unix()
		taskModelUpdates = map[string]interface{}{
			DbFieldLastRunAt: now,
			DbFieldRunTimes: gorm.Expr(DbFieldRunTimes + " + 1"),
		}
	)
	switch oneTask.GetSchedMode() {
	case task.SchedModeTimeInterval:
		taskModelUpdates[DbFieldPlanSchedNextAt] = now + int64(oneTask.GetTimeIntervalSec())
	case task.SchedModeTimeCron:
		expr, err := cronexpr.Parse(oneTask.GetTimeCronExpr())
		if err != nil {
			return false, err
		}
		taskModelUpdates[DbFieldPlanSchedNextAt] = expr.Next(time.Now()).Unix()
	}

	taskModelId, err := toTaskModelId(oneTask.GetId())
	if err != nil {
		return false, err
	}
	conn := r.mustGetConn()
	res := conn.WithContext(ctx).
		Model(&TaskModel{}).
		Where(DbFieldId + " = ?", taskModelId).
		Where(DbFieldLastRunAt + " = ?", oneTask.GetLastRunAt()).
		Where(DbFieldRunTimes + " = ?", oneTask.GetRunTimes()).
		Updates(taskModelUpdates)
	if res.Error != nil {
		return false, res.Error
	}

	if res.RowsAffected > 0 {
		oneTask.IncrRunTimes()

		err = conn.Model(&TaskLogModel{}).Create(&TaskLogModel{
			TaskId: taskModelId,
			StartedAt: now,
			TaskStatus: statusRunning,
			TryTimes: oneTask.GetRunTimes(),
		}).Error
		if err != nil {
			return false, err
		}
	}

	return false, nil
}

func (r *TaskRepo) ConfirmTask(ctx context.Context, resp *task.TaskResp) error {
	now := time.Now().Unix()
	conn := r.mustGetConn().WithContext(ctx)
	taskModelId, err := toTaskModelId(resp.GetTaskId())
	taskStatus := statusRunning

	if task.IsTaskSuccess(resp.GetTaskStatus()) {
		err = conn.Model(&TaskModel{}).
			Where(DbFieldId + " = ?", taskModelId).
			Updates(map[string]interface{}{
				DbFieldLastSuccessAt: now,
			}).Error
		if err != nil {
			return err
		}
		taskStatus = statusSuccess
	}

	if task.IsTaskFailed(resp.GetTaskStatus()) {
		err = conn.Model(&TaskModel{}).
			Where(DbFieldId + " = ?", taskModelId).
			Updates(map[string]interface{}{
				DbFieldLastFailedAt: now,
			}).Error
		if err != nil {
			return err
		}
		taskStatus = statusFailed
	}

	err = conn.Model(&TaskLogModel{}).
		Where(DbFieldTaskId + " = ?", taskModelId).
		Where(DbFieldRunTimes, resp.GetTaskRunTimes()).
		Updates(map[string]interface{}{
			DbFieldIsRunInAsync: resp.IsRunInAsync(),
			DbFieldTaskStatus: taskStatus,
		}).Error
	if err != nil {
		return err
	}

	return nil
}

func (r *TaskRepo) ConfirmTasks(ctx context.Context, resps []*task.TaskResp) error {
	var (
		successTaskModelIds, failedTaskModelIds []uint64
		runTimesToRunningAsyncTaskModelIdsMap = make(map[int][]uint64)
		runTimesToSuccAsyncTaskModelIdsMap, runTimesToSuccSyncTaskModelIdsMap = make(map[int][]uint64), make(map[int][]uint64)
		runTimesToFailAsyncTaskModelIdsMap, runTimesToFailSyncTaskModelIdsMap =  make(map[int][]uint64), make(map[int][]uint64)
	)
	for _, resp := range resps {
		taskModelId, err := toTaskModelId(resp.GetTaskId())
		if err != nil {
			return err
		}


		taskRunTimes := resp.GetTaskRunTimes()

		if task.IsTaskSuccess(resp.GetTaskStatus()) {
			successTaskModelIds = append(successTaskModelIds, taskModelId)
			if resp.IsRunInAsync() {
				runTimesToSuccAsyncTaskModelIdsMap[taskRunTimes] = append(runTimesToSuccAsyncTaskModelIdsMap[taskRunTimes], taskModelId)
				continue
			}
			runTimesToSuccSyncTaskModelIdsMap[taskRunTimes] = append(runTimesToSuccSyncTaskModelIdsMap[taskRunTimes], taskModelId)
			continue
		}

		if task.IsTaskFailed(resp.GetTaskStatus()) {
			failedTaskModelIds = append(failedTaskModelIds, taskModelId)
			if resp.IsRunInAsync() {
				runTimesToFailAsyncTaskModelIdsMap[taskRunTimes] = append(runTimesToFailAsyncTaskModelIdsMap[taskRunTimes], taskModelId)
				continue
			}
			runTimesToFailSyncTaskModelIdsMap[taskRunTimes] = append(runTimesToFailSyncTaskModelIdsMap[taskRunTimes], taskModelId)
			continue
		}

		if task.IsTaskRunning(resp.GetTaskStatus()) {
			if !resp.IsRunInAsync() {
				continue
			}
			runTimesToRunningAsyncTaskModelIdsMap[taskRunTimes] = append(runTimesToRunningAsyncTaskModelIdsMap[taskRunTimes], taskModelId)
			continue
		}
	}

	now := time.Now().Unix()
	conn := r.mustGetConn().WithContext(ctx)
	if len(successTaskModelIds) > 0 {
		err := conn.Where(DbFieldId + " IN ?", successTaskModelIds).
			Updates(map[string]interface{}{
				DbFieldLastSuccessAt: now,
			}).Error
		if err != nil {
			return err
		}
	}

	if len(failedTaskModelIds) > 0 {
		err := conn.Where(DbFieldId + " IN ?", failedTaskModelIds).
			Updates(map[string]interface{}{
				DbFieldLastFailedAt: now,
			}).Error
		if err != nil {
			return err
		}
	}

	for taskRunTimes, taskModelIds := range runTimesToRunningAsyncTaskModelIdsMap {
		err := conn.Model(&TaskLogModel{}).
			Where(DbFieldTaskId + " IN ?", taskModelIds).
			Where(DbFieldRunTimes, taskRunTimes).
			Updates(map[string]interface{}{
				DbFieldIsRunInAsync: true,
			}).Error
		if err != nil {
			return err
		}
	}

	for taskRunTimes, taskModelIds := range runTimesToSuccAsyncTaskModelIdsMap {
		err := conn.Model(&TaskLogModel{}).
			Where(DbFieldTaskId + " IN ?", taskModelIds).
			Where(DbFieldRunTimes, taskRunTimes).
			Updates(map[string]interface{}{
				DbFieldIsRunInAsync: true,
				DbFieldTaskStatus: statusSuccess,
			}).Error
		if err != nil {
			return err
		}
	}

	for taskRunTimes, taskModelIds := range runTimesToFailAsyncTaskModelIdsMap {
		err := conn.Model(&TaskLogModel{}).
			Where(DbFieldTaskId + " IN ?", taskModelIds).
			Where(DbFieldRunTimes, taskRunTimes).
			Updates(map[string]interface{}{
				DbFieldIsRunInAsync: true,
				DbFieldTaskStatus: statusFailed,
			}).Error
		if err != nil {
			return err
		}
	}

	return nil
}

func NewTaskRepo(connDsn string, srvRepo repo.TaskCallbackSrvRepo) (*TaskRepo, error) {
	taskRepo := &TaskRepo{
		srvRepo: srvRepo,
		repoConnector: repoConnector{
			connDsn: connDsn,
		},
	}

	if err := taskRepo.mustGetConn().AutoMigrate(&TaskModel{}); err != nil {
		return nil, err
	}

	return taskRepo, nil
}

var _ repo.TaskRepo = (*TaskRepo)(nil)
