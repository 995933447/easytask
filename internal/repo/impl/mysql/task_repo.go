package mysql

import (
	"context"
	"github.com/995933447/easytask/internal/repo"
	"github.com/995933447/easytask/internal/task"
	"github.com/995933447/easytask/internal/util/logger"
	"github.com/995933447/reflectutil"
	"gorm.io/gorm"
	"math"
	"sync/atomic"
	"time"
)

var migratedTaskRepoDB atomic.Bool

type TaskRepo struct {
	srvRepo repo.TaskCallbackSrvRepo
	repoConnector
	defaultTaskMaxRunTimeSec int64
}

func (r *TaskRepo) DelTaskById(ctx context.Context, id string) error {
	log := logger.MustGetSysLogger()
	modelId, err := toTaskModelId(id)
	if err != nil {
		log.Error(ctx, err)
		return err
	}
	if err = r.mustGetConn(ctx).Delete(&TaskModel{}, modelId).Error; err != nil {
		log.Error(ctx, err)
		return err
	}
	return nil
}

func (r *TaskRepo) GetTaskById(ctx context.Context, id string) (*task.Task, error) {
	log := logger.MustGetSysLogger()
	var taskModel TaskModel
	if err := r.mustGetConn(ctx).Where(DbFieldId, toTaskModelId(id)).Take(&taskModel).Error; err != nil {
		log.Error(ctx, err)
		return nil, err
	}

	srvId := toTaskCallbackSrvEntityId(taskModel.CallbackSrvId)
	srvs, err := r.srvRepo.GetSrvsByIds(ctx, []string{srvId})
	if err != nil {
		log.Error(ctx, err)
		return nil, err
	}
	if len(srvs) == 0 {
		log.Warnf(ctx, "get TaskCallbackSrv(id:%s) is empty", srvId)
		return nil, gorm.ErrRecordNotFound
	}

	srv := srvs[0]

	schedMode, err := taskModel.toEntitySchedMode()
	if err != nil {
		log.Error(ctx, err)
		return nil, err
	}

	var timeSpecAt int64
	if taskModel.SchedMode == schedModeTimeSpec {
		timeSpecAt = taskModel.PlanSchedNextAt
	}

	oneTask, err := task.NewTask(&task.NewTaskReq{
		Id: taskModel.toEntityId(),
		CallbackSrv: srv,
		CallbackPath: taskModel.CallbackPath,
		Name: taskModel.Name,
		Arg: taskModel.Arg,
		RunTimes: taskModel.RunTimes,
		LastRunAt: taskModel.LastRunAt,
		AllowMaxRunTimes: taskModel.AllowMaxRunTimes,
		SchedMode: schedMode,
		TimeCronExpr: taskModel.TimeCronExpr,
		TimeIntervalSec: taskModel.TimeIntervalSec,
		TimeSpecAt: timeSpecAt,
	})
	if err != nil {
		log.Error(ctx, err)
		return nil, err
	}

	return oneTask, nil
}

func (r *TaskRepo) AddTask(ctx context.Context, oneTask *task.Task) (string, error) {
	log := logger.MustGetSysLogger()
	conn := r.mustGetConn(ctx)

	srvId, err := toCallbackSrvRouteModelId(oneTask.GetCallbackSrv().GetId())
	if err != nil {
		log.Error(ctx, err)
		return "", err
	}

	schedModel, err := toTaskModelSchedMode(oneTask.GetSchedMode())
	if err != nil {
		log.Error(ctx, err)
		return "", err
	}

	schedNextAt, err := oneTask.GetSchedNextAt()
	if err != nil {
		log.Error(ctx, err)
		return "", nil
	}

	var allowMaxRunTimes int
	switch oneTask.GetSchedMode() {
	case task.SchedModeTimeSpec:
		allowMaxRunTimes = 1
	case task.SchedModeTimeCron, task.SchedModeTimeInterval:
		allowMaxRunTimes = math.MaxInt
	}

	taskModel := &TaskModel{
		Name: oneTask.GetName(),
		Arg: oneTask.GetArg(),
		Status: statusReady,
		SchedMode: schedModel,
		TimeCronExpr: oneTask.GetTimeCronExpr(),
		TimeIntervalSec: oneTask.GetTimeIntervalSec(),
		PlanSchedNextAt: schedNextAt,
		AllowMaxRunTimes: allowMaxRunTimes,
		CallbackPath: oneTask.GetCallbackPath(),
		CallbackSrvId: srvId,
	}
	err = conn.Create(taskModel).Error
	if err != nil {
		log.Error(ctx, err)
		return "", nil
	}

	return taskModel.toEntityId(), nil
}

func (r *TaskRepo) TimeoutTasks(ctx context.Context, size int) ([]*task.Task, error) {
	log := logger.MustGetSysLogger()
	
	var taskModels []*TaskModel
	err := r.mustGetConn(ctx).
		WithContext(ctx).
		Where(DbFieldRunTimes + " < " + DbFieldAllowMaxRunTimes).
		Where(DbFieldPlanSchedNextAt + " <= ?", time.Now().Unix()).
		Limit(size).
		Scan(&taskModels).
		Error
	if err != nil {
		log.Error(ctx, err)
		return nil, err
	}

	callbackSrvModelIds := reflectutil.PluckUint64(taskModels, FieldCallbackSrvId)
	var callbackSrvIds []string
	for _, callbackSrvModelId := range callbackSrvModelIds {
		callbackSrvIds = append(callbackSrvIds, toCallbackSrvRouteEntityId(callbackSrvModelId))
	}
	callbackSrvs, err := r.srvRepo.GetSrvsByIds(ctx, callbackSrvIds)
	if err != nil {
		log.Error(ctx, err)
		return nil, err
	}

	callbackSrvMap := reflectutil.MapByKey(callbackSrvs, FieldId).(map[uint64]*task.TaskCallbackSrv)

	var tasks []*task.Task
	for _, taskModel := range taskModels {
		callbackSrv, ok := callbackSrvMap[taskModel.CallbackSrvId]
		if !ok {
			continue
		}
		schedMode, err := taskModel.toEntitySchedMode()
		if err != nil {
			log.Error(ctx, err)
			return nil, err
		}
		var timeSpecAt int64
		if taskModel.SchedMode == schedModeTimeSpec {
			timeSpecAt = taskModel.PlanSchedNextAt
		}
		oneTask, err := task.NewTask(&task.NewTaskReq{
			Id: taskModel.toEntityId(),
			CallbackSrv: callbackSrv,
			CallbackPath: taskModel.CallbackPath,
			Name: taskModel.Name,
			Arg: taskModel.Arg,
			RunTimes: taskModel.RunTimes,
			LastRunAt: taskModel.LastRunAt,
			AllowMaxRunTimes: taskModel.AllowMaxRunTimes,
			SchedMode: schedMode,
			TimeSpecAt: timeSpecAt,
			TimeIntervalSec: taskModel.TimeIntervalSec,
			TimeCronExpr: taskModel.TimeCronExpr,
		})
		if err != nil {
			log.Error(ctx, err)
			return nil, err
		}
		tasks = append(tasks, oneTask)
	}

	return nil, nil
}

func (r *TaskRepo) LockTask(ctx context.Context, oneTask *task.Task) (bool, error) {
	var (
		log = logger.MustGetSysLogger()
		now = time.Now().Unix()
	)

	schedNextAt, err := oneTask.GetSchedNextAt()
	if err != nil {
		return false, err
	}

	taskModelUpdates := map[string]interface{}{
		DbFieldLastRunAt: now,
		DbFieldRunTimes: gorm.Expr(DbFieldRunTimes + " + 1"),
		DbFieldPlanSchedNextAt: schedNextAt,
	}

	taskModelId, err := toTaskModelId(oneTask.GetId())
	if err != nil {
		log.Error(ctx, err)
		return false, err
	}
	conn := r.mustGetConn(ctx)
	res := conn.WithContext(ctx).
		Model(&TaskModel{}).
		Where(DbFieldId + " = ?", taskModelId).
		Where(DbFieldLastRunAt + " = ?", oneTask.GetLastRunAt()).
		Where(DbFieldRunTimes + " = ?", oneTask.GetRunTimes()).
		Updates(taskModelUpdates)
	if res.Error != nil {
		log.Error(ctx, res.Error)
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
			log.Error(ctx, err)
			return false, err
		}
	}

	return false, nil
}

func (r *TaskRepo) ConfirmTask(ctx context.Context, resp *task.TaskResp) error {
	now := time.Now().Unix()
	conn := r.mustGetConn(ctx)
	taskModelId, err := toTaskModelId(resp.GetTaskId())
	taskStatus := statusRunning
	log := logger.MustGetSysLogger()

	if task.IsTaskSuccess(resp.GetTaskStatus()) {
		err = conn.Model(&TaskModel{}).
			Where(DbFieldId + " = ?", taskModelId).
			Updates(map[string]interface{}{
				DbFieldLastSuccessAt: now,
			}).Error
		if err != nil {
			log.Error(ctx, err)
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
			log.Error(ctx, err)
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
		log.Error(ctx, err)
		return err
	}

	return nil
}

func (r *TaskRepo) ConfirmTasks(ctx context.Context, resps []*task.TaskResp) error {
	log := logger.MustGetSysLogger()

	var (
		successTaskModelIds, failedTaskModelIds []uint64
		runTimesToRunningAsyncTaskModelIdsMap = make(map[int][]uint64)
		runTimesToSuccAsyncTaskModelIdsMap, runTimesToSuccSyncTaskModelIdsMap = make(map[int][]uint64), make(map[int][]uint64)
		runTimesToFailAsyncTaskModelIdsMap, runTimesToFailSyncTaskModelIdsMap =  make(map[int][]uint64), make(map[int][]uint64)
	)
	for _, resp := range resps {
		taskModelId, err := toTaskModelId(resp.GetTaskId())
		if err != nil {
			log.Error(ctx, err)
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
	conn := r.mustGetConn(ctx)
	if len(successTaskModelIds) > 0 {
		err := conn.Where(DbFieldId + " IN ?", successTaskModelIds).
			Updates(map[string]interface{}{
				DbFieldLastSuccessAt: now,
			}).Error
		if err != nil {
			log.Error(ctx, err)
			return err
		}
	}

	if len(failedTaskModelIds) > 0 {
		err := conn.Where(DbFieldId + " IN ?", failedTaskModelIds).
			Updates(map[string]interface{}{
				DbFieldLastFailedAt: now,
			}).Error
		if err != nil {
			log.Error(ctx, err)
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
			log.Error(ctx, err)
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
			log.Error(ctx, err)
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
			log.Error(ctx, err)
			return err
		}
	}

	return nil
}

func NewTaskRepo(ctx context.Context, connDsn string, srvRepo repo.TaskCallbackSrvRepo) (*TaskRepo, error) {
	taskRepo := &TaskRepo{
		srvRepo: srvRepo,
		repoConnector: repoConnector{
			connDsn: connDsn,
		},
	}

	if !migratedTaskRepoDB.Load() {
		if err := taskRepo.mustGetConn(ctx).AutoMigrate(&TaskModel{}); err != nil {
			logger.MustGetSysLogger().Error(ctx, err)
			return nil, err
		}
	}

	return taskRepo, nil
}

var _ repo.TaskRepo = (*TaskRepo)(nil)
