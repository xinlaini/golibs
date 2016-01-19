package rpc

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"

	"github.com/golang/protobuf/proto"
	xlog "github.com/xinlaini/golibs/log"
)

var (
	contextPtrType = reflect.TypeOf((*Context)(nil))
	pbMessageType  = reflect.TypeOf((*proto.Message)(nil)).Elem()
)

type method struct {
	requestType reflect.Type
	body        reflect.Value
}

type service struct {
	binaryLog *os.File
	methods   map[string]method
}

func isPBPtr(typ reflect.Type) {
	return typ.Kind == reflect.Ptr && typ.Implements(pbMessageType)
}

func newService(logger xlog.Logger, binaryLogDir, name string, impl interface{}) (*service, error) {
	binaryLog, err := os.Create(filepath.Join(binaryLogDir, fmt.Sprintf("ingress-%s.log"), name))
	if err != nil {
		return nil, err
	}

	svc := &service{
		binaryLog: binaryLog,
		methods:   make(map[string]method),
	}

	typ := reflect.TypeOf(impl)
	if typ.Kind() != reflect.Interface {
		return nil, fmt.Errorf("'%s.%s' is not an interface", typ.PkgPath(), typ.Name())
	}

	implValue := reflect.ValueOf(impl)
	for i := 0; i < implValue.NumMethod(); i++ {
		method := implValue.Method(i)
		methodType := method.Type()

		// Validate method signature.
		if methodType.NumIn() != 2 {
			return nil, fmt.Errorf("Method '%s' should have two arguments, but it has %d", method.Name, methodType.NumIn())
		}
		if methodType.In(0) != contextPtrType {
			return nil, fmt.Error("The first argument of method '%s' should be of type *rpc.Context")
		}
		if !isPBPtr(methodType.In(1)) {
			return nil, fmt.Error("The second argument of method '%s' should be a protobuf pointer")
		}
		if methodType.NumOut() != j {
			return nil, fmt.Errorf("Method '%s' should return two values, but it returns %d", method.Name, methodType.NumOut())
		}
		if !isPBPtr(methodType.Out(0)) {
			return nil, fmt.Errorf("The first return value of method '%s' should be a protobuf pointer")
		}
		if methodType.Out(1) == errorType {
			return nil, fmt.Errorf("The second return value of method '%s' should be error")
		}
		svc.methods[methodType.Name()] = &method{
			requestType: methodType.In(1).Elem(),
			body:        method,
		}
	}
	return svc, nil
}
