package rpc

import (
	"fmt"
	"log"
	"math"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"

	xlog "github.com/xinlaini/golibs/log"
)

type DialRetryPolicy struct {
	Sleep    time.Duration
	Backoff  float64
	MaxSleep time.Duration
}

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
	shouldStop      chan struct{}
	connectLoopDone chan struct{}
	logLoopDone     chan struct{}
	chLog           chan [3][]byte
	recentCalls     [recentEgressCount][3][]byte
	mtxRecentCalls  sync.RWMutex
}

func newClient(ctrl *Controller, opts *ClientOptions) (*Client, error) {
	binaryLog, err := os.Create(filepath.Join(
		ctrl.binaryLogDir, fmt.Sprintf("%s-egress-%x.log", opts.ServiceName, time.Now().UnixNano())))
	if err != nil {
		return nil, err
	}

	c := &Client{
		logger:          ctrl.logger,
		serviceName:     opts.ServiceName,
		serviceAddr:     opts.ServiceAddr,
		entries:         make(map[string]*connEntry),
		freeConns:       make(chan *connEntry, math.Max(1, opts.ConnPoolSize)),
		shouldConnect:   make(chan struct{}),
		shouldStop:      make(chan struct{}),
		connectLoopDone: make(chan struct{}),
		logLoopDone:     make(chan struct{}),
		chLog:           make(chan [3][]byte),
	}

	go func() {
		c.connectLoop(opts)
		close(c.connectLoopDone)
	}()

	go func() {
		c.logLoop(ctrl.binaryLogDir, opts.serviceName)
		close(c.logLoopDone)
	}()
}

func (c *Client) connectWithRetry(opts *ClientOptions) *connEntry {
	// Retry loop for one connection.
	sleep := opts.Retry.Sleep
	for {
		select {
		case <-c.shouldStop:
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
		case <-c.shouldStop:
			return
		case <-c.shouldConnect:
			entry := c.connectWithRetry(opts)
			if entry == nil {
				return
			}
			entriesCount := 0
			c.mtxEntries.Lock()
			if _, found := c.entries[entry.localPort]; found {
				c.mtxEntries.Unlock()
				log.Panicf("Entry with local port '%s' already exists", entry.localPort)
			}
			c.entries[entry.localPort] = entry
			entriesCount = len(c.entries)
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
			binaryLogDir, fmt.Sprintf("%s-egress-%x.log"), serviceName, time.Now().UnixNano())
		binaryLog, err = os.Create(logName)
		if err != nil {
			c.logger.Errorf("Failed to create binary log '%s': %s", logName, err)
		}
	}

	next := 0
	for {
		select {
		case <-c.shouldStop:
			if binaryLog != nil {
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
						svc.logger.Errorf(
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
