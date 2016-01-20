package rpc

import (
	"net/http"
	"os"
	"sync"

	xlog "github.com/xinlaini/golibs/log"
)

type Config struct {
	xlog.Logger
	BinaryLogDir string
	HTTPMux      *http.ServeMux
	Services     map[string]interface{}
}

type Controller struct {
	logger       xlog.Logger
	binaryLogDir string

	server     *server
	clients    []*Client
	mtxClients sync.RWMutex
}

func (ctrl *Controller) Serve(port int) error {
	return ctrl.server.serve(port)
}

func (ctrl *Controller) NewClient(opts ClientOptions) (*Client, error) {
	c, err := newClient(ctrl, &opts)
	if err != nil {
		return nil, err
	}
	ctrl.mtxClients.Lock()
	ctrl.clients = append(ctrl.clients, c)
	ctrl.mtxClients.Unlock()
	return c, nil
}

func (ctrl *Controller) showRPCs(w http.ResponseWriter, req *http.Request) {
}

func NewController(config Config) (*Controller, error) {
	ctrl := &Controller{
		logger:       config.Logger,
		binaryLogDir: config.BinaryLogDir,
	}

	var err error
	if config.BinaryLogDir != "" {
		if err = os.MkdirAll(config.BinaryLogDir, 0755); err != nil {
			return nil, err
		}
	}

	if ctrl.server, err = newServer(ctrl, config.Services); err != nil {
		return nil, err
	}
	if config.HTTPMux != nil {
		config.HTTPMux.HandleFunc("/rpcs", func(w http.ResponseWriter, req *http.Request) {
			ctrl.showRPCs(w, req)
		})
	}
	return ctrl, nil
}
