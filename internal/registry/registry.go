package registry

import (
	"context"
	"github.com/995933447/autoelect"
	"github.com/995933447/easytask/internal/callbacksrvexec"
	"github.com/995933447/easytask/internal/repo"
	"github.com/995933447/easytask/internal/task"
	"github.com/995933447/easytask/internal/util/errs"
	"github.com/995933447/optionstream"
	"time"
)

const (
	DefaultCheckWorkerPoolSize = 100
)

type Registry struct {
	checkHealthIntervalSec int64
	checkHealthWorkerPoolSize uint
	srvRepo repo.TaskCallbackSrvRepo
	callbackSrvExec callbacksrvexec.TaskCallbackSrvExec
	readyCheckSrvChan chan *task.TaskCallbackSrv
	elect autoelect.AutoElection
}

func NewRegistry(
	checkHealthWorkerPoolSize uint,
	srvRepo repo.TaskCallbackSrvRepo,
	callbackSrvExec callbacksrvexec.TaskCallbackSrvExec,
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
	}
}

// 注册路由，如果服务名称已经存在，则增量添加路由
func (r *Registry) Register(ctx context.Context, srv *task.TaskCallbackSrv) error {
	return r.srvRepo.AddSrvRoutes(ctx, srv)
}

// 删除路由
func (r *Registry) Unregister(ctx context.Context, srv *task.TaskCallbackSrv) error {
	return r.srvRepo.DelSrvRoutes(ctx, srv)
}

func (r *Registry) HeathCheck(ctx context.Context) error {
	if !r.elect.IsMaster() {
		return errs.ErrCurrentNodeNoMaster
	}

	queryStream := optionstream.NewQueryStream(nil, 1000, 0).
		SetOption(repo.QueryOptKeyCheckedHealthLt, time.Now().Unix() - r.checkHealthIntervalSec).
		SetOption(repo.QueryOptKeyEnabledHeathCheck, nil)
	for {
		srvs, err := r.srvRepo.GetSrvs(ctx, queryStream)
		if err != nil {
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
	go r.createHealthCheckWorkerPool(ctx)
	go r.sched(ctx)
}

func (r *Registry) sched(ctx context.Context) {
	for {
		if err := r.HeathCheck(ctx); err != nil {

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
		go r.runWorker(ctx, withReplyRouteSrvCh, withNoReplyRouteSrvCh)
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
				if err := r.srvRepo.DelSrvRoutes(ctx, srv); err != nil {

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
				if err := r.srvRepo.SetSrvRoutesPassHealthCheck(ctx, srv); err != nil {

				}
			}
		}
	}
}

func (r *Registry) runWorker(ctx context.Context, withNoReplyRouteSrvCh, withReplyRouteSrvCh chan *task.TaskCallbackSrv) {
	srv := <- r.readyCheckSrvChan
	heatBeatResp, err := r.callbackSrvExec.HeartBeat(ctx, srv)
	if err != nil {
		return
	}
	withNoReplyRouteSrvCh <- task.NewTaskCallbackSrv(srv.GetName(), heatBeatResp.GetNoReplyRoutes(), true)
	withReplyRouteSrvCh <- task.NewTaskCallbackSrv(srv.GetName(), heatBeatResp.GetReplyRoutes(), true)
}