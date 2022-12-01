package mysql

import (
	"context"
	"github.com/995933447/easytask/internal/repo"
	"github.com/995933447/easytask/internal/task"
	"github.com/995933447/optionstream"
	"github.com/995933447/reflectutil"
	"gorm.io/gorm"
	"time"
)

type TaskSrvRepo struct {
	repoConnector
}

func (r *TaskSrvRepo) AddSrvRoutes(ctx context.Context, srv *task.TaskCallbackSrv) error {
	conn := r.mustGetConn().WithContext(ctx)
	var srvModel TaskCallbackSrvModel
	if err := conn.Where(DbFieldName + " = ?", srv.GetName()).Take(&srvModel).Error; err != nil {
		if err != gorm.ErrRecordNotFound {
			return err
		}
		srvModel.Name = srv.GetName()
		if err = conn.Create(&srv).Error; err != nil {
			return err
		}
	}

	var hasEnableHealthCheck bool
	for _, route := range srv.GetRoutes() {
		routeModel := TaskCallbackSrvRouteModel{
			Schema: route.GetSchema(),
			Host: route.GetHost(),
			Port: route.GetPort(),
			SrvId: srvModel.Id,
			CallbackTimeoutSec: route.GetCallbackTimeoutSec(),
			EnableHealthCheck: route.IsEnableHeathCheck(),
		}
		res := conn.Unscoped().
			Select(DbFieldId + " = ?", DbFieldCallbackTimeoutSec).
			Where(DbFieldSrvId + " = ?", srvModel.Id).
			Where(DbFieldHost + " = ?", route.GetHost()).
			Where(DbFieldPort + " = ?", route.GetPort()).
			Where(DbFieldSchema + " = ?", route.GetSchema()).
			FirstOrCreate(&routeModel)
		if res.Error != nil {
			return res.Error
		}

		if res.RowsAffected > 0 {
			continue
		}

		if route.IsEnableHeathCheck() && !hasEnableHealthCheck {
			hasEnableHealthCheck = true
		}

		updateMap := map[string]interface{}{
			DbFieldDeletedAt: 0,
			DbFieldEnableHealthCheck: route.IsEnableHeathCheck(),
		}
		if route.GetCallbackTimeoutSec() != routeModel.CallbackTimeoutSec {
			updateMap[DbFieldCallbackTimeoutSec] = route.GetCallbackTimeoutSec()
		}
		err := conn.Model(&TaskCallbackSrvRouteModel{}).
			Where(DbFieldId + " = ?", routeModel.Id).
			Updates(updateMap).
			Error
		if err != nil {
			return err
		}
	}

	if hasEnableHealthCheck {
		err := conn.Model(&TaskCallbackSrvModel{}).
			Where(DbFieldId, srvModel.Id).
			Update(DbFieldHasEnableHealthCheck, true).
			Error
		if err != nil {
			return err
		}
	}

	return nil
}

func(r *TaskSrvRepo)  DelSrvRoutes(ctx context.Context, srv *task.TaskCallbackSrv) error {
	var srvModel TaskCallbackSrvModel
	err := r.mustGetConn().WithContext(ctx).Where(DbFieldName + " = ?", srv.GetName()).Take(&srvModel).Error
	if err != nil {
		return err
	}
	return nil
}

func (r *TaskSrvRepo) SetSrvRoutesPassHealthCheck(ctx context.Context, srv *task.TaskCallbackSrv) error {
	var srvModel TaskCallbackSrvModel
	if len(srv.GetRoutes()) == 0 {
		return nil
	}

	conn := r.mustGetConn().WithContext(ctx)
	if err := conn.Where(DbFieldName + " = ?", srv.GetName()).Take(&srvModel).Error; err != nil {
		return err
	}

	var routeModelIds []uint64
	for _, route := range srv.GetRoutes() {
		routeModelId, err := toTackCallbackSrvRouteModelId(route.GetId())
		if err != nil {
			return err
		}
		routeModelIds = append(routeModelIds, routeModelId)
	}
	now := time.Now().Unix()
	res := conn.Model(&TaskCallbackSrvRouteModel{}).
		Where(DbFieldSrvId + " = ?", srvModel.Id).
		Where(DbFieldId + " IN ?", routeModelIds).
		Updates(map[string]interface{}{
			DbFieldCheckedHealthAt: now,
		})
	if res.Error != nil {
		return res.Error
	}

	res = conn.Model(&TaskCallbackSrvModel{}).
		Where(DbFieldId, srvModel.Id).
		Updates(map[string]interface{}{
			DbFieldCheckedHealthAt: now,
		})
	if res.Error != nil {
		return res.Error
	}
	return nil
}

func (r *TaskSrvRepo) GetSrvsByIds(ctx context.Context, ids []string) ([]*task.TaskCallbackSrv, error) {
	return r.GetSrvs(
		ctx,
		optionstream.NewQueryStream(nil, int64(len(ids)), 0).SetOption(repo.QueryOptKeyInIds, ids),
		)
}

func (r *TaskSrvRepo) GetSrvs(ctx context.Context, queryStream *optionstream.QueryStream) ([]*task.TaskCallbackSrv, error) {
	conn :=  r.mustGetConn().WithContext(ctx)
	err := optionstream.NewQueryStreamProcessor(queryStream).
		OnStringList(repo.QueryOptKeyInIds, func(val []string) error {
			var srvModelIds []uint64
			for _, id := range val {
				srvModelId, err := toTackCallbackSrvRouteModelId(id)
				if err != nil {
					return err
				}
				srvModelIds = append(srvModelIds, srvModelId)
			}
			conn.Where(DbFieldId + " IN ?", srvModelIds)
			return nil
		}).
		OnBool(repo.QueryOptKeyEnabledHeathCheck, func(val bool) error {
			conn.Where(DbFieldHasEnableHealthCheck + " = 1")
			return nil
		}).
		OnInt64(repo.QueryOptKeyCheckedHealthLt, func(val int64) error {
			conn.Where(DbFieldCheckedHealthAt + " < ?", val)
			return nil
		}).
		Process()
	if err != nil {
		return nil, err
	}

	var srvModels []*TaskCallbackSrvModel
	err = conn.Scan(&srvModels).Error
	if err != nil {
		return nil, err
	}

	srvModelIds := reflectutil.PluckUint64(srvModels, FieldId)

	var routeModels []*TaskCallbackSrvRouteModel
	err = conn.Where(DbFieldSrvId + " IN ?", srvModelIds).Scan(&routeModels).Error
	if err != nil {
		return nil, err
	}

	srvIdToRoutesMap := make(map[uint64][]*TaskCallbackSrvRouteModel)
	for _, routeModel := range routeModels {
		srvIdToRoutesMap[routeModel.SrvId] = append(srvIdToRoutesMap[routeModel.SrvId], routeModel)
	}

	var srvs []*task.TaskCallbackSrv
	for _, srvModel := range srvModels {
		routeModels := srvIdToRoutesMap[srvModel.Id]
		var routes []*task.TaskCallbackSrvRoute
		for _, routeModel := range routeModels {
			routes = append(routes, routeModel.toEntity())
		}
		srvs = append(srvs, srvModel.toEntity(routes))
	}

	return srvs, nil
}

func NewTaskSrvRepo(connDsn string) (*TaskSrvRepo, error) {
	srvRepo := &TaskSrvRepo{
		repoConnector: repoConnector{
			connDsn: connDsn,
		},
	}
	if err := srvRepo.mustGetConn().AutoMigrate(&TaskCallbackSrvModel{}, &TaskCallbackSrvRouteModel{}); err != nil {
		return nil, err
	}
	return srvRepo, nil
}