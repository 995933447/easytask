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
	"github.com/go-playground/validator"
	"net/http"
	"reflect"
	"sync"
	"sync/atomic"
)

type handlerReflect struct {
	handler reflect.Value
	req reflect.Type
}

type Route struct {
	Path string `validate:"required"`
	Method string `validate:"required"`
	Handler any `validate:"required"`
}

func (r *Route) Check() error {
	return validator.New().Struct(r)
}

type Router struct {
	host string
	port int
	routeMap map[string]map[string]*handlerReflect
	routeMu sync.RWMutex
	isBooted atomic.Bool
}

func (r *Router) RegisterBatch(ctx context.Context, routes []*Route) error {
	for _, route := range routes {
		if err := r.Register(ctx, route); err != nil {
			logger.MustGetSysLogger().Error(ctx, err)
		}
	}
	return nil
}

func (r *Router) Register(ctx context.Context, route *Route) error {
	if r.isBooted.Load() {
		return internalerr.ErrServerStarted
	}

	log := logger.MustGetSysLogger()

	if err := route.Check(); err != nil {
		log.Error(ctx, err)
		return err
	}

	handlerType := reflect.TypeOf(route.Handler)
	errInvalidHandler := errors.New("handler must be implements func(api.*Context, *req)) (*resp, error)")

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
	if handlerType.Out(0).Kind() != reflect.Pointer {
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
	methodToHandlerMap, ok := r.routeMap[route.Path]
	if !ok {
		methodToHandlerMap = make(map[string]*handlerReflect)
	}
	methodToHandlerMap[route.Method] = &handlerReflect{
		handler: reflect.ValueOf(route.Handler),
		req: reqType,
	}
	r.routeMap[route.Path] = methodToHandlerMap

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
			logger.MustGetSessLogger().Error(ctx, err)
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
				logger.MustGetSessLogger().Error(ctx, err)
				respErr(ctx, writer, errs.ErrCodeInternal)
				return
			}

			resp := replies[0].Interface()
			respJson, err := json.Marshal(resp)
			if err != nil {
				logger.MustGetSessLogger().Error(ctx, err)
				respErr(ctx, writer, errs.ErrCodeInternal)
				return
			}

			err = r.writeResp(ctx, writer, 200, respJson, map[string]string{"Content-Type": "application/json"})
			if err != nil {
				logger.MustGetSessLogger().Error(ctx, err)
				return
			}
		})
	}

	srv := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", r.host, r.port),
		Handler:      srvMux,
	}
	if err := srv.ListenAndServe(); err != nil {
		logger.MustGetSysLogger().Error(ctx, err)
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
			logger.MustGetSessLogger().Error(ctx, err)
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