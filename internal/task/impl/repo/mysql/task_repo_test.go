package mysql

import (
	"context"
	"github.com/995933447/easytask/internal/util/logger"
	"github.com/995933447/optionstream"
	"testing"
)

func TestTimeoutTask(t *testing.T) {
	logger.Init(&logger.Conf{
		SysProcLogDir: "/var/log/easytask/test",
		SessionProcLogDir: "/var/log/easytask/test",
		FileSize: 1024 * 1024 * 100,
	})
	srvRepo, err := NewTaskSrvRepo(context.TODO(), "root:@tcp(127.0.0.1:3306)/easytask?charset=utf8mb4&parseTime=True&loc=Local")
	if err != nil {
		t.Error(err)
		return
	}
	srvs, err := srvRepo.GetSrvs(context.Background(), optionstream.NewQueryStream(nil, 10, 0))
	if err != nil {
		t.Error(err)
		return
	}
	t.Log(srvs)
}