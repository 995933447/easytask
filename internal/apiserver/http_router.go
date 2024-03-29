package apiserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	internalerr "github.com/995933447/easytask/internal/util/errs"
	"github.com/995933447/easytask/internal/util/logger"
	"github.com/995933447/easytask/internal/util/runtime"
	"github.com/995933447/easytask/pkg/contxt"
	"github.com/995933447/easytask/pkg/errs"
	"github.com/995933447/easytask/pkg/rpc/proto/httpproto"
	simpletracectx "github.com/995933447/simpletrace/context"
	"github.com/ahmek/kit/args"
	"github.com/go-playground/validator"
	"net/http"
	_ "net/http/pprof"
	"reflect"
	"sync"
	"sync/atomic"
)

type HttpRoute struct {
	Path string `validate:"required"`
	Method string `validate:"required"`
	Handler any `validate:"required"`
}

func (r *HttpRoute) Check() error {
	return validator.New().Struct(r)
}

type HttpRouter struct {
	host string
	port int
	pprofPort int
	routeMap map[string]map[string]*handlerReflect
	routeMu sync.RWMutex
	isBooted atomic.Bool
	isPaused atomic.Bool
	servingWait sync.WaitGroup
}

func (r *HttpRouter) RegisterBatch(ctx context.Context, routes []*HttpRoute) error {
	for _, route := range routes {
		if err := r.Register(ctx, route); err != nil {
			logger.MustGetSysLogger().Error(ctx, err)
			return err
		}
	}
	return nil
}

func (r *HttpRouter) Register(ctx context.Context, route *HttpRoute) error {
	if r.isBooted.Load() {
		return internalerr.ErrServerStarted
	}

	if err := route.Check(); err != nil {
		logger.MustGetSysLogger().Error(ctx, err)
		return err
	}

	handlerType := reflect.TypeOf(route.Handler)
	errInvalidHandler := errors.New("handler must be implements func(context.Context, *req)) (*resp, error)")

	if handlerType.NumIn() != 2 || handlerType.NumOut() != 2 {
		logger.MustGetSysLogger().Error(ctx, errInvalidHandler)
		return errInvalidHandler
	}

	if _, ok := reflect.New(handlerType.In(0)).Interface().(*context.Context); !ok {
		logger.MustGetSysLogger().Error(ctx, errInvalidHandler)
		return errInvalidHandler
	}

	reqType := handlerType.In(1)
	if reqType.Kind() == reflect.Pointer {
		if reqType = reqType.Elem(); reqType.Kind() != reflect.Struct {
			logger.MustGetSysLogger().Error(ctx, errInvalidHandler)
			return errInvalidHandler
		}
	}

	respType := handlerType.Out(0)
	if handlerType.Out(0).Kind() != reflect.Pointer {
		if respType = respType.Elem(); respType.Kind() != reflect.Struct {
			logger.MustGetSysLogger().Error(ctx, errInvalidHandler)
			return errInvalidHandler
		}
	}

	if _, ok := reflect.New(handlerType.Out(1)).Interface().(*error); !ok {
		logger.MustGetSysLogger().Error(ctx, errInvalidHandler)
		return errInvalidHandler
	}

	r.routeMu.Lock()
	defer r.routeMu.Unlock()

	if r.isBooted.Load() {
		return internalerr.ErrServerStarted
	}

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

func (r *HttpRouter) Stop() {
	r.isPaused.Store(true)
	r.servingWait.Wait()
}

func (r *HttpRouter) Boot(ctx context.Context) error {
	if r.isBooted.Load() {
		return internalerr.ErrServerStarted
	}
	r.routeMu.Lock()
	if r.isBooted.Load() {
		return internalerr.ErrServerStarted
	}
	r.isBooted.Store(true)
	r.routeMu.Unlock()
	defer func() {
		if r.isBooted.Load() {
			r.isBooted.Store(false)
		}
	}()

	go func() {
		if err := http.ListenAndServe(fmt.Sprintf("%s:%d", r.host, r.pprofPort), nil); err != nil {
			logger.MustGetSessLogger().Error(ctx, err)
		}
	}()

	respErr := func(ctx context.Context, writer http.ResponseWriter, errCode errs.ErrCode, errMsg, traceId string, header map[string]string) {
		if errMsg == "" {
			errMsg = errs.GetErrMsg(errCode)
		}
		content := httpproto.FinalStdoutResp{
			Code: errCode,
			Msg: errMsg,
			Hint: traceId,
		}
		contentJson, err := json.Marshal(content)
		if err != nil {
			logger.MustGetSessLogger().Error(ctx, err)
			return
		}
		if err := r.writeResp(ctx, writer, 200, contentJson, header); err != nil {
			logger.MustGetSessLogger().Error(ctx, err)
		}
	}

	srvMux := http.NewServeMux()

	for path, methodToHandlerMap := range r.routeMap {
		func(path string, methodToHandlerMap map[string]*handlerReflect) {
			srvMux.HandleFunc(path, func(writer http.ResponseWriter, req *http.Request) {
				r.servingWait.Add(1)
				defer r.servingWait.Done()

				var (
					ctx context.Context
					origTraceId = req.Header.Get(httpproto.HeaderSimpleTraceId)
					traceModule = "api"
				)
				if req.Header.Get(httpproto.HeaderSimpleTraceId) != "" {
					ctx = contxt.NewWithTrace(
						traceModule,
						contxt.NewWithTrace(traceModule, req.Context(), origTraceId, req.Header.Get(httpproto.HeaderSimpleTraceParentSpanId)),
						origTraceId,
						req.Header.Get(httpproto.HeaderSimpleTraceSpanId),
						)
				} else {
					ctx = contxt.New(traceModule, req.Context())
				}

				var (
					traceId string
					respHeader = map[string]string{
						"Content-Type": "application/json",
					}
				)
				if traceCtx, ok := ctx.(*simpletracectx.Context); ok {
					traceId = traceCtx.GetTraceId()
					respHeader[httpproto.HeaderSimpleTraceId] = traceId
					respHeader[httpproto.HeaderSimpleTraceSpanId] = traceCtx.GetSpanId()
					respHeader[httpproto.HeaderSimpleTraceParentSpanId] = traceCtx.GetParentSpanId()
				}

				defer runtime.RecoverToTraceAndExit(ctx)

				logger.MustGetSessLogger().Infof(
					ctx,
					"receive http request. method:%s, path:%s",
					req.Method, req.RequestURI,
					)

				if r.isPaused.Load() {
					respErr(ctx, writer, errs.ErrCodeServerStopped, "", traceId, respHeader)
					return
				}

				handlerReflec, ok := methodToHandlerMap[req.Method]
				if !ok {
					respErr(ctx, writer, errs.ErrCodeRouteMethodNotAllow, "", traceId, respHeader)
					return
				}

				handleReq := reflect.New(handlerReflec.req)
				httpCtx := args.NewHTTPContext(writer, req.WithContext(req.Context()))
				switch req.Method {
				case http.MethodPost:
					if err := httpCtx.PostArg(handleReq.Interface()); err != nil {
						logger.MustGetSessLogger().Error(ctx, err)
						respErr(ctx, writer, errs.ErrCodeArgsInvalid, "", traceId, respHeader)
						return
					}
				case http.MethodGet:
					if err := httpCtx.GetArg(handleReq.Interface()); err != nil {
						logger.MustGetSessLogger().Error(ctx, err)
						respErr(ctx, writer, errs.ErrCodeArgsInvalid, "", traceId, respHeader)
						return
					}
				}

				logger.MustGetSessLogger().Infof(ctx, "request param:%+v", handleReq.Interface())

				if err := validator.New().Struct(handleReq.Interface()); err != nil {
					logger.MustGetSessLogger().Error(ctx, err)
					respErr(ctx, writer, errs.ErrCodeArgsInvalid, err.Error(), traceId, respHeader)
					return
				}

				replies := handlerReflec.handler.Call([]reflect.Value{reflect.ValueOf(ctx), handleReq})
				replyErr := replies[1].Interface()
				if replyErr != nil {
					logger.MustGetSessLogger().Error(ctx, replyErr)
					if bizErr, ok := replyErr.(*errs.BizError); ok {
						respErr(ctx, writer, bizErr.Code(), bizErr.Msg(), traceId, respHeader)
						return
					}
					respErr(ctx, writer, errs.ErrCodeInternal, "", traceId, respHeader)
					return
				}

				resp := &httpproto.FinalStdoutResp{
					Data: replies[0].Interface(),
					Hint: traceId,
				}
				respJson, err := json.Marshal(resp)
				if err != nil {
					logger.MustGetSessLogger().Error(ctx, err)
					if handleErr, ok := err.(*HandleErr); ok {
						respErr(ctx, writer, handleErr.errCode, "", traceId, respHeader)
					} else {
						respErr(ctx, writer, errs.ErrCodeInternal, "", traceId, respHeader)
					}
					return
				}

				err = r.writeResp(ctx, writer, 200, respJson, respHeader)
				if err != nil {
					logger.MustGetSessLogger().Error(ctx, err)
					return
				}
			})
		} (path, methodToHandlerMap)
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

func (r *HttpRouter) writeResp(ctx context.Context, writer http.ResponseWriter, code int, content []byte, header map[string]string) error {
	defer func() {
		var headerLine string
		for key, val := range header {
			headerLine += key + ":" + val + "; "
		}
		logger.MustGetSessLogger().Infof(ctx, "resp body:%s, resp header:[%s]", string(content), headerLine)
	}()

	for key, val := range header {
		writer.Header().Set(key, val)
	}

	writer.WriteHeader(code)

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

func NewHttpRouter(host string, port, pprofPort int) *HttpRouter {
	return &HttpRouter{
		host: host,
		port: port,
		pprofPort: pprofPort,
		routeMap: make(map[string]map[string]*handlerReflect),
	}
}