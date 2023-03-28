// Copyright (c) 2022 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package muxlistener

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"sync"
	"time"

	"go.uber.org/net/metrics"
	yarpctls "go.uber.org/yarpc/api/transport/tls"
	tlsmetrics "go.uber.org/yarpc/transport/internal/tls/metrics"
	"go.uber.org/zap"
)

var (
	errListenerClosed = errors.New("listener closed")

	// Connection has 15s to transmit first 5 bytes for sniffing.
	_sniffReadTimeout = time.Second * 15
	// Handshake timeout set to 120s, see gRPC-go:
	// https://github.com/grpc/grpc-go/blob/fdc5d2f3da856f3cdd3483280ae33da5bee17a93/server.go#L467
	_tlsHandshakeTimeout = time.Second * 120
)

// Config describes how listener should be configured.
type Config struct {
	Listener  net.Listener
	TLSConfig *tls.Config

	ServiceName   string
	TransportName string
	Meter         *metrics.Scope
	Logger        *zap.Logger
	Mode          yarpctls.Mode
}

// listener wraps original net listener and it accepts both TLS and non-TLS connections.
type listener struct {
	net.Listener

	tlsConfig *tls.Config
	observer  *tlsmetrics.Observer
	logger    *zap.Logger
	mode      yarpctls.Mode

	closeOnce   sync.Once
	connChan    chan net.Conn
	stopChan    chan struct{}
	stoppedChan chan struct{}
}

// NewListener returns a multiplexed listener which accepts both TLS and
// plaintext connections.
func NewListener(c Config) net.Listener {
	if c.Mode == yarpctls.Disabled {
		return c.Listener
	}

	observer := tlsmetrics.NewObserver(tlsmetrics.Params{
		Meter:         c.Meter,
		Logger:        c.Logger,
		ServiceName:   c.ServiceName,
		TransportName: c.TransportName,
		Mode:          c.Mode,
		Direction:     "inbound",
	})

	lis := &listener{
		Listener:    c.Listener,
		tlsConfig:   c.TLSConfig,
		observer:    observer,
		logger:      c.Logger,
		connChan:    make(chan net.Conn),
		stoppedChan: make(chan struct{}),
		stopChan:    make(chan struct{}),
		mode:        c.Mode,
	}

	// Starts go routine for the connection server
	go lis.serve()

	return lis
}

// Accept returns multiplexed plaintext connetion.
// After close, returned error is errListenerClosed.
func (l *listener) Accept() (net.Conn, error) {
	select {
	case conn, ok := <-l.connChan:
		if !ok {
			return nil, errListenerClosed
		}
		return conn, nil
	case <-l.stopChan:
		return nil, errListenerClosed
	}
}

// Close closes the listener and waits until the connection server drains
// accepted connections and stops the server.
func (l *listener) Close() error {
	var err error
	l.closeOnce.Do(func() {
		err = l.Listener.Close()
		close(l.stopChan)
		<-l.stoppedChan
	})
	return err
}

// serve starts accepting the connection from the underlying listener and creates a new
// go routine for each connection for async muxing.
func (l *listener) serve() {
	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		cancel()
		wg.Wait()
		close(l.connChan)
		close(l.stoppedChan)
	}()

	for {
		conn, err := l.Listener.Accept()
		if err != nil {
			return
		}

		wg.Add(1)
		go l.serveConnection(ctx, conn, &wg)
	}
}

// serveConnection muxes the given connection and sends muxed connection to the
// connection channel.
func (l *listener) serveConnection(ctx context.Context, conn net.Conn, wg *sync.WaitGroup) {
	defer wg.Done()

	c, err := l.mux(ctx, conn)
	if err != nil {
		conn.Close()
		return
	}

	select {
	case l.connChan <- c:
	case <-l.stopChan:
		c.Close()
	}
}

// mux accepts both plaintext and tls connection, and returns a plaintext
// connection.
func (l *listener) mux(ctx context.Context, conn net.Conn) (net.Conn, error) {
	if l.mode == yarpctls.Enforced {
		return l.handleTLSConn(ctx, conn)
	}

	c := newConnectionSniffer(conn)
	isTLS, err := matchTLSConnection(c)
	if err != nil {
		l.logger.Error("TLS connection matcher failed", zap.Error(err))
		return nil, err
	}

	if isTLS {
		return l.handleTLSConn(ctx, c)
	}

	return l.handlePlaintextConn(c)
}

// handleTLSConn completes the TLS handshake for the given connection and
// returns a TLS server wrapped plaintext connection.
func (l *listener) handleTLSConn(ctx context.Context, conn net.Conn) (net.Conn, error) {
	ctx, cancel := context.WithTimeout(ctx, _tlsHandshakeTimeout)
	defer cancel()

	tlsConn := tls.Server(conn, l.tlsConfig)
	if err := tlsConn.HandshakeContext(ctx); err != nil {
		l.observer.IncTLSHandshakeFailures()
		l.logger.Error("TLS handshake failed", zap.Error(err))
		return nil, err
	}

	l.observer.IncTLSConnections(tlsConn.ConnectionState().Version)
	return tlsConn, nil
}

func (l *listener) handlePlaintextConn(conn net.Conn) (net.Conn, error) {
	l.observer.IncPlaintextConnections()
	return conn, nil
}

func matchTLSConnection(cs *connSniffer) (bool, error) {
	if err := cs.SetReadDeadline(time.Now().Add(_sniffReadTimeout)); err != nil {
		return false, err
	}

	// Reset read deadline after sniffing. See below:
	// https://github.com/golang/go/blob/be0b2a393a5a7297a3c8f42ca7d5ad3e4b15dcbe/src/net/http/server.go#L1887
	defer func() {
		_ = cs.SetReadDeadline(time.Time{})
	}()

	isTLS, err := isTLSClientHelloRecord(cs)
	if err != nil {
		return false, err
	}

	cs.stopSniffing()
	return isTLS, nil
}
