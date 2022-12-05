package apihandler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/995933447/easytask/internal/util/logger"
	"github.com/995933447/easytask/pkg/contx"
	"github.com/995933447/easytask/pkg/errs"
	internalerr "github.com/995933447/easytask/internal/util/errs"
	"github.com/ggicci/httpin"
	"net/http"
	"reflect"
	"sync"
	"sync/atomic"
)

type handlerReflect struct {
	handler reflect.Value
	req reflect.Type
	resp reflect.Type
}

type Router struct {
	host string
	port int
	routeMap map[string]map[string]*handlerReflect
	routeMu sync.RWMutex
	isBooted atomic.Bool
}

func (r *Router) Register(ctx context.Context, path, method string, handler any) error {
	if r.isBooted.Load() {
		return internalerr.ErrServerStarted
	}

	handlerType := reflect.TypeOf(handler)
	log := logger.MustGetSysProcLogger()
	errInvalidHandler := errors.New("handler must be implements func(api.*Context, req)) (resp, error)")

	if handlerType.NumIn() != 2 || handlerType.NumOut() != 2 {
		log.Error(ctx, errInvalidHandler)
		return errInvalidHandler
	}

	if _, ok := reflect.New(handlerType.In(0)).Interface().(context.Context); !ok {
		log.Error(ctx, errInvalidHandler)
		return errInvalidHandler
	}

	reqType := handlerType.In(1)
	if reqType.Kind() == reflect.Pointer {
		if reqType = reqType.Elem(); reqType.Kind() != reflect.Struct {
			log.Error(ctx, errInvalidHandler)
			return errInvalidHandler
		}
	}
	
	respType := handlerType.Out(0)
	if respType.Kind() != reflect.Pointer {
		if respType = respType.Elem(); respType.Kind() != reflect.Struct {
			log.Error(ctx, errInvalidHandler)
			return errInvalidHandler
		}
	}

	if _, ok := reflect.New(handlerType.Out(1)).Interface().(error); !ok {
		log.Error(ctx, errInvalidHandler)
		return errInvalidHandler
	}

	r.routeMu.Lock()
	defer r.routeMu.Unlock()
	methodToHandlerMap, ok := r.routeMap[path]
	if !ok {
		methodToHandlerMap = make(map[string]*handlerReflect)
	}
	methodToHandlerMap[method] = &handlerReflect{
		handler: reflect.ValueOf(handler),
		req: reqType,
		resp: respType,
	}
	r.routeMap[path] = methodToHandlerMap

	return nil
}

func (r *Router) Boot(ctx context.Context) error {
	if r.isBooted.Load() {
		return internalerr.ErrServerStarted
	}

	r.isBooted.Store(true)
	defer func() {
		if r.isBooted.Load() {
			r.isBooted.Store(false)
		}
	}()

	respErr := func(ctx context.Context, writer http.ResponseWriter, errCode errs.ErrCode) {
		if err := r.writeResp(ctx, writer, int(errCode), []byte(errs.GetErrMsg(errCode)), nil); err != nil {
			logger.MustGetSessProcLogger().Error(ctx, err)
		}
	}

	srvMux := http.NewServeMux()

	for path, methodToHandlerMap := range r.routeMap {
		srvMux.HandleFunc(path, func(writer http.ResponseWriter, req *http.Request) {
			ctx := contx.New("api", req.Context())

			handlerReflec, ok := methodToHandlerMap[req.Method]
			if !ok {
				respErr(ctx, writer, errs.ErrCodeRouteMethodNotAllow)
				return
			}

			argsVal := req.Context().Value(httpin.Input)

			if !reflect.TypeOf(argsVal).Elem().ConvertibleTo(handlerReflec.req) {
				respErr(ctx, writer, errs.ErrCodeArgsInvalid)
				return
			}

			handleReq := reflect.ValueOf(argsVal).Elem().Convert(handlerReflec.req)

			replies := handlerReflec.handler.Call([]reflect.Value{reflect.ValueOf(ctx), handleReq})
			err := replies[1].Interface().(error)
			if err != nil {
				logger.MustGetSessProcLogger().Error(ctx, err)
				respErr(ctx, writer, errs.ErrCodeInternal)
				return
			}

			resp := replies[0].Interface()
			respJson, err := json.Marshal(resp)
			if err != nil {
				logger.MustGetSessProcLogger().Error(ctx, err)
				respErr(ctx, writer, errs.ErrCodeInternal)
				return
			}

			err = r.writeResp(ctx, writer, 200, respJson, map[string]string{"Content-Type": "application/json"})
			if err != nil {
				logger.MustGetSessProcLogger().Error(ctx, err)
				return
			}
		})
	}

	srv := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", r.host, r.port),
		Handler:      srvMux,
	}
	if err := srv.ListenAndServe(); err != nil {
		logger.MustGetSysProcLogger().Error(ctx, err)
		return err
	}

	return nil
}

func (r *Router) writeResp(ctx context.Context, writer http.ResponseWriter, code int, content []byte, header map[string]string) error {
	writer.WriteHeader(code)

	for key, val := range header {
		writer.Header().Set(key, val)
	}

	var (
		contentLen = len(content)
		written int
	)
	for {
		n, err := writer.Write(content)
		if err != nil {
			logger.MustGetSessProcLogger().Error(ctx, err)
			return err
		}
		written += n
		if written >= contentLen {
			break
		}
		content = content[written:]
	}

	return nil
}

func NewRouter(host string, port int) *Router {
	return &Router{
		host: host,
		port: port,
		routeMap: make(map[string]map[string]*handlerReflect),
	}
}