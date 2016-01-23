package rpc

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"time"

	"gen/pb/rpc/rpc_proto"

	"github.com/golang/protobuf/proto"
	"github.com/xinlaini/golibs/log"
)

const (
	recentEgressCount = 64
)

var (
	DefaultDialRetry = DialRetryPolicy{
		Sleep:    15 * time.Second,
		Backoff:  1.3,
		MaxSleep: 60 * time.Second,
	}
)

type DialRetryPolicy struct {
	Sleep    time.Duration
	Backoff  float64
	MaxSleep time.Duration
}

type ClientOptions struct {
	ServiceName  string
	ServiceAddr  string
	ConnPoolSize int
	Retry        DialRetryPolicy
}

type connEntry struct {
	conn           net.Conn
	localPort      string
	connectedSince time.Time
	idleSince      time.Time
}

type Client struct {
	logger      xlog.Logger
	serviceName string
	serviceAddr string

	entries         map[string]*connEntry
	mtxEntries      sync.RWMutex
	freeConns       chan *connEntry
	shouldConnect   chan struct{}
	closed          chan struct{}
	connectLoopDone chan struct{}
	logLoopDone     chan struct{}
	chLog           chan [3][]byte
	recentCalls     [recentEgressCount][3][]byte
	mtxRecentCalls  sync.RWMutex
}

func (c *Client) Call(
	methodName string,
	ctx *ClientContext,
	requestPB proto.Message,
	responseType reflect.Type) (proto.Message, error) {
	request := &rpc_proto.Request{
		Metadata: &rpc_proto.RequestMetadata{
			ClientJobName:   proto.String(os.Args[0]),
			ClientRequestId: proto.String(fmt.Sprintf("%x", time.Now().UnixNano())),
			ServiceName:     proto.String(c.serviceName),
			MethodName:      proto.String(methodName),
		},
	}
	// Propagate the deadline, if one is set.
	deadline, ok := ctx.Deadline()
	if ok {
		request.Metadata.TimeoutUs = proto.Int64(int64(deadline.Sub(time.Now())))
	}
	var err error
	if requestPB != nil {
		c.logger.Infof("%v, %v, %+v", requestPB != nil, requestPB == nil, requestPB)
		request.RequestPb, err = proto.Marshal(requestPB)
		if err != nil {
			return nil, makeErrf("Failed to marshal method request: %s", err)
		}
	}
	requestBytes, err := proto.Marshal(request)
	if err != nil {
		return nil, makeErrf("Failed to marshal RPC request: %s", err)
	}
	requestSize := make([]byte, 4)
	binary.BigEndian.PutUint32(requestSize, uint32(len(requestBytes)))
	responseBytes, err := c.runNetIO(ctx, requestSize, requestBytes)
	if err != nil {
		return nil, err
	}

	go c.log(requestSize, requestBytes, responseBytes)

	response := &rpc_proto.Response{}
	if err = proto.Unmarshal(responseBytes[4:], response); err != nil {
		return nil, makeErrf("Failed to unmarshal RPC response: %s", err)
	}
	ctx.Metadata = response.Metadata
	if response.Error != nil {
		// Copy the response error verbatim.
		return nil, errors.New(response.GetError())
	}
	if response.ResponsePb == nil {
		return nil, nil
	}
	responsePB := reflect.New(responseType).Interface().(proto.Message)
	if err = proto.Unmarshal(response.ResponsePb, responsePB); err != nil {
		return nil, makeErrf("Failed to unmarshal method response: %s", err)
	}
	return responsePB, nil
}

func (c *Client) Close() {
	close(c.closed)
	<-c.connectLoopDone
	<-c.logLoopDone

	// Close all the connections.
	c.mtxEntries.Lock()
	for _, entry := range c.entries {
		c.logger.Infof("Closing connection from local port '%s' to '%s'", entry.localPort, c.serviceAddr)
		entry.conn.Close()
	}
	c.mtxEntries.Unlock()

	c.logger.Infof("Client for '%s' to '%s' is closed", c.serviceName, c.serviceAddr)
}

func (c *Client) log(requestSize, requestBytes, responseBytes []byte) {
	c.chLog <- [3][]byte{requestSize, requestBytes, responseBytes}
}

func (c *Client) runNetIO(ctx *ClientContext, requestSize, requestBytes []byte) ([]byte, error) {
	select {
	case <-c.closed:
		return nil, makeErr("Client is closed")
	case <-ctx.Done():
		return nil, makeErr(ctx.Err().Error())
	case entry := <-c.freeConns:
		responseBytes, err := roundtrip(ctx, entry.conn, requestSize, requestBytes)
		if err != nil {
			// Connection is not reusable, must discard.
			c.logger.Errorf(
				"Connection from local port '%s' to '%s' is unreusable (error: %s), discarding...",
				entry.localPort,
				c.serviceAddr,
				err)
			entry.conn.Close()

			c.mtxEntries.Lock()
			delete(c.entries, entry.localPort)
			c.mtxEntries.Unlock()

			// Signal connectLoop to re-establish a new connection.
			c.shouldConnect <- struct{}{}
		} else {
			// No error, push the connection back to the pool.
			c.mtxEntries.Lock()
			entry.idleSince = time.Now()
			c.mtxEntries.Unlock()

			c.freeConns <- entry
		}
		return responseBytes, err
	}
}

func (c *Client) connectWithRetry(opts *ClientOptions) *connEntry {
	// Retry loop for one connection.
	sleep := opts.Retry.Sleep
	for {
		select {
		case <-c.closed:
			return nil
		default:
			conn, err := net.Dial("tcp", c.serviceAddr)
			if err != nil {
				c.logger.Errorf("Failed to dial to '%s', will retry after '%s'", c.serviceAddr, sleep.String())
				time.Sleep(sleep)
				// Exponential backoff with cap.
				if sleep = time.Duration(float64(sleep) * opts.Retry.Backoff); sleep > opts.Retry.MaxSleep {
					sleep = opts.Retry.MaxSleep
				}
				continue
			}
			_, localPort, _ := net.SplitHostPort(conn.LocalAddr().String())
			c.logger.Infof("Established connection from local port '%s' to '%s'", localPort, c.serviceAddr)
			now := time.Now()
			return &connEntry{
				conn:           conn,
				localPort:      localPort,
				connectedSince: now,
				idleSince:      now,
			}
		}
	}
}

func (c *Client) connectLoop(opts *ClientOptions) {
	defer c.logger.Infof("Quitting connectLoop for remote address '%s'", c.serviceAddr)

	for {
		select {
		case <-c.closed:
			return
		case <-c.shouldConnect:
			entry := c.connectWithRetry(opts)
			if entry == nil {
				return
			}
			c.mtxEntries.Lock()
			if _, found := c.entries[entry.localPort]; found {
				c.mtxEntries.Unlock()
				log.Panicf("Entry with local port '%s' already exists", entry.localPort)
			}
			c.entries[entry.localPort] = entry
			c.mtxEntries.Unlock()

			// Make this entry available for consumption.
			c.freeConns <- entry
		}
	}
}

func (c *Client) logLoop(binaryLogDir, serviceName string) {
	var (
		binaryLog *os.File
		err       error
	)
	if binaryLogDir != "" {
		logName := filepath.Join(
			binaryLogDir, fmt.Sprintf("%s-egress-%x.log", serviceName, time.Now().UnixNano()))
		binaryLog, err = os.Create(logName)
		if err != nil {
			c.logger.Errorf("Failed to create binary log '%s': %s", logName, err)
		}
	}

	next := 0
	for {
		select {
		case <-c.closed:
			if binaryLog != nil {
				c.logger.Infof("Binary log '%s' is now closed", binaryLog.Name())
				binaryLog.Close()
			}
			return
		case data := <-c.chLog:
			c.mtxRecentCalls.Lock()
			c.recentCalls[next] = data
			c.mtxRecentCalls.Unlock()
			next = (next + 1) % recentEgressCount
			if binaryLog != nil {
				for i := 0; i < 3; i++ {
					if _, err := binaryLog.Write(data[i]); err != nil {
						c.logger.Errorf(
							"Failed to write to '%s', it's now closed and may be compromised: %s",
							binaryLog.Name(), err)
						binaryLog.Close()
						binaryLog = nil
						break
					}
				}
			}
		}
	}
}

func validateOpts(opts *ClientOptions) error {
	if opts.ConnPoolSize <= 0 {
		return errors.New("ClientOptions.ConnPoolSize must be >0")
	}
	if opts.Retry.Sleep <= 0 {
		return errors.New("ClientOptions.Retry.Sleep must be >0")
	}
	if opts.Retry.Backoff <= 1.0 {
		return errors.New("ClientOptions.Retry.Backoff must be >1.0")
	}
	if opts.Retry.MaxSleep < opts.Retry.Sleep {
		return errors.New("ClientOptions.Retry.MaxSleep must be > Sleep")
	}
	return nil
}

func newClient(ctrl *Controller, opts *ClientOptions) (*Client, error) {
	if err := validateOpts(opts); err != nil {
		return nil, err
	}
	c := &Client{
		logger:          ctrl.logger,
		serviceName:     opts.ServiceName,
		serviceAddr:     opts.ServiceAddr,
		entries:         make(map[string]*connEntry),
		freeConns:       make(chan *connEntry, opts.ConnPoolSize),
		shouldConnect:   make(chan struct{}, opts.ConnPoolSize),
		closed:          make(chan struct{}),
		connectLoopDone: make(chan struct{}),
		logLoopDone:     make(chan struct{}),
		chLog:           make(chan [3][]byte),
	}

	go func() {
		c.connectLoop(opts)
		close(c.connectLoopDone)
	}()

	for i := 0; i < opts.ConnPoolSize; i++ {
		c.shouldConnect <- struct{}{}
	}
	go func() {
		c.logLoop(ctrl.binaryLogDir, opts.ServiceName)
		close(c.logLoopDone)
	}()
	return c, nil
}

func roundtrip(ctx *ClientContext, conn net.Conn, requestSize, requestBytes []byte) ([]byte, error) {
	deadline, ok := ctx.Deadline()
	if !ok {
		deadline = time.Time{}
	}

	var err error
	if err = conn.SetDeadline(deadline); err != nil {
		return nil, makeErr(err.Error())
	}
	if _, err = conn.Write(requestSize); err != nil {
		return nil, makeErrf("Failed to write 4 bytes for request size: %s", err)
	}
	if _, err = conn.Write(requestBytes); err != nil {
		return nil, makeErrf("Failed to write %d bytes for request: %s", err)
	}
	buf := bytes.NewBuffer(make([]byte, 0, 4))
	if _, err = io.CopyN(buf, conn, 4); err != nil {
		return nil, makeErrf(
			"Failed to read 4 bytes for response size from '%s': %s",
			conn.RemoteAddr().String(), err)
	}
	responseSize := binary.BigEndian.Uint32(buf.Bytes())
	if _, err = io.CopyN(buf, conn, int64(responseSize)); err != nil {
		return nil, makeErrf(
			"Failed to read %d bytes for response from '%s': %s",
			conn.RemoteAddr().String(), err)
	}
	return buf.Bytes(), nil
}
