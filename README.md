## Introduction
easytask is a distributed task scheduling framework using golang coding. 
It's core design goal is to develop quickly and learn simple, lightweight, and easy to expand. 
Now, it's already open source, and our company use it in production environments, real "out-of-the-box".

easytask是使用go语言开发的一个分布式任务调度平台，其核心设计目标是开发迅速、学习简单、轻量级、易扩展。现已开放源代码并接入公司线上产品线，开箱即用。

## Features
- 1、简单：操作简单，一分钟上手，代码设计简单，容易二次开发；
- 2、动态：支持动态修改任务状态、启动/停止任务，以及终止运行中任务，即时生效；
- 3、调度中心HA（中心式）：调度采用中心式设计，“调度中心”自研调度组件并支持集群部署，可保证调度中心HA；
- 4、回调服务HA（分布式）：任务分布式执行，任务"会掉粉服务"支持集群部署，可保证任务执行HA；
- 5、注册中心: 执行器会周期性自动注册任务, 调度中心将会自动发现注册的任务并触发执行。同时，也支持手动录入执行器地址；
- 6、弹性扩容缩容：一旦有新执行器机器上线或者下线，下次调度时将会重新分配任务；
- 7、触发策略：提供丰富的任务触发策略，包括：Cron触发、固定间隔触发、固定延时触发;
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
// 项目依赖: mysql+redis或mysql+etcd(二选一即可),reids或etcd配置其一即可,HA主备选举
easytask -c easytask/conf/conf.json

// 调用示例在项目代码根目录test/api_server_test.go
````