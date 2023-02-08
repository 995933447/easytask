## Introduction
easytask is a high performing and distributed task scheduling framework using golang coding. 
It's core design goal is to develop quickly and learn simple, lightweight, and easy to expand. 
Now, it's already open source, and our company use it in production environments, real "out-of-the-box".

easytask是使用go语言开发的一个高性能的分布式任务调度平台，其核心设计目标是开发迅速、学习简单、轻量级、易扩展。现已开放源代码并接入公司线上产品线，开箱即用。

## Features
- 1、简单：操作简单，一分钟上手。golang开发，部署简单。代码设计简单，容易二次开发(做UI管理后台只涉及简单的数据表curd而已)；
- 2、动态：支持动态修改任务状态、启动/停止任务，以及终止运行中任务，即时生效；
- 3、调度中心HA（中心式）：调度采用中心式设计，“调度中心”自研调度组件并支持集群部署，可保证调度中心HA；
- 4、回调服务HA（分布式）：任务分布式执行，任务"回调服务"支持集群部署，可保证任务执行HA；
- 5、注册中心：对注册的回调服务进行周期性健康检查。 
- 6、弹性扩容缩容：一旦有回调机器上线或者下线，下次调度时将会加入回调触发节点池中，随机获取一个节点触发回调；
- 7、触发策略：提供丰富的任务触发策略，包括：根据Cron表达式定时触发、固定间隔触发、固定延时触发;
- 8、调度线程池：调度系统多线程/协程触发调度运行，确保调度精确执行，不被堵塞；
- 10、支持异步确认模式：由于调度任务采用协程池进行，为保证调度中心性能，回调任务支持异步确认模式，防止单个任务阻塞过久影响协程池执行其他任务；
- 11、http json api：采用http json方式进行api交互，各种语言可以轻松对接；

## Usage
服务运行
````
// 下载代码
git clone https://github.com/995933447/easytask.git

// 编译
go build -o easytask .
mv easytask /bin/bash/easytask

// 运行服务, 配置文件示例在项目代码根目录conf/conf.json
// 项目依赖: mysql+redis或mysql+etcd(二选一即可),reids或etcd配置其一即可,用于HA主备选举。任务相关数据存储在mysql中。
easytask -c easytask/conf/conf.json

// CTRL+C 会优雅退出.easytak捕获了SIGINT和SIGTERM信号,收到信号均会优雅退出.

// api调用示例在项目代码根目录test/api_server_test.go:(https://github.com/995933447/easytask/blob/master/test/api_server_test.go)
````

# HTTP API LIST
##### (golang可以使用封装好的客户端：https://github.com/995933447/easytask/blob/master/pkg/rpc/http_cli.go)
- 1、注册回调服务
````
URL:${api_server_host}:${api_server_port}/add_task_server

METHOD:POST

REQUEST PARAM:
name string 服务名称
schema string 回调协议(http或者https)
host string 服务host
port int 服务端口
callback_timeout_sec int 服务回调超时时间(将作为回调任务时候默认的http超时时间)
is_enable_health_check bool 是否开启健康检查

RESPONSE PARAM:
````
- 2、注销回调服务
````
URL:${api_server_host}:${api_server_port}/del_task_server

METHOD:POST

REQUEST PARAM:
name string 服务名称
schema string 回调协议(http或者https)
host string 服务host
port int 服务端口

RESPONSE PARAM:
````
- 3、注册任务
````
URL:${api_server_host}:${api_server_port}/add_task

METHOD:POST

REQUEST PARAM:
name string 任务名称
srv_name string 回调服务名称，注册任务之前请确保已经注册可用的任务服务
callback_path string 回调路径uri,选传。最终回调url为：${task_server_url}/callback_path
sched_mode int 调度模式，1.cron表达式模式。2.指定时间模式。3.间隔执行模式。
time_cron string cron表达式，sched_mode是1时候必传
time_interval_sec int 间隔执行模式，sched_mode是3时候必传
time_spec_at int 指定时间执行模式
arg string 任务执行参数
biz_id string 任务业务唯一id,如果存在则更新任务配置


RESPONSE PARAM:
task_id string 任务id
````
- 4、确认任务结果
````
URL:${api_server_host}:${api_server_port}/confirm_task

METHOD:POST

REQUEST PARAM:
task_id string 任务id
is_success bool 是否执行成功，将记录到mysql任务日志表（task_log）
extra string 自定义扩展信息，将记录到mysql任务日志表（task_log）
task_run_times int 确认的是第几次执行的任务


RESPONSE PARAM:
````
- 5、停止任务
````
URL:${api_server_host}:${api_server_port}/confirm_task

METHOD:POST

REQUEST PARAM:
task_id string 任务id

RESPONSE PARAM:
````

# TASK HTTP CALLBACK LIST
- 1、心跳检查回调
````
URL:${task_server_node_schema}//:${task_server_node_host}:{task_server_node_port}/

METHOD:POST

REQUEST PARAM:
cmd int 固定为1

RESPONSE PARAM:
pong bool true,代表服务可用（一般固定回复true即可）。如节点未返回正确响应或者服务异常未正确响应则任务服务节点不可以，会将该节点路由从注册中心剔除。
````
- 2、任务调度回调
````
URL:${task_server_node_schema}//:${task_server_node_host}:{task_server_node_port}/${task_callback_path}

METHOD:POST

REQUEST PARAM:
cmd int 固定为0
task_id string 任务id
task_name string 任务名称
arg string 任务参数
run_times int 第几次执行任务
biz_id string 任务业务唯一id

RESPONSE PARAM:
is_run_in_async bool 是否异步执行，异步模式需要把执行结果调用异步确认api进行任务结果确认，将记录到mysql任务日志表（task_log）
is_success bool 任务是否执行成功，将记录到mysql任务日志表（task_log）
extra string 任务执行响应自定义参数，将记录到mysql任务日志表（task_log）
`````

# example
#### api调用示例:
````
127.0.0.1:8801/add_task
request:
{
    "name":"try",
    "srv_name":"srv_test", 
    "callback_path":"/add/task/persist/async_callback",
    "sched_mode":1,
    "time_cron":"* * * * * *",
    "arg":"{\"foo\":\"bar\"}",
    "biz_id":"123"
}
response:
{
    "code": 0,
    "msg": "",
    "data": {
        "task_id": "26"
    },
    "hint": "2H7j9fWR1mFKf1Yfu8Cd"
}
````

# assistant tools
基于php开发的任务面板和管理后台:https://github.com/995933447/easytask-admin