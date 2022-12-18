package contxt

import (
	"context"
	simpletracectx "github.com/995933447/simpletrace/context"
)

func New(moduleName string, ctx context.Context) context.Context {
	return simpletracectx.New(moduleName, ctx, "", "")
}

func NewWithTrace(moduleName string, ctx context.Context, traceId, spanId string) context.Context {
	return simpletracectx.New(moduleName, ctx, traceId, spanId)
}

func ChildOf(ctx context.Context) context.Context {
	return ctx.(*simpletracectx.Context).NewChildSpan()
}