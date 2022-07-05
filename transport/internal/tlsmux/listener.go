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

package tlsmux

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"sync"
	"time"

	"go.uber.org/net/metrics"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/zap"
)

var (
	errListenerClosed = errors.New("listener closed")

	// TODO(jronak): below timeouts are experimental, will be tuned after testing.
	_sniffReadTimeout    = time.Second * 10
	_tlsHandshakeTimeout = time.Second * 10
)

// Config describes how listener should be configured.
type Config struct {
	Listener  net.Listener
	TLSConfig *tls.Config

	ServiceName   string
	TransportName string
	Meter         *metrics.Scope
	Logger        *zap.Logger
	Mode          transport.InboundTLSMode
}

// listener wraps original net listener and it accepts both TLS and non-TLS connections.
type listener struct {
	net.Listener

	tlsConfig *tls.Config
	observer  *observer
	logger    *zap.Logger
	mode      transport.InboundTLSMode

	closeOnce   sync.Once
	connChan    chan net.Conn
	stopChan    chan struct{}
	stoppedChan chan struct{}
}

// NewListener returns a multiplexed listener which accepts both TLS and
// plaintext connections.
func NewListener(c Config) net.Listener {
	if c.Mode == transport.Disabled {
		return c.Listener
	}

	lis := &listener{
		Listener:    c.Listener,
		tlsConfig:   c.TLSConfig,
		observer:    newObserver(c.Meter, c.Logger, c.ServiceName, c.TransportName, c.Mode),
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

	defer func() {
		wg.Wait()
		close(l.stoppedChan)
		close(l.connChan)
	}()

	for {
		conn, err := l.Listener.Accept()
		if err != nil {
			return
		}

		wg.Add(1)
		go l.serveConnection(conn, &wg)
	}
}

// serveConnection muxes the given connection and sends muxed connection to the
// connection channel.
func (l *listener) serveConnection(conn net.Conn, wg *sync.WaitGroup) {
	defer wg.Done()

	c, err := l.mux(conn)
	if err != nil {
		conn.Close()
		return
	}

	select {
	case l.connChan <- c:
	case <-l.stopChan:
		conn.Close()
	}
}

// mux accepts both plaintext and tls connection, and returns a plaintext
// connection.
func (l *listener) mux(conn net.Conn) (net.Conn, error) {
	c := newConnectionSniffer(conn)
	isTLS, err := matchTLSConnection(c)
	if err != nil {
		l.logger.Error("TLS connection matcher failed", zap.Error(err))
		return nil, err
	}

	if isTLS {
		return l.handleTLSConn(c)
	}

	return l.handlePlaintextConn(c)
}

// handleTLSConn completes the TLS handshake for the given connection and
// returns a TLS server wrapped plaintext connection.
func (l *listener) handleTLSConn(conn net.Conn) (net.Conn, error) {
	ctx, cancel := context.WithTimeout(context.Background(), _tlsHandshakeTimeout)
	defer cancel()

	tlsConn := tls.Server(conn, l.tlsConfig)
	if err := tlsConn.HandshakeContext(ctx); err != nil {
		l.observer.incTLSHandshakeFailures()
		l.logger.Error("TLS handshake failed", zap.Error(err))
		return nil, err
	}

	l.observer.incTLSConnections(tlsConn.ConnectionState().Version)
	return tlsConn, nil
}

func (l *listener) handlePlaintextConn(conn net.Conn) (net.Conn, error) {
	if l.mode == transport.Enforced {
		l.logger.Error("plaintext connection not allowed in enforced TLS mode: rejecting connection")
		l.observer.incPlaintextConnectionRejects()
		return nil, errors.New("plaintext connection not allowed in enforced TLS mode")
	}
	l.observer.incPlaintextConnections()
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
