package rpc

import (
	"os"

	xlog "github.com/xinlaini/golibs/log"
)

type ClientOptions struct {
	ServiceName  string
	ServiceAddr  string
	ConnPoolSize int
}

type Client struct {
	logger    xlog.Logger
	binaryLog *os.File
}
