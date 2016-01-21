package rpc

import (
	"golang.org/x/net/context"

	"gen/pb/rpc"
)

type ServerContext struct {
	context.Context
	Metadata *rpc_proto.RequestMetadata
}

type ClientContext struct {
	context.Context
	Metadata *rpc_proto.ResponseMetadata
}
