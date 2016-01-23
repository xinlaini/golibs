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
		response.Error = makeServerErr("Request.Metadata is missing method_name")
		return
	}
	m, found := svc.methods[reqMeta.GetMethodName()]
	if !found {
		response.Error = makeServerErrf(
			"Method '%s.%s' is not found", reqMeta.GetServiceName(), reqMeta.GetMethodName())
		return
	}
	var requestPB reflect.Value
	if request.RequestPb == nil {
		requestPB = reflect.Zero(reflect.PtrTo(m.requestType))
	} else {
		requestPB = reflect.New(m.requestType)
		if err = proto.Unmarshal(request.RequestPb, requestPB.Interface().(proto.Message)); err != nil {
			response.Error = makeServerErrf(
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
		ch <- m.body.Call([]reflect.Value{reflect.ValueOf(ctx), requestPB})
	}()

	select {
	case callResults := <-ch:
		if callResults[1].IsNil() {
			if !callResults[0].IsNil() {
				if response.ResponsePb, err = proto.Marshal(callResults[0].Interface().(proto.Message)); err != nil {
					response.Error = makeServerErrf(
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
		response.Error = makeServerErrf(
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

func newService(ctrl *Controller, name string, cfg *ServiceConfig) (*service, error) {
	if cfg.Type.Kind() != reflect.Interface {
		return nil, fmt.Errorf("'%s' is not an Interface kind", cfg.Type)
	}

	if !reflect.TypeOf(cfg.Impl).Implements(cfg.Type) {
		return nil, fmt.Errorf("'%s' does not implement '%s'", reflect.TypeOf(cfg.Impl), cfg.Type)
	}

	svc := &service{
		logger:  ctrl.logger,
		methods: make(map[string]*method),
		chLog:   make(chan [3][]byte),
	}

	implValue := reflect.ValueOf(cfg.Impl)
	for i := 0; i < cfg.Type.NumMethod(); i++ {
		mName := cfg.Type.Method(i).Name
		m := implValue.MethodByName(mName)
		mType := m.Type()
		// Validate method signature.
		if mType.NumIn() != 2 || mType.In(0) != serverCtxPtrType || !isPBPtr(mType.In(1)) {
			continue
		}
		if mType.NumOut() != 2 || !isPBPtr(mType.Out(0)) || mType.Out(1) != errorType {
			continue
		}
		ctrl.logger.Infof("Will serve method '%s.%s'", name, mName)
		svc.methods[mName] = &method{
			requestType: mType.In(1).Elem(),
			body:        m,
		}
	}
	go svc.logLoop(ctrl.binaryLogDir, name)
	return svc, nil
}
