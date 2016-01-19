package rpc

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"

	xlog "github.com/xinlaini/golibs/log"
)

type Config struct {
	xlog.Logger
	BinaryLogDir string
	HTTPMux      *http.ServeMux
	Services     map[string]interface{}
}

type Controller struct {
	logger xlog.Logger

	services map[string]service
}

func (ctrl *Controller) Serve(port int) error {
	l, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return err
	}
	defer l.Close()
	ctrl.logger.Infof("Start listening on TCP port %d...", port)

	for {
		conn, err := l.Accept()
		if err != nil {
			ctrl.logger.Errorf("TCP accept failed with error: %s", err)
			continue
		}
		go ctrl.handleConn(conn)
	}
}

func (ctrl *Controller) handleConn(conn net.Conn) {
	// If this function is returned, the connection must have lost its integrity.
	defer conn.Close()
	var err error

	for {
		requestBytes := ctrl.readRequest(conn)
		if requestBytes == nil {
			return
		}
		responseBytes := ctrl.serveRequest(requestBytes)
		if responseBytes == nil {
			return
	}
}

func (ctrl *Controller) readRequest(conn net.Conn) []byte {
	buf := bytes.NewBuffer(make([]byte, 4))
	if _, err := io.CopyN(buf, conn, 4); err != nil {
		ctrl.logger.Errorf("Failed to read 4 bytes for request size: %s", err)
		return nil
	}
	requestSize := binary.BigEndian.Uint32(buf.Bytes())
	if _, err = io.CopyN(buf, conn, int64(requestSize)); err != nil {
		ctrl.logger.Errorf("Failed to read %d bytes for request: %s", err)
		return nil
	}
	return buf.Bytes()
}

func NewController(config Config) (*Controller, error) {
	ctrl := &Controller{
		logger: config.Logger,
	}

	var err error
	if err = os.MkdirAll(config.BinaryLogDir, 0755); err != nil {
		return nil, err
	}

	for name, impl := range config.Services {
		svc, err := newService(logger, config.BinaryLogDir, name, impl)
		if err != nil {
			return nil, err
		}
		ctrl.services[name] = svc
	}
	if config.HTTPMux != nil {
		config.HTTPMux.HandleFunc("/rpcs", func(w http.ResponseWriter, req *http.Request) {
			ctrl.showRPCs(w, req)
		})
	}
	return ctrl, nil
}
