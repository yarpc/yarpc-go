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
	"time"
)

// mux provides a listener which accepts both TLS and non-TLS connections.
type mux struct {
	net.Listener
	tlsConfig *tls.Config
}

// NewListener returns a multiplexed listener which accepts both TLS and non-TLS connections.
func NewListener(listener net.Listener, tlsConfig *tls.Config) net.Listener {
	return &mux{
		Listener:  listener,
		tlsConfig: tlsConfig,
	}
}

// Accept returns the multiplexed connetions.
func (m *mux) Accept() (net.Conn, error) {
	conn, err := m.Listener.Accept()
	if err != nil {
		return nil, err
	}

	// TODO(jronak): avoid slow connections causing head of the line blocking by spawning
	// connection processing in separate routine.

	return m.handle(conn)
}

func (m *mux) handle(conn net.Conn) (net.Conn, error) {
	cs := &connSniffer{Conn: conn}
	// TODO(jronak): set temporary connection read and write timeout.
	cs.SetReadDeadline(time.Now().Add(time.Second * 2))
	isTLS, err := isTLSClientHelloRecord(cs)
	if err != nil {
		return nil, err
	}
	cs.SetReadDeadline(time.Time{})

	cs.stopSniffing()
	if isTLS {
		// TODO(jronak): initiate tls handshake to catch tls errors and
		// version metrics.
		return tls.Server(cs, m.tlsConfig), nil
	}

	return cs, nil
}
