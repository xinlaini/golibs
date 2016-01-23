package rpc

import (
	"golang.org/x/net/context"

	"gen/pb/rpc/rpc_proto"
)

type ServerContext struct {
	context.Context
	Metadata *rpc_proto.RequestMetadata
}

type ClientContext struct {
	context.Context
	Metadata *rpc_proto.ResponseMetadata
}
