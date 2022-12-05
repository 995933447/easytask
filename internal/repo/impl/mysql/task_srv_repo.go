package mysql

import (
	"context"
	"github.com/995933447/easytask/internal/repo"
	"github.com/995933447/easytask/internal/task"
	"github.com/995933447/easytask/internal/util/logger"
	"github.com/995933447/optionstream"
	"github.com/995933447/reflectutil"
	"gorm.io/gorm"
	"sync/atomic"
	"time"
)

var migratedTaskSrvRepoDB atomic.Bool

type TaskSrvRepo struct {
	repoConnector
}

func (r *TaskSrvRepo) AddSrvRoutes(ctx context.Context, srv *task.TaskCallbackSrv) error {
	log := logger.MustGetSysLogger()
	conn := r.mustGetConn(ctx)
	var srvModel TaskCallbackSrvModel
	if err := conn.Where(DbFieldName + " = ?", srv.GetName()).Take(&srvModel).Error; err != nil {
		if err != gorm.ErrRecordNotFound {
			log.Error(ctx, err)
			return err
		}
		srvModel.Name = srv.GetName()
		if err = conn.Create(&srv).Error; err != nil {
			log.Error(ctx, err)
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
			log.Error(ctx, res.Error)
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
			log.Error(ctx, err)
			return err
		}
	}

	if hasEnableHealthCheck {
		err := conn.Model(&TaskCallbackSrvModel{}).
			Where(DbFieldId, srvModel.Id).
			Update(DbFieldHasEnableHealthCheck, true).
			Error
		if err != nil {
			log.Error(ctx, err)
			return err
		}
	}

	return nil
}

func(r *TaskSrvRepo) DelSrvRoutes(ctx context.Context, srv *task.TaskCallbackSrv) error {
	var (
		srvModel TaskCallbackSrvModel
		conn = r.mustGetConn(ctx)
		log = logger.MustGetSysLogger()
	)
	err := conn.Where(DbFieldName + " = ?", srv.GetName()).Take(&srvModel).Error
	if err != nil {
		return err
	}
	var routeModelIds []uint64
	for _, route := range srv.GetRoutes() {
		routeModelId, err := toCallbackSrvRouteModelId(route.GetId())
		if err != nil {
			log.Error(ctx, err)
			return err
		}
		routeModelIds = append(routeModelIds, routeModelId)
	}
	err = r.mustGetConn(ctx).
		Where(DbFieldSrvId + " = ?", srvModel.Id).
		Where(DbFieldId + " IN ?", routeModelIds).
		Delete(&TaskCallbackSrvRouteModel{}).
		Error
	if err != nil {
		log.Error(ctx, err)
		return err
	}
	return nil
}

func (r *TaskSrvRepo) SetSrvRoutesPassHealthCheck(ctx context.Context, srv *task.TaskCallbackSrv) error {
	var (
		log = logger.MustGetSysLogger()
		srvModel TaskCallbackSrvModel
	)
	if len(srv.GetRoutes()) == 0 {
		return nil
	}

	conn := r.mustGetConn(ctx)
	if err := conn.Where(DbFieldName + " = ?", srv.GetName()).Take(&srvModel).Error; err != nil {
		log.Error(ctx, err)
		return err
	}

	var routeModelIds []uint64
	for _, route := range srv.GetRoutes() {
		routeModelId, err := toTackCallbackSrvRouteModelId(route.GetId())
		if err != nil {
			log.Error(ctx, err)
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
		log.Error(ctx, res.Error)
		return res.Error
	}

	res = conn.Model(&TaskCallbackSrvModel{}).
		Where(DbFieldId, srvModel.Id).
		Updates(map[string]interface{}{
			DbFieldCheckedHealthAt: now,
		})
	if res.Error != nil {
		log.Error(ctx, res.Error)
		return res.Error
	}
	return nil
}

func (r *TaskSrvRepo) GetSrvsByIds(ctx context.Context, ids []string) ([]*task.TaskCallbackSrv, error) {
	srvs, err := r.GetSrvs(
		ctx,
		optionstream.NewQueryStream(nil, int64(len(ids)), 0).SetOption(repo.QueryOptKeyInIds, ids),
		)
	if err != nil {
		logger.MustGetSysLogger().Error(ctx, err)
		return nil, err
	}
	return srvs, nil
}

func (r *TaskSrvRepo) GetSrvs(ctx context.Context, queryStream *optionstream.QueryStream) ([]*task.TaskCallbackSrv, error) {
	log := logger.MustGetSysLogger()
	conn :=  r.mustGetConn(ctx)
	err := optionstream.NewQueryStreamProcessor(queryStream).
		OnStringList(repo.QueryOptKeyInIds, func(val []string) error {
			var srvModelIds []uint64
			for _, id := range val {
				srvModelId, err := toTackCallbackSrvRouteModelId(id)
				if err != nil {
					log.Error(ctx, err)
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
		log.Error(ctx, err)
		return nil, err
	}

	var srvModels []*TaskCallbackSrvModel
	err = conn.Scan(&srvModels).Error
	if err != nil {
		log.Error(ctx, err)
		return nil, err
	}

	srvModelIds := reflectutil.PluckUint64(srvModels, FieldId)

	var routeModels []*TaskCallbackSrvRouteModel
	err = conn.Where(DbFieldSrvId + " IN ?", srvModelIds).Scan(&routeModels).Error
	if err != nil {
		log.Error(ctx, err)
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

func NewTaskSrvRepo(ctx context.Context, connDsn string) (*TaskSrvRepo, error) {
	srvRepo := &TaskSrvRepo{
		repoConnector: repoConnector{
			connDsn: connDsn,
		},
	}
	if !migratedTaskSrvRepoDB.Load() {
		if err := srvRepo.mustGetConn(ctx).AutoMigrate(&TaskCallbackSrvModel{}, &TaskCallbackSrvRouteModel{}); err != nil {
			logger.MustGetSysLogger().Error(ctx, err)
			return nil, err
		}
	}
	return srvRepo, nil
}