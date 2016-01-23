package rpc

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"

	"gen/pb/rpc/rpc_proto"

	"github.com/golang/protobuf/proto"
	"github.com/xinlaini/golibs/log"
)

type server struct {
	logger   xlog.Logger
	services map[string]*service
}

func (svr *server) serve(port int) error {
	if len(svr.services) == 0 {
		return errors.New("No service to serve")
	}
	l, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return err
	}
	defer l.Close()
	svr.logger.Infof("Start listening on TCP port %d...", port)

	for {
		conn, err := l.Accept()
		if err != nil {
			svr.logger.Errorf("TCP accept failed with error: %s", err)
			continue
		}
		go svr.handleConn(conn)
	}
}

func (svr *server) handleConn(conn net.Conn) {
	// If this function returns, the connection must have lost its integrity.
	defer conn.Close()

	for {
		requestBytes := svr.readRequest(conn)
		if requestBytes == nil {
			return
		}
		response, svc := svr.serveRequest(conn, requestBytes[4:])
		responseBytes, err := proto.Marshal(response)
		if err != nil {
			svr.logger.Errorf("Failed to marshal response: %s", err)
			return
		}
		responseSize := make([]byte, 4)
		binary.BigEndian.PutUint32(responseSize, uint32(len(responseBytes)))

		if _, err = conn.Write(responseSize); err != nil {
			svr.logger.Errorf("Failed to write 4 bytes for response size: %s", err)
			return
		}
		if _, err = conn.Write(responseBytes); err != nil {
			svr.logger.Errorf("Failed to write %d bytes for response: %s", err)
			return
		}
		if svc != nil {
			go svc.log(requestBytes, responseSize, responseBytes)
		}
	}
}

func (svr *server) serveRequest(conn net.Conn, requestBytes []byte) (*rpc_proto.Response, *service) {
	request := &rpc_proto.Request{}
	response := &rpc_proto.Response{}

	var err error
	if err = proto.Unmarshal(requestBytes, request); err != nil {
		response.Error = makePBErrf("Failed to unmarshal request: %s", err)
		return response, nil
	}
	if request.Metadata == nil {
		response.Error = makePBErr("Request is missing metadata")
		return response, nil
	}
	request.Metadata.ClientAddr = proto.String(conn.RemoteAddr().String())
	if request.Metadata.ServiceName == nil {
		response.Error = makePBErr("Request.Metadata is missing service_name")
		return response, nil
	}
	svc, found := svr.services[request.Metadata.GetServiceName()]
	if !found {
		response.Error = makePBErrf("Service '%s' is not found", request.Metadata.GetServiceName())
		return response, nil
	}
	svc.serveRequest(request, response)
	return response, svc
}

func (svr *server) readRequest(conn net.Conn) []byte {
	buf := bytes.NewBuffer(make([]byte, 0, 4))
	var err error
	if _, err = io.CopyN(buf, conn, 4); err != nil {
		svr.logger.Errorf(
			"Failed to read 4 bytes for request size from '%s': %s",
			conn.RemoteAddr().String(), err)
		return nil
	}
	requestSize := binary.BigEndian.Uint32(buf.Bytes())
	if _, err = io.CopyN(buf, conn, int64(requestSize)); err != nil {
		svr.logger.Errorf(
			"Failed to read %d bytes for request from '%s': %s",
			conn.RemoteAddr().String(), err)
		return nil
	}
	return buf.Bytes()
}

func newServer(ctrl *Controller, services map[string]interface{}) (*server, error) {
	svr := &server{
		logger:   ctrl.logger,
		services: make(map[string]*service),
	}
	for name, impl := range services {
		svc, err := newService(ctrl, name, impl)
		if err != nil {
			return nil, err
		}
		svr.services[name] = svc
	}
	return svr, nil
}
