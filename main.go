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
	mysqlrepo "github.com/995933447/easytask/internal/task/impl/repo/mysql"
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

type MysqlConf struct {
	ConnDsn string `json:"dsn"`
}

type EtcdConf struct {
	Endpoints []string `json:"endpoints"`
}

type RedisNodeConf struct {
	Host string `json:"host"`
	Port int `json:"port"`
	Password string `json:"password"`
}

type RedisConf struct {
	Nodes []*RedisNodeConf `json:"nodes"`
}

type HttpApiSrvConf struct {
	Host string `json:"host"`
	Port int `json:"port"`
	PprofPort int `json:"pprof_port"`
}

type ApiSrvConf struct {
	*HttpApiSrvConf `json:"http"`
}

type Conf struct {
	ClusterName               string `json:"cluster_name"`
	TaskWorkerPoolSize        uint `json:"task_worker_pool_size"`
	ElectDriver               string `json:"elect_driver"`
	*MysqlConf                `json:"mysql"`
	*EtcdConf                 `json:"etcd"`
	*RedisConf                `json:"redis"`
	HealthCheckWorkerPoolSize uint `json:"health_check_worker_pool_size"`
	LoggerConf                *logger.Conf `json:"log"`
	*ApiSrvConf               `json:"api_server"`
}

func main() {
	ctx := contxt.New("main", context.TODO())

	conf, errDuringWatchConf, err := loadConf()
	if err != nil {
		panic(any(err))
	}

	logger.Init(conf.LoggerConf)

	logger.MustGetSysLogger().Info(ctx, "easytask starting...")

	defer runtime.RecoverToTraceAndExit(ctx)

	elect, doElectErrCh, err := startElect(ctx, conf)
	if err != nil {
		panic(any(err))
	}

	go func() {
		select {
		case err := <- errDuringWatchConf:
			logger.MustGetSysLogger().Errorf(ctx, "watching config error:%s", err)
			panic(any(err))
		case err := <- doElectErrCh:
			logger.MustGetSysLogger().Errorf(ctx, "electing occur error:%s", err)
			panic(any(err))
		}
	}()

	taskRepo, taskCallbackSrvRepo, taskLogRepo, err := NewRepos(ctx, conf)
	if err != nil {
		panic(any(err))
	}

	reg := runRegistry(ctx, conf, taskLogRepo, taskCallbackSrvRepo, elect)

	workerEngine := runTaskWorker(ctx, conf, taskLogRepo, taskRepo, elect)

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

	if err = runHttpApiServer(ctx, conf, taskRepo, reg, stopApiSrvSignCh, stoppedApiSrvSignCh); err != nil {
		panic(any(err))
	}
}

func loadConf() (*Conf, chan error, error) {
	confFile := scan.OptStr("c")
	var conf Conf
	confLoader := confloader.NewLoader(confFile, 3, &conf)
	if err := confLoader.Load(); err != nil {
		return nil, nil, err
	}
	errCh := make(chan error)
	go confLoader.WatchToLoad(errCh)
	return &conf, errCh, nil
}

func startElect(ctx context.Context, conf *Conf) (autoelect.AutoElection, chan error, error) {
	var (
		elect autoelect.AutoElection
		err error
	)
	switch conf.ElectDriver {
	case "etcd":
		etcdCli, err := client.New(client.Config{
			Endpoints: conf.Endpoints,
		})
		if err != nil {
			logger.MustGetSysLogger().Error(ctx, err)
			return nil, nil, err
		}

		elect, err = electfactory.NewAutoElection(
			electfactory.ElectDriverGitDistribMu,
			electfactory.NewDistribMuElectDriverConf(
				conf.ClusterName,
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
		for _, nodeConf := range conf.RedisConf.Nodes {
			nodes = append(nodes, redisgroup.NewNode(nodeConf.Host, nodeConf.Port, nodeConf.Password))
		}
		redis := redisgroup.NewGroup(nodes, logger.MustGetElectLogger().(*log.Logger))
		elect, err = electfactory.NewAutoElection(
			electfactory.ElectDriverGitDistribMu,
			electfactory.NewDistribMuElectDriverConf(
				conf.ClusterName,
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

func runRegistry(ctx context.Context, conf *Conf, taskLogRepo task.TaskLogRepo, taskCallbackSrvRepo task.TaskCallbackSrvRepo, elect autoelect.AutoElection) *registry.Registry {
	reg := registry.NewRegistry(
		conf.HealthCheckWorkerPoolSize,
		taskCallbackSrvRepo,
		callback.NewHttpExec(task.NewTaskLogger(taskLogRepo, logger.MustGetCallbackLogger())),
		elect,
		)
	go reg.Run(contxt.ChildOf(ctx))
	return reg
}

func NewRepos(ctx context.Context, conf *Conf) (task.TaskRepo, task.TaskCallbackSrvRepo, task.TaskLogRepo, error) {
	var (
		taskRepo            task.TaskRepo
		taskCallbackSrvRepo task.TaskCallbackSrvRepo
		taskLogRepo         task.TaskLogRepo
		err                 error
	)
	if taskLogRepo, err = mysqlrepo.NewTaskLogRepo(ctx, conf.MysqlConf.ConnDsn); err != nil {
		logger.MustGetSysLogger().Error(ctx, err)
		return nil, nil, nil, err
	}
	if taskCallbackSrvRepo, err = mysqlrepo.NewTaskSrvRepo(ctx, conf.MysqlConf.ConnDsn); err != nil {
		logger.MustGetSysLogger().Error(ctx, err)
		return nil, nil, nil, err
	}
	if taskRepo, err = mysqlrepo.NewTaskRepo(ctx, conf.MysqlConf.ConnDsn, taskCallbackSrvRepo, taskLogRepo); err != nil {
		logger.MustGetSysLogger().Error(ctx, err)
		return nil, nil, nil, err
	}
	return taskRepo, taskCallbackSrvRepo, taskLogRepo, nil
}

func runTaskWorker(ctx context.Context, conf *Conf, taskLogRepo task.TaskLogRepo, taskRepo task.TaskRepo, elect autoelect.AutoElection) *task.WorkerEngine {
	engine := task.NewWorkerEngine(
		conf.TaskWorkerPoolSize,
		task.NewSched(taskRepo, elect),
		callback.NewHttpExec(task.NewTaskLogger(taskLogRepo, logger.MustGetCallbackLogger())),
		)
	go engine.Run(contxt.ChildOf(ctx))
	return engine
}

func runHttpApiServer(ctx context.Context, conf *Conf, taskRepo task.TaskRepo, reg *registry.Registry, stopSignCh, stoppedSignCh chan struct{}) error {
	router := apiserver.NewHttpRouter(conf.HttpApiSrvConf.Host, conf.HttpApiSrvConf.Port, conf.PprofPort)
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
