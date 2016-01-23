package rpc

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"time"

	"gen/pb/rpc/rpc_proto"

	"golang.org/x/net/context"

	"github.com/golang/protobuf/proto"
	"github.com/xinlaini/golibs/log"
)

const (
	recentIngressCount = 64
)

var (
	serverCtxPtrType = reflect.TypeOf((*ServerContext)(nil))
	pbMessageType    = reflect.TypeOf((*proto.Message)(nil)).Elem()
	errorType        = reflect.TypeOf((*error)(nil)).Elem()
)

type method struct {
	requestType reflect.Type
	body        reflect.Value
	receiver    reflect.Value
}

type service struct {
	logger         xlog.Logger
	methods        map[string]*method
	chLog          chan [3][]byte
	recentCalls    [recentIngressCount][3][]byte
	mtxRecentCalls sync.RWMutex
}

func (svc *service) logLoop(binaryLogDir, name string) {
	var (
		binaryLog *os.File
		err       error
	)
	if binaryLogDir != "" {
		logName := filepath.Join(binaryLogDir, fmt.Sprintf("%s-ingress.log", name))
		binaryLog, err = os.Create(logName)
		if err != nil {
			svc.logger.Errorf("Failed to create binary log '%s': %s", logName, err)
		}
	}

	next := 0
	for {
		data := <-svc.chLog
		svc.mtxRecentCalls.Lock()
		svc.recentCalls[next] = data
		svc.mtxRecentCalls.Unlock()
		next = (next + 1) % recentIngressCount
		if binaryLog != nil {
			for i := 0; i < 3; i++ {
				if _, err := binaryLog.Write(data[i]); err != nil {
					svc.logger.Errorf(
						"Failed to write to '%s', it's now closed and may be compromised: %s",
						binaryLog.Name(), err)
					binaryLog.Close()
					binaryLog = nil
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
		response.Error = makePBErr("Request.Metadata is missing method_name")
		return
	}
	m, found := svc.methods[reqMeta.GetMethodName()]
	if !found {
		response.Error = makePBErrf(
			"Method '%s.%s' is not found", reqMeta.GetServiceName(), reqMeta.GetMethodName())
		return
	}
	var requestPB reflect.Value
	if request.RequestPb == nil {
		requestPB = reflect.Zero(reflect.PtrTo(m.requestType))
	} else {
		requestPB = reflect.New(m.requestType)
		if err = proto.Unmarshal(request.RequestPb, requestPB.Interface().(proto.Message)); err != nil {
			response.Error = makePBErrf(
				"Failed to unmarshal request for '%s.%s': %s",
				reqMeta.GetServiceName(),
				reqMeta.GetMethodName(),
				err)
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
		ch <- m.body.Call([]reflect.Value{m.receiver, reflect.ValueOf(ctx), requestPB})
	}()

	select {
	case callResults := <-ch:
		if callResults[1].IsNil() {
			if !callResults[0].IsNil() {
				if response.ResponsePb, err = proto.Marshal(callResults[0].Interface().(proto.Message)); err != nil {
					response.Error = makePBErrf(
						"Failed to marshal response for '%s.%s': %s",
						reqMeta.GetServiceName(),
						reqMeta.GetMethodName(),
						err)
					return
				}
			}
		} else {
			// This is an app-level error.
			response.Error = proto.String(callResults[1].Interface().(error).Error())
		}
	case <-ctx.Done():
		response.Error = makePBErrf(
			"Method '%s.%s' timed out", reqMeta.GetServiceName(), reqMeta.GetMethodName())
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
	svc := &service{
		logger:  ctrl.logger,
		methods: make(map[string]*method),
		chLog:   make(chan [3][]byte),
	}

	typ := reflect.TypeOf(impl)
	for i := 0; i < typ.NumMethod(); i++ {
		m := typ.Method(i)
		if m.PkgPath != "" {
			// This is unexported method, ignore it.
			continue
		}
		// Validate method signature.
		if m.Type.NumIn() != 3 || m.Type.In(1) != serverCtxPtrType || !isPBPtr(m.Type.In(2)) {
			continue
		}
		if m.Type.NumOut() != 2 || !isPBPtr(m.Type.Out(0)) || m.Type.Out(1) != errorType {
			continue
		}
		ctrl.logger.Infof("Will serve method '%s.%s'", name, m.Name)
		svc.methods[m.Name] = &method{
			requestType: m.Type.In(2).Elem(),
			body:        m.Func,
			receiver:    reflect.ValueOf(impl),
		}
	}
	go svc.logLoop(ctrl.binaryLogDir, name)
	return svc, nil
}
