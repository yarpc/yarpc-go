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
	"crypto/tls"
	"errors"
	"net"
	"sync"

	"go.uber.org/net/metrics"
	"go.uber.org/zap"
)

var (
	errListenerClosed = errors.New("listener closed")
)

// Config describes how listener should be configured.
type Config struct {
	Listener  net.Listener
	TLSConfig *tls.Config

	ServiceName   string
	TransportName string
	Meter         *metrics.Scope
	Logger        *zap.Logger
}

// listener wraps original net listener and it accepts both TLS and non-TLS connections.
type listener struct {
	net.Listener

	tlsConfig *tls.Config
	observer  *observer
	logger    *zap.Logger

	serveOnce   sync.Once
	connChan    chan net.Conn
	stopChan    chan struct{}
	stoppedChan chan struct{}
}

// NewListener returns a multiplexed listener which accepts both TLS and
// plaintext connections.
func NewListener(c Config) net.Listener {
	return &listener{
		Listener:    c.Listener,
		tlsConfig:   c.TLSConfig,
		observer:    newObserver(c.Meter, c.Logger, c.ServiceName, c.TransportName),
		logger:      c.Logger,
		connChan:    make(chan net.Conn),
		stoppedChan: make(chan struct{}),
		stopChan:    make(chan struct{}),
	}
}

// Accept returns multiplexed plaintext connetion.
// After close, returned error is errListenerClosed.
func (l *listener) Accept() (net.Conn, error) {
	// Starts the connection server only once.
	l.serveOnce.Do(func() { go l.serve() })

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
	err := l.Listener.Close()
	close(l.stopChan)
	<-l.stoppedChan
	return err
}

// server accepts the connection from the underlying listener and creates a new
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

	return l.handlePlaintextConn(c), nil
}

// handleTLSConn completes the TLS handshake for the given connection and
// returns a TLS server wrapped plaintext connection.
func (l *listener) handleTLSConn(conn net.Conn) (net.Conn, error) {
	tlsConn := tls.Server(conn, l.tlsConfig)
	if err := tlsConn.Handshake(); err != nil {
		l.observer.incTLSHandshakeFailures()
		l.logger.Error("TLS handshake failed", zap.Error(err))
		return nil, err
	}

	l.observer.incTLSConnections(tlsConn.ConnectionState().Version)
	return tlsConn, nil
}

func (l *listener) handlePlaintextConn(conn net.Conn) net.Conn {
	l.observer.incPlaintextConnections()
	return conn
}

func matchTLSConnection(cs *connSniffer) (bool, error) {
	// TODO(jronak): set temporary connection read and write timeout.
	isTLS, err := isTLSClientHelloRecord(cs)
	if err != nil {
		return false, err
	}

	cs.stopSniffing()
	return isTLS, nil
}
