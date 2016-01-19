package rpc

import (
	"golang.org/x/net/context"

	"github.com/xinlaini/golibs/rpc/proto/gen-go"
)

type ServerContext struct {
	context.Context
	Metadata *rpc_proto.RequestMetadata
}

type ClientContext struct {
	context.Context
	Metadata *rpc_proto.ResponseMetadata
}
