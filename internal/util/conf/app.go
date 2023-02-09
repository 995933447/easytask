package conf

import (
	"github.com/995933447/easytask/internal/util/logger"
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

type AppConf struct {
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

