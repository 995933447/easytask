package mysql

import (
	"context"
	"github.com/995933447/easytask/internal/repo"
	"github.com/995933447/easytask/internal/task"
	"github.com/995933447/reflectutil"
	"gorm.io/gorm"
	"time"
)

type TaskRepo struct {
	srvRepo repo.TaskCallbackSrvRepo
	repoConnector
}

func (r *TaskRepo) TimeoutTasks(ctx context.Context, size int) ([]*task.Task, error) {
	var (
		taskModels []*TaskModel
	)
	err := r.mustGetConn().
		WithContext(ctx).
		Where(DbFieldNextRunAt + " <= ?", time.Now().Unix()).
		Where(DbFieldTriedCnt + " < ?", DbFieldMaxTryCnt).
		Where(DbFieldStatus + " IN ?", []int{statusFailed, statusNil}).
		Limit(size).
		Scan(&taskModels).
		Error
	if err != nil {
		return nil, err
	}

	callbackSrvModelIds := reflectutil.PluckUint64(taskModels, FieldCallbackSrvId)
	var callbackSrvIds []string
	for _, callbackSrvModelId := range callbackSrvModelIds {
		callbackSrvIds = append(callbackSrvIds, toCallbackSrvRouteEntityId(callbackSrvModelId))
	}
	callbackSrvs, err := r.srvRepo.GetSrvsByIds(ctx, callbackSrvIds)
	if err != nil {
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

func (r *TaskRepo) LockTask(ctx context.Context, task *task.Task) (bool, error) {
	res := r.mustGetConn().
		WithContext(ctx).
		Where(DbFieldId + " = ?", toTaskModelId(task.GetId())).
		Where(DbFieldTriedCnt + " = ?", task.GetTriedCnt()).
		Where(DbFieldLastRunAt + " = ?", task.GetLastRunAt()).
		Updates(map[string]interface{}{
			DbFieldStatus: statusRunning,
			DbFieldLastRunAt: time.Now().Unix(),
			DbFieldTriedCnt: gorm.Expr(DbFieldTriedCnt + " + 1"),
		})
	if res.Error != nil {
		return false, res.Error
	}

	return res.RowsAffected > 0, nil
}

func (r *TaskRepo) ConfirmTasks(ctx context.Context, resps []*task.TaskResp) error {
	var successTaskModelIds, failedTaskModelIds []uint64
	for _, resp := range resps {
		if task.IsTaskSuccess(resp.GetTaskStatus()) {
			taskModelId, err := toTaskModelId(resp.GetTaskId())
			if err != nil {
				return err
			}
			successTaskModelIds = append(successTaskModelIds, taskModelId)
			continue
		}

		if task.IsTaskFailed(resp.GetTaskStatus()) {
			taskModelId, err := toTaskModelId(resp.GetTaskId())
			if err != nil {
				return err
			}
			failedTaskModelIds = append(failedTaskModelIds, taskModelId)
		}
	}

	conn := r.mustGetConn().WithContext(ctx)
	if len(successTaskModelIds) > 0 {
		err := conn.Where(DbFieldId + " IN ?", successTaskModelIds).
			Updates(map[string]interface{}{
				DbFieldStatus: statusSuccess,
			}).Error
		if err != nil {
			return err
		}
	}

	if len(failedTaskModelIds) > 0 {
		err := conn.Where(DbFieldId + " IN ?", failedTaskModelIds).
			Updates(map[string]interface{}{
				DbFieldStatus: statusFailed,
				DbFieldLastFailedAt: time.Now().Unix(),
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
