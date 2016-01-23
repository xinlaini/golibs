package rpc

import (
	"errors"
	"fmt"

	"github.com/golang/protobuf/proto"
)

const (
	serverPrefix = "[RPC_SERVER_ERROR] "
	clientPrefix = "[RPC_CLIENT_ERROR] "
)

func makeErr(prefix, err string) error {
	return errors.New(prefix + err)
}

func makeErrf(prefix, format string, v ...interface{}) error {
	return errors.New(prefix + fmt.Sprintf(format, v...))
}

func makeClientErr(err string) error {
	return makeErr(clientPrefix, err)
}

func makeServerErr(err string) *string {
	return proto.String(makeErr(serverPrefix, err).Error())
}

func makeClientErrf(format string, v ...interface{}) error {
	return errors.New(clientPrefix + fmt.Sprintf(format, v...))
}

func makeServerErrf(format string, v ...interface{}) *string {
	return proto.String(makeErrf(serverPrefix, format, v...).Error())
}
