package test

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/995933447/easytask/pkg/contxt"
	"github.com/995933447/easytask/pkg/rpc"
	"github.com/995933447/easytask/pkg/rpc/proto"
	"github.com/995933447/easytask/pkg/rpc/proto/httpproto"
	"io"
	"net/http"
	"testing"
	"time"
)

func TestApiServer(t *testing.T) {
	srvMux := http.NewServeMux()

	taskCli := rpc.NewHttpCli("http://127.0.0.1:8801")
	_, err := taskCli.RegisterTaskCallbackSrv(contxt.New("test", context.TODO()), &httpproto.RegisterTaskCallbackSrvReq{
		Name: "srv_test",
		Schema: "http",
		Host: "127.0.0.1",
		Port: 8082,
		IsEnableHealthCheck: true,
		CallbackTimeoutSec: 5,
	})
	if err != nil {
		t.Fatal(err)
		return
	}

	srvMux.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		resp := httpproto.HeartBeatResp{
			Pong: true,
		}
		j, err := json.Marshal(resp)
		if err != nil {
			t.Error(err)
			panic(err)
		}
		_, err = writer.Write(j)
		if err != nil {
			t.Error(err)
			panic(err)
		}
	})

	srvMux.HandleFunc("/add/task/persist/callback", func(writer http.ResponseWriter, request *http.Request) {
		body, err := io.ReadAll(request.Body)
		if err != nil {
			t.Fatal(err)
			return
		}
		t.Logf(string(body))

		req := &httpproto.TaskCallbackReq{}
		err = json.Unmarshal(body, req)
		if err != nil {
			t.Error(err)
			panic(err)
		}

		t.Logf("req:%+v", req)
		t.Log("arg:" + req.Arg)

		resp := httpproto.TaskCallbackResp{
			IsSuccess: true,
			Extra: `{"hint":"123abc"}`,
		}

		j, err := json.Marshal(resp)
		if err != nil {
			t.Error(err)
			panic(err)
		}
		_, err = writer.Write(j)
		if err != nil {
			t.Error(err)
			panic(err)
		}
	})

	srvMux.HandleFunc("/add/task/persist/async_callback", func(writer http.ResponseWriter, request *http.Request) {
		body, err := io.ReadAll(request.Body)
		if err != nil {
			t.Fatal(err)
			return
		}
		t.Logf(string(body))

		req := &httpproto.TaskCallbackReq{}
		err = json.Unmarshal(body, req)
		if err != nil {
			t.Error(err)
			panic(err)
		}

		t.Logf("req:%+v", req)

		resp := httpproto.TaskCallbackResp{
			IsRunInAsync: true,
		}

		j, err := json.Marshal(resp)
		if err != nil {
			t.Error(err)
			panic(err)
		}
		_, err = writer.Write(j)
		if err != nil {
			t.Error(err)
			panic(err)
		}

		time.Sleep(time.Minute)
		_, err = taskCli.ConfirmTask(context.Background(), &httpproto.ConfirmTaskReq{
			TaskId: req.TaskId,
			IsSuccess: true,
			TaskRunTimes: req.RunTimes,
			Extra: `{"hint":"abcdefg"}`,
		})
		if err != nil {
			t.Error(err)
			panic(err)
		}
	})

	srvMux.HandleFunc("/add/task/persist", func(writer http.ResponseWriter, request *http.Request) {
		addTaskResp, err := taskCli.AddTask(contxt.New("add_task", context.Background()), &httpproto.AddTaskReq{
			Name: "test_task_persist",
			SrvName: "test_srv",
			CallbackPath: "/add/task/persist/callback",
			SchedMode: proto.SchedModeTimeCron,
			TimeCron:  "*/3 * * * *",
			Arg: `{"hello":"world"}`,
		})
		if err != nil {
			t.Fatal(err)
			return
		}

		t.Logf("task id：%s", addTaskResp.Id)
	})

	srvMux.HandleFunc("/add/task/once", func(writer http.ResponseWriter, request *http.Request) {
		addTaskResp, err := taskCli.AddTask(contxt.New("add_task", context.Background()), &httpproto.AddTaskReq{
			Name: "test_task_once",
			SrvName: "test_srv",
			CallbackPath: "/add/task/persist/async_callback",
			SchedMode: proto.SchedModeTimeSpec,
			TimeSpecAt: time.Now().Add(time.Minute).Unix(),
			Arg: `{"hello":"world"}`,
		})
		if err != nil {
			t.Fatal(err)
			return
		}

		t.Logf("task id：%s", addTaskResp.Id)
	})

	srvMux.HandleFunc("/add/task/int", func(writer http.ResponseWriter, request *http.Request) {
		addTaskResp, err := taskCli.AddTask(contxt.New("add_task", context.Background()), &httpproto.AddTaskReq{
			Name: "test_task_interval",
			SrvName: "test_srv",
			CallbackPath: "/add/task/persist/callback",
			SchedMode: proto.SchedModeTimeInterval,
			TimeIntervalSec: 60 * 5,
			Arg: `{"hello":"world"}`,
		})
		if err != nil {
			t.Fatal(err)
			return
		}

		t.Logf("task id：%s", addTaskResp.Id)
	})

	srv := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", "0.0.0.0", 8082),
		Handler:      srvMux,
	}
	if err := srv.ListenAndServe(); err != nil {
		t.Fatal(err)
	}
}
