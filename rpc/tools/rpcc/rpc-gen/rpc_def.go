package hello

import "github.com/xinlaini/rpctest/server/proto/gen-go"
import host_proto "github.com/xinlaini/cluster/host/proto/gen-go"
import "github.com/xinlaini/golibs/rpc"

type HelloService interface {
	Say(ctx *rpc.ServerContext, req *hello_proto.HelloRequest) (*hello_proto.HelloResponse, error)
	Say2(ctx *rpc.ServerContext, req *hello_proto.HelloRequest) (*hello_proto.HelloResponse, error)
}

type HelloClient struct {
	rpcClient *rpc.Client
}

func NewHelloClient(ctrl *rpc.Controller, opts rpc.ClientOptions) (*HelloClient, error) {
	rpcClient, err := ctrl.NewClient(opts)
	if err != nil {
		return nil, err
	}
	return &HelloClient{rpcClient: rpcClient}, nil
}

func (c *HelloClient) Say(ctx *rpc.ClientContext, req *hello_proto.HelloRequest) (*hello_proto.HelloResponse, err) {
}

func (c *HelloClient) Say2(ctx *rpc.ClientContext, req *hello_proto.HelloRequest) (*hello_proto.HelloResponse, err) {
}

