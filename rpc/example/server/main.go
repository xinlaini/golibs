package main

import (
	"errors"
	"flag"
	"fmt"

	"gen/pb/rpc/example/say_proto"
	"gen/pb/rpc/example/sing_proto"

	"github.com/golang/protobuf/proto"
	"github.com/xinlaini/golibs/log"
	"github.com/xinlaini/golibs/rpc"
)

type helloService struct {
	logger xlog.Logger
}

func (svc *helloService) Say(ctx *rpc.ServerContext, req *say.Request) (*say.Response, error) {
	if req == nil {
		svc.logger.Info("Received nil say request")
		return nil, nil
	}
	svc.logger.Infof("Request metadata:\n%s", ctx.Metadata.String())
	svc.logger.Infof("Request:\n%s", req.String())
	if req.Hdr == nil {
		return nil, errors.New("Missing header")
	}

	return &say.Response{
		Msg: proto.String(fmt.Sprintf("Say received %s", req.String())),
	}, nil
}

func (svc *helloService) Sing(ctx *rpc.ServerContext, req *sing.Request) (*sing.Response, error) {
	if req == nil {
		svc.logger.Info("Received nil sing request")
		return nil, nil
	}
	svc.logger.Infof("Request metadata:\n%s", ctx.Metadata.String())
	svc.logger.Infof("Request:\n%s", req.String())
	if req.Hdr == nil {
		return nil, errors.New("Missing header")
	}

	return &sing.Response{
		Msg: proto.String(fmt.Sprintf("Sing received %s", req.String())),
	}, nil
}

func main() {
	flag.Parse()

	logger := xlog.NewConsoleLogger()

	ctrl, err := rpc.NewController(rpc.Config{
		Logger:   logger,
		Services: map[string]interface{}{"Hello": &helloService{logger: logger}},
	})
	if err != nil {
		logger.Fatal("Failed to create controller: %s", err)
	}
	if err = ctrl.Serve(9090); err != nil {
		logger.Fatal("Failed to start server: %s", err)
	}
}
