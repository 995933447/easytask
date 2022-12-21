package main

import (
	"context"
	"fmt"
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
	"github.com/995933447/easytask/pkg/contxt"
	"github.com/995933447/log-go"
	"github.com/995933447/redisgroup"
	"github.com/995933447/std-go/scan"
	"github.com/etcd-io/etcd/client"
	"os"
	"os/signal"
	"runtime/debug"
	"strings"
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
}

type ApiSrvConf struct {
	*HttpApiSrvConf `json:"http"`
}

type Conf struct {
	IsClusterMode             bool `json:"is_cluster_mode"`
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

	defer recoverPanic(ctx)

	elect, errDuringElect, err := startElect(contxt.ChildOf(ctx), conf)
	if err != nil {
		panic(any(err))
	}

	go func() {
		select {
		case err := <- errDuringWatchConf:
			logger.MustGetSysLogger().Errorf(ctx, "watching config error:%s", err)
			panic(any(err))
		case err := <- errDuringElect:
			logger.MustGetSysLogger().Errorf(ctx, "electing occur error:%s", err)
			panic(any(err))
		}
	}()

	taskRepo, taskCallbackSrvRepo, err := NewRepos(contxt.ChildOf(ctx), conf)
	if err != nil {
		panic(any(err))
	}

	reg := runRegistry(contxt.ChildOf(ctx), conf, taskCallbackSrvRepo, elect)

	runTaskWorker(contxt.ChildOf(ctx), conf, taskRepo, elect)

	signCh := make(chan os.Signal)
	go func() {
		for {
			_ = <-signCh
			elect.StopElect()
			os.Exit(0)
		}
	}()
	signal.Notify(signCh, syscall.SIGINT, syscall.SIGTERM)

	if err = runHttpApiServer(contxt.ChildOf(ctx), conf, taskRepo, reg); err != nil {
		panic(any(err))
	}
}

func recoverPanic(ctx context.Context) {
	if err := recover(); err != nil {
		fmt.Println(err)
		logger.MustGetSysLogger().Errorf(ctx, "PROCESS PANIC: err %s", err)
		stack := debug.Stack()
		if len(stack) > 0 {
			lines := strings.Split(string(stack), "\n")
			for _, line := range lines {
				fmt.Println(line)
				logger.MustGetSysLogger().Error(ctx, line)
			}
		}
		logger.MustGetSysLogger().Errorf(ctx, "stack is empty (%s)", err)
		os.Exit(0)
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
				distribmufactory.NewRedisMuDriverConf(redis, 500),
			),
		)
		if err != nil {
			logger.MustGetSysLogger().Error(ctx, err)
			return nil, nil, err
		}
	}

	errDuringLoopCh := make(chan error)
	go func() {
		if err = elect.LoopInElect(contxt.ChildOf(ctx), errDuringLoopCh); err != nil {
			logger.MustGetSysLogger().Error(ctx, err)
			errDuringLoopCh <- err
		}
	}()

	return elect, errDuringLoopCh, nil
}

func runRegistry(ctx context.Context, conf *Conf, taskCallbackSrvRepo task.TaskCallbackSrvRepo, elect autoelect.AutoElection) *registry.Registry {
	reg := registry.NewRegistry(
		conf.IsClusterMode,
		conf.HealthCheckWorkerPoolSize,
		taskCallbackSrvRepo,
		callback.NewHttpExec(),
		elect,
		)
	go reg.Run(contxt.ChildOf(ctx))
	return reg
}

func NewRepos(ctx context.Context, conf *Conf) (task.TaskRepo, task.TaskCallbackSrvRepo, error) {
	var (
		taskRepo            task.TaskRepo
		taskCallbackSrvRepo task.TaskCallbackSrvRepo
		err                 error
	)
	if taskCallbackSrvRepo, err = mysqlrepo.NewTaskSrvRepo(contxt.ChildOf(ctx), conf.MysqlConf.ConnDsn); err != nil {
		logger.MustGetSysLogger().Error(ctx, err)
		return nil, nil, err
	}
	if taskRepo, err = mysqlrepo.NewTaskRepo(contxt.ChildOf(ctx), conf.MysqlConf.ConnDsn, taskCallbackSrvRepo); err != nil {
		logger.MustGetSysLogger().Error(ctx, err)
		return nil, nil, err
	}
	return taskRepo, taskCallbackSrvRepo, nil
}

func runTaskWorker(ctx context.Context, conf *Conf, taskRepo task.TaskRepo, elect autoelect.AutoElection) {
	engine := task.NewWorkerEngine(
		conf.TaskWorkerPoolSize,
		task.NewSched(conf.IsClusterMode, taskRepo, elect),
		callback.NewHttpExec(),
		)
	go engine.Run(contxt.ChildOf(ctx))
}

func runHttpApiServer(ctx context.Context, conf *Conf, taskRepo task.TaskRepo, reg *registry.Registry) error {
	router := apiserver.NewHttpRouter(conf.HttpApiSrvConf.Host, conf.HttpApiSrvConf.Port)
	if err := router.RegisterBatch(contxt.ChildOf(ctx), getHttpApiRoutes(taskRepo, reg)); err != nil {
		logger.MustGetSysLogger().Error(ctx, err)
		return err
	}
	if err := router.Boot(contxt.ChildOf(ctx)); err != nil {
		logger.MustGetSysLogger().Error(ctx, err)
		return err
	}
	return nil
}