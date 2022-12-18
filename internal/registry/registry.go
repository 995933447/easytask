package registry

import (
	"context"
	"github.com/995933447/autoelect"
	"github.com/995933447/easytask/internal/task"
	"github.com/995933447/easytask/internal/util/errs"
	"github.com/995933447/easytask/internal/util/logger"
	"github.com/995933447/easytask/pkg/contxt"
	"github.com/995933447/optionstream"
	"time"
)

const (
	DefaultCheckWorkerPoolSize = 100
)

type Registry struct {
	checkHealthIntervalSec    int64
	checkHealthWorkerPoolSize uint
	srvRepo                   task.TaskCallbackSrvRepo
	callbackSrvExec           task.TaskCallbackSrvExec
	readyCheckSrvChan         chan *task.TaskCallbackSrv
	elect                     autoelect.AutoElection
	isClusterMode             bool
}

func NewRegistry(
	isClusterMode bool,
	checkHealthWorkerPoolSize uint,
	srvRepo task.TaskCallbackSrvRepo,
	callbackSrvExec task.TaskCallbackSrvExec,
	elect autoelect.AutoElection,
	) *Registry {
	if checkHealthWorkerPoolSize == 0 {
		checkHealthWorkerPoolSize = DefaultCheckWorkerPoolSize
	}
	return &Registry{
		checkHealthWorkerPoolSize: checkHealthWorkerPoolSize,
		srvRepo: srvRepo,
		callbackSrvExec: callbackSrvExec,
		readyCheckSrvChan: make(chan *task.TaskCallbackSrv),
		elect: elect,
		isClusterMode: isClusterMode,
	}
}

func (r *Registry) Discover(ctx context.Context, srvName string) (*task.TaskCallbackSrv, error) {
	log := logger.MustGetRegistryLogger()

	srvs, err := r.srvRepo.GetSrvs(
		contxt.ChildOf(ctx),
		optionstream.NewQueryStream(nil, 1, 0).SetOption(task.QueryOptKeyEqName, srvName),
	)
	if err != nil {
		log.Error(ctx, err)
		return nil, err
	}

	if len(srvs) == 0 {
		log.Warnf(ctx, "task callback server(name:%s) not found", srvName)
		return nil, errs.ErrTaskCallbackServerNotFound
	}

	return srvs[0], nil
}

// 注册路由，如果服务名称已经存在，则增量添加路由
func (r *Registry) Register(ctx context.Context, srv *task.TaskCallbackSrv) error {
	if err := r.srvRepo.AddSrvRoutes(contxt.ChildOf(ctx), srv); err != nil {
		logger.MustGetRegistryLogger().Error(ctx, err)
		return err
	}
	return nil
}

// 删除路由
func (r *Registry) Unregister(ctx context.Context, srv *task.TaskCallbackSrv) error {
	if err := r.srvRepo.DelSrvRoutes(contxt.ChildOf(ctx), srv); err != nil {
		logger.MustGetRegistryLogger().Error(ctx, err)
		return err
	}
	return nil
}

func (r *Registry) HeathCheck(ctx context.Context) error {
	if r.isClusterMode && !r.elect.IsMaster() {
		err := errs.ErrCurrentNodeNoMaster
		logger.MustGetRegistryLogger().Error(ctx, err)
		return err
	}

	queryStream := optionstream.NewQueryStream(nil, 1000, 0).
		SetOption(task.QueryOptKeyCheckedHealthLt, time.Now().Unix() - r.checkHealthIntervalSec).
		SetOption(task.QueryOptKeyEnabledHeathCheck, nil)
	for {
		srvs, err := r.srvRepo.GetSrvs(contxt.ChildOf(ctx), queryStream)
		if err != nil {
			logger.MustGetRegistryLogger().Error(ctx, err)
			return err
		}

		if len(srvs) == 0 {
			break
		}

		for _, srv := range srvs {
			r.readyCheckSrvChan <- srv
		}
	}

	return nil
}

func (r *Registry) Run(ctx context.Context) {
	go r.createHealthCheckWorkerPool(contxt.ChildOf(ctx))
	go r.sched(contxt.ChildOf(ctx))
}

func (r *Registry) sched(ctx context.Context) {
	for {
		if err := r.HeathCheck(contxt.ChildOf(ctx)); err != nil {
			logger.MustGetRegistryLogger().Error(ctx, err)
		}
		time.Sleep(time.Duration(r.checkHealthIntervalSec) * time.Second)
	}
}

func (r *Registry) createHealthCheckWorkerPool(ctx context.Context) {
	var (
		i uint
		withNoReplyRouteSrvCh = make(chan *task.TaskCallbackSrv)
		withReplyRouteSrvCh = make(chan *task.TaskCallbackSrv)
	)
	for ; i < r.checkHealthWorkerPoolSize; i++ {
		go r.runWorker(contxt.ChildOf(ctx), withReplyRouteSrvCh, withNoReplyRouteSrvCh)
	}

	for {
		var (
			withNoReplyRouteSrvs []*task.TaskCallbackSrv
			withReplyRouteSrvs []*task.TaskCallbackSrv
		)
		select {
		case withNoReplyRouteSrv := <- withNoReplyRouteSrvCh:
			withNoReplyRouteSrvs = append(withNoReplyRouteSrvs, withNoReplyRouteSrv)
			var noCheckedMoreSrv bool
			for {
				select {
				case withNoReplyRouteSrv = <- withNoReplyRouteSrvCh:
					withNoReplyRouteSrvs = append(withNoReplyRouteSrvs, withNoReplyRouteSrv)
				default:
					noCheckedMoreSrv = true
				}
				if noCheckedMoreSrv {
					break
				}
			}
			for _, srv := range withNoReplyRouteSrvs {
				if err := r.srvRepo.DelSrvRoutes(contxt.ChildOf(ctx), srv); err != nil {
					logger.MustGetRegistryLogger().Error(ctx, err)
				}
			}
		case withReplyRouteSrv := <- withReplyRouteSrvCh:
			withReplyRouteSrvs = append(withReplyRouteSrvs, withReplyRouteSrv)
			var noCheckedMoreSrv bool
			for {
				select {
				case withReplyRouteSrv = <- withNoReplyRouteSrvCh:
					withNoReplyRouteSrvs = append(withReplyRouteSrvs, withReplyRouteSrv)
				default:
					noCheckedMoreSrv = true
				}
				if noCheckedMoreSrv {
					break
				}
			}

			for _, srv := range withReplyRouteSrvs {
				if err := r.srvRepo.SetSrvRoutesPassHealthCheck(contxt.ChildOf(ctx), srv); err != nil {
					logger.MustGetRegistryLogger().Error(ctx, err)
				}
			}
		}
	}
}

func (r *Registry) runWorker(ctx context.Context, withNoReplyRouteSrvCh, withReplyRouteSrvCh chan *task.TaskCallbackSrv) {
	srv := <- r.readyCheckSrvChan
	heatBeatResp, err := r.callbackSrvExec.HeartBeat(contxt.ChildOf(ctx), srv)
	if err != nil {
		logger.MustGetRegistryLogger().Error(ctx, err)
		return
	}
	withNoReplyRouteSrvCh <- task.NewTaskCallbackSrv(srv.GetId(), srv.GetName(), heatBeatResp.GetNoReplyRoutes(), true)
	withReplyRouteSrvCh <- task.NewTaskCallbackSrv(srv.GetId(), srv.GetName(), heatBeatResp.GetReplyRoutes(), true)
}