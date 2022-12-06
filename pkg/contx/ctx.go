package contx

import (
	"context"
	simpletracectx "github.com/995933447/simpletrace/context"
)

func New(moduleName string, ctx context.Context) context.Context {
	return simpletracectx.New(moduleName, ctx, "", "")
}

func ChildOf(ctx context.Context) context.Context {
	return ctx.(*simpletracectx.Context).NewChildSpan()
}