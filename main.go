package easytask

import (
	"context"
	"fmt"
	"github.com/995933447/autoelect"
	electfactory "github.com/995933447/autoelect/factory"
	"github.com/995933447/confloader"
	"github.com/995933447/distribmu"
	distribmufactory "github.com/995933447/distribmu/factory"
	callbacksrvexecimpl "github.com/995933447/easytask/internal/callbacksrvexec/impl"
	"github.com/995933447/easytask/internal/registry"
	"github.com/995933447/easytask/internal/repo"
	"github.com/995933447/easytask/internal/sched"
	"github.com/995933447/easytask/internal/task"
	mysqlrepo "github.com/995933447/easytask/internal/repo/impl/mysql"
	"github.com/995933447/std-go/print"
	"github.com/995933447/log-go"
	"github.com/995933447/log-go/impls/loggerwriters"
	"github.com/995933447/redisgroup"
	"github.com/995933447/std-go/scan"
	"errors"
	"github.com/etcd-io/etcd/client"
	"github.com/fengleng/mars/config"
	p "golang.org/x/tools/go/analysis/passes/sigchanyzer/testdata/src/a"
	"os"
	"os/signal"
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

type Conf struct {
	ClusterName string `json:"cluster_name"`
	TaskWorkerPoolSize uint `json:"task_worker_pool_size"`
	ElectDriver string `json:"elect_driver"`
	*MysqlConf `json:"mysql_conf"`
	*EtcdConf `json:"etcd_conf"`
	*RedisConf `json:"redis_conf"`
	HealthCheckWorkerPoolSize uint `json:"health_check_worker_pool_size"`
}

func main() {
	conf, _ := loadConf()

	ctx := context.TODO()

	taskRepo, taskCallbackSrvRepo, err := NewRepos(conf)
	if err != nil {
		panic(any(err))
	}

	elect, err := startElect(ctx, conf)
	if err != nil {
		panic(any(err))
	}

	runRegistry(ctx, conf, taskCallbackSrvRepo, elect)

	runTaskWorker(ctx, conf, taskRepo, elect)

	sysSignCh := make(chan os.Signal)
	go func() {
		for {
			_ = <-sysSignCh
			elect.StopElect()
			os.Exit(0)
		}
	}()
	signal.Notify(sysSignCh, syscall.SIGINT, syscall.SIGTERM)
}

func loadConf() (*Conf, *confloader.Loader) {
	confFile := scan.OptStr("c")
	var conf Conf
	confLoader := confloader.NewLoader(confFile, 3, &conf)
	errCh := make(chan error)
	go func() {
		for {
			_ = <- errCh
		}
	}()
	go confLoader.WatchToLoad(errCh)
	return &conf, confLoader
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
		fmt.Println(errDuringLoop)
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
		conf.HealthCheckWorkerPoolSize,
		taskCallbackSrvRepo,
		callbacksrvexecimpl.NewHttpExec(),
		elect,
		)
	go reg.Run(ctx)
}

func NewRepos(conf *Conf) (repo.TaskRepo, repo.TaskCallbackSrvRepo, error) {
	var (
		taskRepo repo.TaskRepo
		taskCallbackSrvRepo repo.TaskCallbackSrvRepo
		err error
	)
	if taskCallbackSrvRepo, err = mysqlrepo.NewTaskSrvRepo(conf.MysqlConf.ConnDsn); err != nil {
		return nil, nil, err
	}
	if taskRepo, err = mysqlrepo.NewTaskRepo(conf.MysqlConf.ConnDsn, taskCallbackSrvRepo); err != nil {
		return nil, nil, err
	}
	return taskRepo, taskCallbackSrvRepo, nil
}

func runTaskWorker(ctx context.Context, conf *Conf, taskRepo repo.TaskRepo, elect autoelect.AutoElection) {
	engine :=  task.NewWorkerEngine(
		conf.TaskWorkerPoolSize,
		sched.NewSched(taskRepo, elect),
		callbacksrvexecimpl.NewHttpExec(),
		)
	go engine.Run(ctx)
}