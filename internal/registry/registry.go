package registry

import (
	"context"
	"github.com/995933447/autoelect"
	"github.com/995933447/easytask/internal/task"
	"github.com/995933447/easytask/internal/util/errs"
	"github.com/995933447/easytask/internal/util/logger"
	"github.com/995933447/easytask/pkg/contxt"
	bizerrs "github.com/995933447/easytask/pkg/errs"
	"github.com/995933447/optionstream"
	"github.com/995933447/simpletrace"
	simpletracectx "github.com/995933447/simpletrace/context"
	"sync"
	"sync/atomic"
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
	isPaused				  atomic.Bool
	exitWorkerWait			  sync.WaitGroup
	exitSchedSignCh			  chan struct{}
}

func NewRegistry(
	checkHealthWorkerPoolSize uint,
	srvRepo task.TaskCallbackSrvRepo,
	callbackSrvExec task.TaskCallbackSrvExec,
	elect autoelect.AutoElection,
	) *Registry {
	if checkHealthWorkerPoolSize == 0 {
		checkHealthWorkerPoolSize = DefaultCheckWorkerPoolSize
	}
	return &Registry{
		checkHealthIntervalSec: 5,
		checkHealthWorkerPoolSize: checkHealthWorkerPoolSize,
		srvRepo: srvRepo,
		callbackSrvExec: callbackSrvExec,
		readyCheckSrvChan: make(chan *task.TaskCallbackSrv),
		elect: elect,
		exitSchedSignCh: make(chan struct{}),
	}
}

func (r *Registry) Stop() {
	r.isPaused.Store(true)
	r.exitSchedSignCh <- struct{}{}
	r.exitWorkerWait.Wait()
}

func (r *Registry) Discover(ctx context.Context, srvName string) (*task.TaskCallbackSrv, error) {
	srvs, err := r.srvRepo.GetSrvs(
		ctx,
		optionstream.NewQueryStream(nil, 1, 0).SetOption(task.QueryOptKeyEqName, srvName),
	)
	if err != nil {
		logger.MustGetRegistryLogger().Error(ctx, err)
		return nil, err
	}

	if len(srvs) == 0 {
		logger.MustGetRegistryLogger().Warnf(ctx, "task callback server(name:%s) not found", srvName)
		return nil, bizerrs.NewBizErr(bizerrs.ErrCodeTaskCallbackSrvNotFound)
	}

	return srvs[0], nil
}

// 注册路由，如果服务名称已经存在，则增量添加路由
func (r *Registry) Register(ctx context.Context, srv *task.TaskCallbackSrv) error {
	if err := r.srvRepo.AddSrvRoutes(ctx, srv); err != nil {
		logger.MustGetRegistryLogger().Error(ctx, err)
		return err
	}
	return nil
}

// 删除路由
func (r *Registry) Unregister(ctx context.Context, srv *task.TaskCallbackSrv) error {
	if err := r.srvRepo.DelSrvRoutes(ctx, srv); err != nil {
		logger.MustGetRegistryLogger().Error(ctx, err)
		return err
	}
	return nil
}

func (r *Registry) HealthCheck(ctx context.Context) error {
	if !r.elect.IsMaster() {
		err := errs.ErrCurrentNodeNoMaster
		logger.MustGetRegistryLogger().Error(ctx, err)
		return err
	}

	var(
		size, offset int64 = 1000, 0
	)
	queryStream := optionstream.NewQueryStream(nil, size, offset).
		SetOption(task.QueryOptKeyEnabledHeathCheck, nil)
	for {
		srvs, err := r.srvRepo.GetSrvs(ctx, queryStream)
		if err != nil {
			logger.MustGetRegistryLogger().Error(ctx, err)
			return err
		}

		if len(srvs) == 0 {
			offset = 0
			logger.MustGetRegistryLogger().Debug(ctx, "no more servers need checking")
			break
		}

		offset += size
		queryStream.SetOffset(offset)

		logger.MustGetRegistryLogger().Debugf(ctx, "checking servers(len:%d)", len(srvs))
		for _, srv := range srvs {
			r.readyCheckSrvChan <- srv
		}
	}

	return nil
}

func (r *Registry) Run(ctx context.Context) {
	go r.createHealthCheckWorkerPool(contxt.ChildOf(ctx))
	r.sched(contxt.ChildOf(ctx))
}

func (r *Registry) sched(ctx context.Context) {
	var (
		traceModule = "registry_sched"
		origCtxTraceId string
	)
	if traceCtx, ok := ctx.(*simpletracectx.Context); ok {
		origCtxTraceId = traceCtx.GetTraceId()
	}

	for {
		var isExitingSched bool
		select {
		case _ = <- r.exitSchedSignCh:
			isExitingSched = true
		default:
			break
		}

		if isExitingSched {
			break
		}

		ctx = contxt.NewWithTrace(traceModule, context.TODO(), traceModule + "_" + origCtxTraceId + "." + simpletrace.NewTraceId(), "")

		logger.MustGetRegistryLogger().Debug(ctx, "checking health")

		if err := r.HealthCheck(ctx); err != nil {
			logger.MustGetRegistryLogger().Error(ctx, err)
		}

		logger.MustGetRegistryLogger().Debug(ctx, "checked health")

		time.Sleep(time.Duration(r.checkHealthIntervalSec) * time.Second)
	}
}

func (r *Registry) createHealthCheckWorkerPool(ctx context.Context) {
	logger.MustGetRegistryLogger().Info(ctx, "start create health check worker pool")

	var i uint
	for ; i < r.checkHealthWorkerPoolSize; i++ {
		go r.runWorker(contxt.ChildOf(ctx))
		r.exitWorkerWait.Add(1)
		logger.MustGetRegistryLogger().Infof(ctx, "worker(id:%d) is running", i)
	}

	logger.MustGetRegistryLogger().Info(ctx, "finish creating health check worker pool")
}

func (r *Registry) runWorker(ctx context.Context) {
	var (
		traceModule = "registry_worker"
		origCtxTraceId string
	)
	if traceCtx, ok := ctx.(*simpletracectx.Context); ok {
		origCtxTraceId = traceCtx.GetTraceId()
	}
	checkPausedTk := time.NewTicker(time.Second * 2)
	defer checkPausedTk.Stop()
	for {
		var (
			srv *task.TaskCallbackSrv
			isPaused bool
		)
		select {
			case srv = <- r.readyCheckSrvChan:
			case <- checkPausedTk.C:
				if isPaused = r.isPaused.Load(); isPaused {
					r.exitWorkerWait.Done()
				}
		}

		if isPaused {
			break
		}

		if srv == nil {
			continue
		}

		ctx = contxt.NewWithTrace(traceModule, context.TODO(), traceModule + "_" + origCtxTraceId + "." + simpletrace.NewTraceId(), "")

		logger.MustGetRegistryLogger().Debugf(ctx, "checking srv(name:%s)", srv.GetName())

		heatBeatResp, err := r.callbackSrvExec.HeartBeat(ctx, srv)
		if err != nil {
			logger.MustGetRegistryLogger().Error(ctx, err)
			continue
		}

		replyRoutes := heatBeatResp.GetReplyRoutes()
		if len(replyRoutes) > 0 {
			withReplyRouteSrv := task.NewTaskCallbackSrv(srv.GetId(), srv.GetName(), replyRoutes, true)
			if err := r.srvRepo.SetSrvRoutesPassHealthCheck(ctx, withReplyRouteSrv); err != nil {
				logger.MustGetRegistryLogger().Error(ctx, err)
			}
		}

		noReplyRoutes := heatBeatResp.GetNoReplyRoutes()
		if len(noReplyRoutes) > 0 {
			withNoReplyRouteSrv := task.NewTaskCallbackSrv(srv.GetId(), srv.GetName(), noReplyRoutes, true)
			if err := r.srvRepo.DelSrvRoutes(ctx, withNoReplyRouteSrv); err != nil {
				logger.MustGetRegistryLogger().Error(ctx, err)
			}
		}
	}
}