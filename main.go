package main

import (
	"context"
	"github.com/995933447/autoelect"
	electfactory "github.com/995933447/autoelect/factory"
	"github.com/995933447/confloader"
	distribmufactory "github.com/995933447/distribmu/factory"
	"github.com/995933447/easytask/internal/apiserver"
	"github.com/995933447/easytask/internal/registry"
	"github.com/995933447/easytask/internal/task"
	"github.com/995933447/easytask/internal/task/impl/callback"
	"github.com/995933447/easytask/internal/task/impl/repo/mysql"
	"github.com/995933447/easytask/internal/util/conf"
	"github.com/995933447/easytask/internal/util/logger"
	"github.com/995933447/easytask/internal/util/runtime"
	"github.com/995933447/easytask/pkg/contxt"
	"github.com/995933447/log-go"
	"github.com/995933447/redisgroup"
	"github.com/995933447/std-go/scan"
	"github.com/etcd-io/etcd/client"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	ctx := contxt.New("main", context.TODO())

	cfg, err := loadConf()
	if err != nil {
		panic(any(err))
	}

	logger.Init(cfg.LoggerConf)

	logger.MustGetSysLogger().Info(ctx, "easytask starting...")

	defer runtime.RecoverToTraceAndExit(ctx)

	elect, doElectErrCh, err := startElect(ctx, cfg)
	if err != nil {
		panic(any(err))
	}

	go func() {
		select {
		case err := <- doElectErrCh:
			logger.MustGetSysLogger().Errorf(ctx, "electing occur error:%s", err)
			panic(any(err))
		}
	}()

	taskRepo, taskCallbackSrvRepo, taskLogRepo, err := NewRepos(ctx, cfg)
	if err != nil {
		panic(any(err))
	}

	reg := runRegistry(ctx, cfg, taskLogRepo, taskCallbackSrvRepo, elect)

	workerEngine := runTaskWorker(ctx, cfg, taskLogRepo, taskRepo, elect)

	sysSignCh := make(chan os.Signal)
	stopApiSrvSignCh := make(chan struct{})
	stoppedApiSrvSignCh := make(chan struct{})
	// 优雅退出
	go func() {
		for {
			_ = <- sysSignCh
			elect.StopElect()
			logger.MustGetSysLogger().Info(ctx, "stopped elect")
			reg.Stop()
			logger.MustGetSysLogger().Info(ctx, "stopped registry")
			workerEngine.Stop()
			logger.MustGetSysLogger().Info(ctx, "stopped worker engine")
			stopApiSrvSignCh <- struct{}{}
			<- stoppedApiSrvSignCh
			logger.MustGetSysLogger().Info(ctx, "stopped api server")
			logger.Flush()
			os.Exit(0)
		}
	}()
	signal.Notify(sysSignCh, syscall.SIGINT, syscall.SIGTERM)

	if err = runHttpApiServer(ctx, cfg, taskRepo, reg, stopApiSrvSignCh, stoppedApiSrvSignCh); err != nil {
		panic(any(err))
	}
}

func loadConf() (*conf.AppConf, error) {
	confFile := scan.OptStr("c")
	var cfg conf.AppConf
	confLoader := confloader.NewLoader(confFile, 10, &cfg)
	if err := confLoader.Load(); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func startElect(ctx context.Context, cfg *conf.AppConf) (autoelect.AutoElection, chan error, error) {
	var (
		elect autoelect.AutoElection
		err error
	)
	switch cfg.ElectDriver {
	case "etcd":
		etcdCli, err := client.New(client.Config{
			Endpoints: cfg.Endpoints,
		})
		if err != nil {
			logger.MustGetSysLogger().Error(ctx, err)
			return nil, nil, err
		}

		elect, err = electfactory.NewAutoElection(
			electfactory.ElectDriverGitDistribMu,
			electfactory.NewDistribMuElectDriverConf(
				cfg.ClusterName,
				time.Second * 5,
				distribmufactory.MuTypeEtcd,
				distribmufactory.NewEtcdMuDriverConf(etcdCli),
			),
		)
		if err != nil {
			logger.MustGetSysLogger().Error(ctx, err)
			return nil, nil, err
		}
	case "redis":
		var nodes []*redisgroup.Node
		for _, nodeConf := range cfg.RedisConf.Nodes {
			nodes = append(nodes, redisgroup.NewNode(nodeConf.Host, nodeConf.Port, nodeConf.Password))
		}
		redis := redisgroup.NewGroup(nodes, logger.MustGetElectLogger().(*log.Logger))
		elect, err = electfactory.NewAutoElection(
			electfactory.ElectDriverGitDistribMu,
			electfactory.NewDistribMuElectDriverConf(
				cfg.ClusterName,
				time.Second * 5,
				distribmufactory.MuTypeRedis,
				distribmufactory.NewRedisMuDriverConf(redis, 1000),
			),
		)
		if err != nil {
			logger.MustGetSysLogger().Error(ctx, err)
			return nil, nil, err
		}
	}

	doElectErrCh := make(chan error)
	go func() {
		if err = elect.LoopInElect(contxt.ChildOf(ctx), doElectErrCh); err != nil {
			logger.MustGetSysLogger().Error(ctx, err)
			doElectErrCh <- err
		}
	}()

	return elect, doElectErrCh, nil
}

func runRegistry(ctx context.Context, cfg *conf.AppConf, taskLogRepo task.TaskLogRepo, taskCallbackSrvRepo task.TaskCallbackSrvRepo, elect autoelect.AutoElection) *registry.Registry {
	reg := registry.NewRegistry(
		cfg.HealthCheckWorkerPoolSize,
		taskCallbackSrvRepo,
		callback.NewHttpExec(task.NewTaskLogger(taskLogRepo, logger.MustGetCallbackLogger())),
		elect,
		)
	go reg.Run(contxt.ChildOf(ctx))
	return reg
}

func NewRepos(ctx context.Context, cfg *conf.AppConf) (task.TaskRepo, task.TaskCallbackSrvRepo, task.TaskLogRepo, error) {
	var (
		taskRepo            task.TaskRepo
		taskCallbackSrvRepo task.TaskCallbackSrvRepo
		taskLogRepo         task.TaskLogRepo
		err                 error
	)
	if taskLogRepo, err = mysql.NewTaskLogRepo(ctx, cfg.MysqlConf.ConnDsn); err != nil {
		logger.MustGetSysLogger().Error(ctx, err)
		return nil, nil, nil, err
	}
	if taskCallbackSrvRepo, err = mysql.NewTaskSrvRepo(ctx, cfg.MysqlConf.ConnDsn); err != nil {
		logger.MustGetSysLogger().Error(ctx, err)
		return nil, nil, nil, err
	}
	if taskRepo, err = mysql.NewTaskRepo(ctx, cfg.MysqlConf.ConnDsn, taskCallbackSrvRepo, taskLogRepo); err != nil {
		logger.MustGetSysLogger().Error(ctx, err)
		return nil, nil, nil, err
	}
	return taskRepo, taskCallbackSrvRepo, taskLogRepo, nil
}

func runTaskWorker(ctx context.Context, cfg *conf.AppConf, taskLogRepo task.TaskLogRepo, taskRepo task.TaskRepo, elect autoelect.AutoElection) *task.WorkerEngine {
	engine := task.NewWorkerEngine(
		cfg.TaskWorkerPoolSize,
		task.NewSched(taskRepo, elect),
		callback.NewHttpExec(task.NewTaskLogger(taskLogRepo, logger.MustGetCallbackLogger())),
		)
	go engine.Run(contxt.ChildOf(ctx))
	return engine
}

func runHttpApiServer(ctx context.Context, cfg *conf.AppConf, taskRepo task.TaskRepo, reg *registry.Registry, stopSignCh, stoppedSignCh chan struct{}) error {
	router := apiserver.NewHttpRouter(cfg.ApiSrvConf.Host, cfg.ApiSrvConf.Port, cfg.PprofPort)
	if err := router.RegisterBatch(ctx, getHttpApiRoutes(taskRepo, reg)); err != nil {
		logger.MustGetSysLogger().Error(ctx, err)
		return err
	}
	go func() {
		<- stopSignCh
		router.Stop()
		stoppedSignCh <- struct{}{}
	}()
	if err := router.Boot(ctx); err != nil {
		logger.MustGetSysLogger().Error(ctx, err)
		return err
	}
	return nil
}
