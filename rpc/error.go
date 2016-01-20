package rpc

import (
	"errors"
	"fmt"

	"github.com/golang/protobuf/proto"
)

const (
	rpcErrPrefix = "[RPC_ERROR] "
)

func makeErr(str string) error {
	return errors.New(rpcErrPrefix + str)
}

func makePBErr(str string) *string {
	return proto.String(makeErr(str).Error())
}

func makeErrf(format string, v ...interface{}) error {
	return errors.New(rpcErrPrefix + fmt.Sprintf(format, v...))
}

func makePBErrf(format string, v ...interface{}) *string {
	return proto.String(makeErrf(format, v...).Error())
}
