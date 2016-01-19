package rpc

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"

	"github.com/golang/protobuf/proto"
	xlog "github.com/xinlaini/golibs/log"
	"github.com/xinlaini/golibs/rpc/proto/gen-go"
)

type Config struct {
	xlog.Logger
	BinaryLogDir string
	HTTPMux      *http.ServeMux
	Services     map[string]interface{}
}

type Controller struct {
	logger xlog.Logger

	services map[string]*service
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
	// If this function returns, the connection must have lost its integrity.
	defer conn.Close()

	for {
		requestBytes := ctrl.readRequest(conn)
		if requestBytes == nil {
			return
		}
		response, svc := ctrl.serveRequest(conn, requestBytes[4:])
		responseBytes, err := proto.Marshal(response)
		if err != nil {
			ctrl.logger.Errorf("Failed to marshal response: %s", err)
			return
		}
		responseSize := make([]byte, 4)
		binary.BigEndian.PutUint32(responseSize, uint32(len(responseBytes)))

		if _, err = conn.Write(responseSize); err != nil {
			ctrl.logger.Errorf("Failed to write 4 bytes for response size: %s", err)
			return
		}
		if _, err = conn.Write(responseBytes); err != nil {
			ctrl.logger.Errorf("Failed to write %d bytes for response: %s", err)
			return
		}
		if svc != nil {
			go svc.log(requestBytes, responseSize, responseBytes)
		}
	}
}

func (ctrl *Controller) serveRequest(conn net.Conn, requestBytes []byte) (*rpc_proto.Response, *service) {
	request := &rpc_proto.Request{}
	response := &rpc_proto.Response{}

	var err error
	if err = proto.Unmarshal(requestBytes, request); err != nil {
		response.Error = makeErrf("Failed to unmarshal request: %s", err)
		return response, nil
	}
	if request.Metadata == nil {
		response.Error = makeErr("Request is missing metadata")
		return response, nil
	}
	request.Metadata.ClientAddr = proto.String(conn.RemoteAddr().String())
	if request.Metadata.ServiceName == nil {
		response.Error = makeErr("Request.Metadata is missing service_name")
		return response, nil
	}
	svc, found := ctrl.services[request.Metadata.GetServiceName()]
	if !found {
		response.Error = makeErrf("Service '%s' is not found", request.Metadata.GetServiceName())
		return response, nil
	}
	svc.serveRequest(request, response)
	return response, svc
}

func (ctrl *Controller) readRequest(conn net.Conn) []byte {
	buf := bytes.NewBuffer(make([]byte, 4))
	var err error
	if _, err = io.CopyN(buf, conn, 4); err != nil {
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

func (ctrl *Controller) showRPCs(w http.ResponseWriter, req *http.Request) {
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
		svc, err := newService(ctrl.logger, config.BinaryLogDir, name, impl)
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
