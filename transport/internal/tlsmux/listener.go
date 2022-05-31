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
	"net"
)

// listener wraps original net listener and it accepts both TLS and non-TLS connections.
type listener struct {
	net.Listener
	tlsConfig *tls.Config
}

// NewListener returns a multiplexed listener which accepts both TLS and non-TLS connections.
func NewListener(lis net.Listener, tlsConfig *tls.Config) net.Listener {
	return &listener{
		Listener:  lis,
		tlsConfig: tlsConfig,
	}
}

// Accept returns the multiplexed connetions.
func (l *listener) Accept() (net.Conn, error) {
	conn, err := l.Listener.Accept()
	if err != nil {
		return nil, err
	}

	// TODO(jronak): avoid slow connections causing head of the line blocking by spawning
	// connection processing in separate routine.

	return l.handle(conn)
}

func (l *listener) handle(conn net.Conn) (net.Conn, error) {
	cs := &connSniffer{Conn: conn}
	isTLS, err := matchTLSConnection(cs)
	if err != nil {
		return nil, err
	}

	if isTLS {
		// TODO(jronak): initiate tls handshake to catch tls errors and
		// version metrics.
		return tls.Server(cs, l.tlsConfig), nil
	}

	return cs, nil
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
