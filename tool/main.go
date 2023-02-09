package main

import (
	"context"
	"github.com/995933447/confloader"
	"github.com/995933447/easytask/internal/task"
	"github.com/995933447/easytask/internal/task/impl/repo/mysql"
	"github.com/995933447/easytask/internal/util/conf"
	"github.com/995933447/easytask/internal/util/logger"
	"github.com/995933447/easytask/pkg/contxt"
	"github.com/995933447/goconsole"
	"github.com/995933447/optionstream"
	"github.com/995933447/std-go/scan"
)

var cfg conf.AppConf

func init() {
	confFile := scan.OptStr("c")
	confLoader := confloader.NewLoader(confFile, 10, &cfg)
	if err := confLoader.Load(); err != nil {
		panic(err)
	}

	logger.Init(cfg.LoggerConf)
}

func NewRepos(ctx context.Context) (task.TaskRepo, task.TaskCallbackSrvRepo, task.TaskLogRepo, error) {
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

func ClearExpiredTasks() {
	expiredAt := scan.OptInt64("e")
	ctx := contxt.New("tool", context.TODO())
	taskRepo, _, _, err := NewRepos(ctx)
	if err != nil {
		logger.MustGetSysLogger().Error(ctx, err)
		return
	}
	err = taskRepo.DelTasks(ctx, optionstream.NewStream(nil).
		SetOption(task.QueryOptKeyCreatedExceed, expiredAt).
		SetOption(task.QueryOptKeyTaskFinished, nil))
	if err != nil {
		logger.MustGetSysLogger().Error(ctx, err)
		return
	}
}

func ClearExpiredTaskLogs() {
	expiredAt := scan.OptInt64("e")
	ctx := contxt.New("tool", context.TODO())
	_, _, taskLogRepo, err := NewRepos(ctx)
	if err != nil {
		logger.MustGetSysLogger().Error(ctx, err)
		return
	}
	err = taskLogRepo.DelLogs(ctx, optionstream.NewStream(nil).
		SetOption(task.QueryOptKeyCreatedExceed, expiredAt))
	if err != nil {
		logger.MustGetSysLogger().Error(ctx, err)
		return
	}
}

func main() {
	goconsole.Register("ClearExpiredTasks", "-e [expired at(过期时间)]", ClearExpiredTasks)
	goconsole.Register("ClearExpiredTaskLogs", "-e [expired at(过期时间)]", ClearExpiredTaskLogs)
	goconsole.Run()
}