package rpc

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"time"

	"golang.org/x/net/context"

	"github.com/golang/protobuf/proto"
	xlog "github.com/xinlaini/golibs/log"
	"github.com/xinlaini/golibs/rpc/proto/gen-go"
)

const (
	recentCount = 64
)

var (
	serverCtxPtrType = reflect.TypeOf((*ServerContext)(nil))
	pbMessageType    = reflect.TypeOf((*proto.Message)(nil)).Elem()
	errorType        = reflect.TypeOf((*error)(nil)).Elem()
)

type method struct {
	requestType reflect.Type
	body        reflect.Value
}

type service struct {
	logger         xlog.Logger
	binaryLog      *os.File
	methods        map[string]*method
	chLog          chan [3][]byte
	recentCalls    [][3][]byte
	mtxRecentCalls sync.RWMutex
}

func (svc *service) logLoop() {
	next := 0
	for {
		data := <-svc.chLog
		svc.mtxRecentCalls.Lock()
		svc.recentCalls[next] = data
		svc.mtxRecentCalls.Unlock()
		next = (next + 1) % recentCount
		if svc.binaryLog != nil {
			for i := 0; i < 3; i++ {
				if _, err := svc.binaryLog.Write(data[i]); err != nil {
					svc.logger.Errorf("Failed to write to %s, it's now closed and may be compromised: %s", svc.binaryLog.Name(), err)
					svc.binaryLog.Close()
					svc.binaryLog = nil
					break
				}
			}
		}
	}
}

func (svc *service) serveRequest(request *rpc_proto.Request, response *rpc_proto.Response) {
	reqMeta := request.Metadata
	var err error
	if reqMeta.MethodName == nil {
		response.Error = makeErr("Request.Metadata is missing method_name")
		return
	}
	m, found := svc.methods[reqMeta.GetMethodName()]
	if !found {
		response.Error = makeErrf("Method '%s.%s' is not found", reqMeta.GetServiceName(), reqMeta.GetMethodName())
		return
	}
	var requestPB reflect.Value
	if request.RequestPb != nil {
		requestPB := reflect.New(m.requestType)
		if err = proto.Unmarshal(request.RequestPb, requestPB.Interface().(proto.Message)); err != nil {
			response.Error = makeErrf("Failed to unmarshal request for '%s.%s': %s", reqMeta.GetServiceName(), reqMeta.GetMethodName(), err)
			return
		}
	}

	parentCtx := context.Background()
	if reqMeta.GetTimeoutUs() > 0 {
		parentCtx, _ = context.WithTimeout(parentCtx, time.Duration(reqMeta.GetTimeoutUs())*time.Microsecond)
	}
	ctx := &ServerContext{
		Context:  parentCtx,
		Metadata: reqMeta,
	}

	ch := make(chan []reflect.Value)
	go func() {
		ch <- m.body.Call([]reflect.Value{reflect.ValueOf(ctx), requestPB})
	}()

	select {
	case callResults := <-ch:
		if callResults[1].IsNil() {
			if !callResults[0].IsNil() {
				if response.ResponsePb, err = proto.Marshal(callResults[0].Interface().(proto.Message)); err != nil {
					response.Error = makeErrf("Failed to marshal response for '%s.%s': %s", reqMeta.GetServiceName(), reqMeta.GetMethodName(), err)
					return
				}
			}
		} else {
			// This is an app-level error.
			response.Error = proto.String(callResults[1].Interface().(error).Error())
		}
	case <-ctx.Done():
		response.Error = makeErrf("Method '%s.%s' timed out", reqMeta.GetServiceName(), reqMeta.GetMethodName())
		return
	}
}

func (svc *service) log(requestBytes, responseSize, responseBytes []byte) {
	svc.chLog <- [3][]byte{requestBytes, responseSize, responseBytes}
}

func isPBPtr(typ reflect.Type) bool {
	return typ.Kind() == reflect.Ptr && typ.Implements(pbMessageType)
}

func newService(ctrl *Controller, name string, impl interface{}) (*service, error) {
	binaryLog, err := os.Create(filepath.Join(ctrl.binaryLogDir, fmt.Sprintf("%s-ingress.log"), name))
	if err != nil {
		return nil, err
	}

	svc := &service{
		logger:    ctrl.logger,
		binaryLog: binaryLog,
		methods:   make(map[string]*method),
		chLog:     make(chan [3][]byte),
	}

	typ := reflect.TypeOf(impl)
	if typ.Kind() != reflect.Interface {
		return nil, fmt.Errorf("'%s.%s' is not an interface", typ.PkgPath(), typ.Name())
	}

	implValue := reflect.ValueOf(impl)
	for i := 0; i < implValue.NumMethod(); i++ {
		m := implValue.Method(i)
		methodType := m.Type()

		// Validate method signature.
		if methodType.NumIn() != 2 || methodType.In(0) != serverCtxPtrType || !isPBPtr(methodType.In(1)) {
			return nil, fmt.Errorf("Method '%s' should take a *rpc.Context and a protobuf pointer", methodType.Name())
		}
		if methodType.NumOut() != 1 || methodType.Out(0) != errorType || !isPBPtr(methodType.Out(1)) {
			return nil, fmt.Errorf("Method '%s' should return a protobuf pointer and an error", methodType.Name())
		}
		svc.methods[methodType.Name()] = &method{
			requestType: methodType.In(1).Elem(),
			body:        m,
		}
	}
	go svc.logLoop()
	return svc, nil
}
