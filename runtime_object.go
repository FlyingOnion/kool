package kool

import (
	"fmt"
	"reflect"

	"k8s.io/apimachinery/pkg/runtime"
)

func toRuntimeObject(p interface{}) (runtime.Object, bool) {
	obj, ok := p.(runtime.Object)
	return obj, ok
}

func mustBeRuntimeObject(p interface{}) runtime.Object {
	obj, ok := toRuntimeObject(p)
	if !ok {
		panic(fmt.Sprintf("object of type (%s) is not a runtime.Object", reflect.TypeOf(p).String()))
	}
	return obj
}
