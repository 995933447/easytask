package easytask

import (
	"context"
	"github.com/995933447/autoelect"
	electfactory "github.com/995933447/autoelect/factory"
	"github.com/995933447/confloader"
	distribmufactory "github.com/995933447/distribmu/factory"
	"github.com/995933447/easytask/internal/apihandler"
	callbacksrvexecimpl "github.com/995933447/easytask/internal/callbacksrvexec/impl"
	"github.com/995933447/easytask/internal/registry"
	"github.com/995933447/easytask/internal/repo"
	mysqlrepo "github.com/995933447/easytask/internal/repo/impl/mysql"
	"github.com/995933447/easytask/internal/sched"
	"github.com/995933447/easytask/internal/task"
	"github.com/995933447/easytask/internal/util/logger"
	"github.com/995933447/easytask/pkg/contx"
	"github.com/995933447/log-go"
	"github.com/995933447/log-go/impls/loggerwriters"
	"github.com/995933447/redisgroup"
	"github.com/995933447/std-go/print"
	"github.com/995933447/std-go/scan"
	"github.com/etcd-io/etcd/client"
	"net/http"
	"os"
	"os/signal"
	"runtime/debug"
	"strings"
	"syscall"
	"time"
)

type MysqlConf struct {
	ConnDsn string `json:"conn_dsn"`
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

type ApiServerConf struct {
	Host string `json:"host"`
	Port int `json:"port"`
}

type Conf struct {
	IsClusterMode bool 	`json:"is_cluster_mode"`
	ClusterName string `json:"cluster_name"`
	TaskWorkerPoolSize uint `json:"task_worker_pool_size"`
	ElectDriver string `json:"elect_driver"`
	*MysqlConf `json:"mysql"`
	*EtcdConf `json:"etcd"`
	*RedisConf `json:"redis"`
	HealthCheckWorkerPoolSize uint `json:"health_check_worker_pool_size"`
	LoggerConf *logger.Conf `json:"log"`
	*ApiServerConf `json:"api_server"`
}

func main() {
	conf, err := loadConf()
	if err != nil {
		panic(any(err))
	}

	logger.Init(conf.LoggerConf)

	ctx := contx.New("main", context.TODO())

	defer recoverPanic(ctx)

	taskRepo, taskCallbackSrvRepo, err := NewRepos(ctx, conf)
	if err != nil {
		panic(any(err))
	}

	elect, err := startElect(ctx, conf)
	if err != nil {
		panic(any(err))
	}

	runRegistry(ctx, conf, taskCallbackSrvRepo, elect)

	runTaskWorker(ctx, conf, taskRepo, elect)

	signCh := make(chan os.Signal)
	go func() {
		for {
			_ = <-signCh
			elect.StopElect()
			os.Exit(0)
		}
	}()
	signal.Notify(signCh, syscall.SIGINT, syscall.SIGTERM)

	if err = runApiServer(ctx, conf); err != nil {
		panic(any(err))
	}
}

func recoverPanic(ctx context.Context) {
	if err := recover(); err != nil {
		logger.MustGetSysProcLogger().Errorf(ctx, "PROCESS PANIC: err %s", err)
		stack := debug.Stack()
		if len(stack) > 0 {
			lines := strings.Split(string(stack), "\n")
			for _, line := range lines {
				logger.MustGetSysProcLogger().Error(ctx, line)
			}
		}
		logger.MustGetSysProcLogger().Errorf(ctx, "stack is empty (%s)", err)
	}
}

func loadConf() (*Conf, error) {
	confFile := scan.OptStr("c")
	var conf Conf
	confLoader := confloader.NewLoader(confFile, 3, &conf)
	if err := confLoader.Load(); err != nil {
		return nil, err
	}
	errCh := make(chan error)
	go func() {
		for {
			err := <- errCh
			panic(any(err))
		}
	}()
	go confLoader.WatchToLoad(errCh)
	return &conf, nil
}

func startElect(ctx context.Context, conf *Conf) (autoelect.AutoElection, error) {
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
			return nil, err
		}

		elect, err = electfactory.NewAuthElection(
			electfactory.ElectDriverGitDistribMu,
			electfactory.NewDistribMuElectDriverConf(
				conf.ClusterName,
				time.Second * 5,
				distribmufactory.MuTypeEtcd,
				distribmufactory.NewEtcdMuDriverConf(etcdCli),
			),
		)
		if err != nil {
			return nil, err
		}
	case "redis":
		var nodes []*redisgroup.Node
		for _, nodeConf := range conf.RedisConf.Nodes {
			nodes = append(nodes, redisgroup.NewNode(nodeConf.Host, nodeConf.Port, nodeConf.Password))
		}
		logger := log.NewLogger(loggerwriters.NewStdoutLoggerWriter(print.ColorRed))
		logger.SetLogLevel(log.LevelPanic)
		redisGroup := redisgroup.NewGroup(nodes, logger)

		elect, err = electfactory.NewAuthElection(
			electfactory.ElectDriverGitDistribMu,
			electfactory.NewDistribMuElectDriverConf(
				conf.ClusterName,
				time.Second * 5,
				distribmufactory.MuTypeRedis,
				distribmufactory.NewRedisMuDriverConf(redisGroup, 500),
			),
		)
		if err != nil {
			return nil, err
		}
	}

	errDuringLoopCh := make(chan error)
	go func() {
		errDuringLoop := <- errDuringLoopCh
		logger.MustGetSysProcLogger().Error(ctx, errDuringLoop)
	}()

	go func() {
		if err = elect.LoopInElect(ctx, errDuringLoopCh); err != nil {
			return
		}
	}()
	if err != nil {
		return nil, err
	}

	return elect, nil
}

func runRegistry(ctx context.Context, conf *Conf, taskCallbackSrvRepo repo.TaskCallbackSrvRepo, elect autoelect.AutoElection) {
	reg := registry.NewRegistry(
		conf.IsClusterMode,
		conf.HealthCheckWorkerPoolSize,
		taskCallbackSrvRepo,
		callbacksrvexecimpl.NewHttpExec(),
		elect,
		)
	go reg.Run(ctx)
}

func NewRepos(ctx context.Context, conf *Conf) (repo.TaskRepo, repo.TaskCallbackSrvRepo, error) {
	var (
		taskRepo repo.TaskRepo
		taskCallbackSrvRepo repo.TaskCallbackSrvRepo
		err error
	)
	if taskCallbackSrvRepo, err = mysqlrepo.NewTaskSrvRepo(ctx, conf.MysqlConf.ConnDsn); err != nil {
		return nil, nil, err
	}
	if taskRepo, err = mysqlrepo.NewTaskRepo(ctx, conf.MysqlConf.ConnDsn, taskCallbackSrvRepo); err != nil {
		return nil, nil, err
	}
	return taskRepo, taskCallbackSrvRepo, nil
}

func runTaskWorker(ctx context.Context, conf *Conf, taskRepo repo.TaskRepo, elect autoelect.AutoElection) {
	engine :=  task.NewWorkerEngine(
		conf.TaskWorkerPoolSize,
		sched.NewSched(conf.IsClusterMode, taskRepo, elect),
		callbacksrvexecimpl.NewHttpExec(),
		)
	go engine.Run(ctx)
}

func runApiServer(ctx context.Context, conf *Conf) error {
	router := apihandler.NewRouter(conf.ApiServerConf.Host, conf.ApiServerConf.Port)
	if err := router.RegisterBatch(ctx, apiRoutes); err != nil {
		logger.MustGetSysProcLogger().Error(ctx, err)
		return err
	}
	if err := router.Boot(ctx); err != nil {
		logger.MustGetSysProcLogger().Error(ctx, err)
		return err
	}
	return nil
}