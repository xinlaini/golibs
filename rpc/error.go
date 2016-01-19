package rpc

import (
	"fmt"

	"github.com/golang/protobuf/proto"
)

const (
	rpcErrPrefix = "[RPC_ERROR] "
)

func makeErr(str string) *string {
	return proto.String(rpcErrPrefix + str)
}

func makeErrf(format string, v ...interface{}) *string {
	return proto.String(rpcErrPrefix + fmt.Sprintf(format, v...))
}
