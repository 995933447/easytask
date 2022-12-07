package apiserver

import (
	"reflect"
)

type handlerReflect struct {
	handler reflect.Value
	req reflect.Type
}
