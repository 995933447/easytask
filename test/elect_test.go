package test

import (
	"context"
	"github.com/995933447/distribmu"
	"github.com/995933447/distribmu/factory"
	"github.com/995933447/log-go"
	"github.com/995933447/log-go/impls/loggerwriters"
	"github.com/995933447/redisgroup"
	"github.com/995933447/std-go/print"
	"testing"
	"time"
)

func TestRedis(t *testing.T) {
	redis := redisgroup.NewGroup([]*redisgroup.Node{
		redisgroup.NewNode("127.0.0.1", 6379, ""),
	}, log.NewLogger(loggerwriters.NewStdoutLoggerWriter(print.ColorBlue)))
	err := redis.Set(context.Background(), "abcd", []byte("123"), time.Minute)
	if err != nil {
		t.Error(err)
	}
	return
}

func TestRedisGroupMuLock(t *testing.T) {
	redisGroup := redisgroup.NewGroup([]*redisgroup.Node{
		redisgroup.NewNode("127.0.0.1", 6379, ""),
	}, log.NewLogger(loggerwriters.NewStdoutLoggerWriter(print.ColorBlue)))

	muCfg := factory.NewMuConf(
		factory.MuTypeRedis,
		"abc",
		time.Second * 10,
		"123",
		factory.NewRedisMuDriverConf(redisGroup, 500),
	)
	mu := factory.MustNewMu(muCfg)
	success, err := mu.Lock(context.Background())
	if err != nil {
		t.Error(err)
	}
	t.Logf("bool:%v", success)
	time.Sleep(time.Second * 2)
	err = mu.RefreshTTL(context.Background())
	if err != nil {
		t.Error(err)
	}
	success, err = mu.LockWait(context.Background(), time.Second * 10)
	if err != nil {
		t.Error(err)
	}
	t.Logf("bool:%v", success)
	//err = mu.WaitKeyRelease(context.Background(), time.Second * 10)
	//if err != nil {
	//	t.Error(err)
	//}
	err = distribmu.DoWithMaxRetry(context.Background(), mu, 3, time.Second, func() error {
		t.Log("hello")
		return nil
	})
	if err != nil {
		t.Error(err)
	}
	err = distribmu.DoWithMustDone(context.Background(), mu, time.Second, func() error {
		t.Log("hello")
		return nil
	})
	if err != nil {
		t.Error(err)
	}
}

