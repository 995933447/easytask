package mysql

import (
	"context"
	"github.com/995933447/easytask/internal/task"
	"github.com/995933447/easytask/internal/util/logger"
	"github.com/995933447/optionstream"
	"github.com/995933447/reflectutil"
	"sync/atomic"
	"time"
)

var migratedTaskSrvRepoDB atomic.Bool

type TaskSrvRepo struct {
	repoConnector
}

func (r *TaskSrvRepo) AddSrvRoutes(ctx context.Context, srv *task.TaskCallbackSrv) error {
	log := logger.MustGetRepoLogger()
	conn := r.mustGetConn(ctx)
	var srvModel TaskCallbackSrvModel
	srvModel.Name = srv.GetName()
	res := conn.Unscoped().Where(DbFieldName + " = ?", srv.GetName()).FirstOrCreate(&srvModel)
	if res.Error != nil {
		log.Error(ctx, res.Error)
		return res.Error
	}

	if res.RowsAffected == 0 && srvModel.DeletedAt > 0 {
		err := conn.Unscoped().Model(&srvModel).Where(DbFieldId + " = ?", srvModel.Id).Update(DbFieldDeletedAt, 0).Error
		if err != nil {
			log.Error(ctx, err)
			return err
		}
	}

	var hasEnableHealthCheck bool
	for _, route := range srv.GetRoutes() {
		if route.IsEnableHeathCheck() && !hasEnableHealthCheck {
			hasEnableHealthCheck = true
		}

		routeModel := TaskCallbackSrvRouteModel{
			SrvSchema: route.GetSchema(),
			Host: route.GetHost(),
			Port: route.GetPort(),
			SrvId: srvModel.Id,
			CallbackTimeoutSec: route.GetCallbackTimeoutSec(),
			EnableHealthCheck: route.IsEnableHeathCheck(),
		}
		res := conn.Unscoped().
			Where(DbFieldSrvId + " = ?", srvModel.Id).
			Where(DbFieldHost + " = ?", route.GetHost()).
			Where(DbFieldPort + " = ?", route.GetPort()).
			Where(DbFieldSrvSchema + " = ?", route.GetSchema()).
			FirstOrCreate(&routeModel)
		if res.Error != nil {
			log.Error(ctx, res.Error)
			return res.Error
		}

		if res.RowsAffected > 0 {
			continue
		}

		updateMap := map[string]interface{}{
			DbFieldDeletedAt:         0,
			DbFieldEnableHealthCheck: route.IsEnableHeathCheck(),
		}
		if route.GetCallbackTimeoutSec() != routeModel.CallbackTimeoutSec {
			updateMap[DbFieldCallbackTimeoutSec] = route.GetCallbackTimeoutSec()
		}
		err := conn.Unscoped().Model(&TaskCallbackSrvRouteModel{}).
			Where(DbFieldId + " = ?", routeModel.Id).
			Updates(updateMap).
			Error
		if err != nil {
			log.Error(ctx, err)
			return err
		}
	}

	if hasEnableHealthCheck && !srvModel.HasEnableHealthCheck {
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
		conn     = r.mustGetConn(ctx)
		log      = logger.MustGetRepoLogger()
	)
	err := conn.Where(DbFieldName+ " = ?", srv.GetName()).Take(&srvModel).Error
	if err != nil {
		log.Error(ctx, err)
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
	err = conn.Where(DbFieldSrvId + " = ?", srvModel.Id).
		Where(DbFieldId + " IN ?", routeModelIds).
		Delete(&TaskCallbackSrvRouteModel{}).
		Error
	if err != nil {
		log.Error(ctx, err)
		return err
	}

	type RouteNums struct {
		Total int64 `json:"total"`
		EnabledCheckHealthNum int64 `json:"enabled_check_health_num"`
	}
	routeNums := &RouteNums{}
	err = conn.Model(&TaskCallbackSrvRouteModel{}).
		Select("COUNT(1) AS total, COUNT(IF(" + DbFieldEnableHealthCheck+ " > 0, 1, 0)) AS enabled_check_health_num").
		Where(DbFieldSrvId + " = ?", srvModel.Id).
		Take(&routeNums).
		Error
	if err != nil {
		log.Error(ctx, err)
		return err
	}

	if routeNums.Total == 0 {
		if err = conn.Where(DbFieldId + " = ?", srvModel.Id).Delete(&TaskCallbackSrvModel{}).Error; err != nil {
			log.Error(ctx, err)
			return err
		}
		return nil
	}

	if routeNums.EnabledCheckHealthNum == 0 && srvModel.HasEnableHealthCheck {
		err = conn.Model(&TaskCallbackSrvModel{}).
			Where(DbFieldId + " = ?", srvModel.Id).
			Update(DbFieldHasEnableHealthCheck, false).
			Error
		if err != nil {
			log.Error(ctx, err)
			return err
		}
	}

	return nil
}

func (r *TaskSrvRepo) SetSrvRoutesPassHealthCheck(ctx context.Context, srv *task.TaskCallbackSrv) error {
	var (
		log      = logger.MustGetRepoLogger()
		srvModel TaskCallbackSrvModel
	)
	if len(srv.GetRoutes()) == 0 {
		return nil
	}

	conn := r.mustGetConn(ctx)
	if err := conn.Where(DbFieldName+ " = ?", srv.GetName()).Take(&srvModel).Error; err != nil {
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
		Where(DbFieldSrvId+ " = ?", srvModel.Id).
		Where(DbFieldId+ " IN ?", routeModelIds).
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
		optionstream.NewQueryStream(nil, int64(len(ids)), 0).SetOption(task.QueryOptKeyInIds, ids),
		)
	if err != nil {
		logger.MustGetRepoLogger().Error(ctx, err)
		return nil, err
	}
	return srvs, nil
}

func (r *TaskSrvRepo) GetSrvs(ctx context.Context, queryStream *optionstream.QueryStream) ([]*task.TaskCallbackSrv, error) {
	log := logger.MustGetRepoLogger()
	conn :=  r.mustGetConn(ctx)
	srvQueryScope := conn
	err := optionstream.NewQueryStreamProcessor(queryStream).
		OnStringList(task.QueryOptKeyInIds, func(val []string) error {
			var srvModelIds []uint64
			for _, id := range val {
				srvModelId, err := toTackCallbackSrvRouteModelId(id)
				if err != nil {
					log.Error(ctx, err)
					return err
				}
				srvModelIds = append(srvModelIds, srvModelId)
			}
			srvQueryScope = srvQueryScope.Where(DbFieldId + " IN ?", srvModelIds)
			return nil
		}).
		OnNone(task.QueryOptKeyEnabledHeathCheck, func() error {
			srvQueryScope = srvQueryScope.Where(DbFieldHasEnableHealthCheck + " = 1")
			return nil
		}).
		OnInt64(task.QueryOptKeyCheckedHealthLt, func(val int64) error {
			srvQueryScope = srvQueryScope.Where(DbFieldCheckedHealthAt+ " < ?", val)
			return nil
		}).
		Process()
	if err != nil {
		log.Error(ctx, err)
		return nil, err
	}

	var srvModels []*TaskCallbackSrvModel
	err = srvQueryScope.Limit(int(queryStream.Limit)).Offset(int(queryStream.Offset)).Find(&srvModels).Error
	if err != nil {
		log.Error(ctx, err)
		return nil, err
	}

	if len(srvModels) == 0 {
		return nil, nil
	}

	srvModelIds := reflectutil.PluckUint64(srvModels, FieldId)

	var routeModels []*TaskCallbackSrvRouteModel
	err = conn.Where(DbFieldSrvId + " IN ?", srvModelIds).Find(&routeModels).Error
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
			logger.MustGetRepoLogger().Error(ctx, err)
			return nil, err
		}
	}
	return srvRepo, nil
}