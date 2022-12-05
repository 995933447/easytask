package apihandler

import (
	"reflect"
	"testing"
)

type Struc2 struct {
	Name string
}

type Struc struct {
	Name string
}

func TestParseArg(t *testing.T) {
	stru := Struc{
		Name: "hello",
	}
	handleReq := reflect.TypeOf(Struc2{})
	val := reflect.TypeOf(stru)
	if val.ConvertibleTo(handleReq) {
		t.Logf("%+v", reflect.ValueOf(stru).Convert(handleReq).Interface())
	} else {
		t.Log("cannot convert")
	}
}
