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

package dialer

import (
	"context"
	"crypto/tls"
	"net"
	"time"
)

const (
	// Yarpc uses default dial timeout of 30s for HTTP. This value seems large
	// enough for all protocols.
	// Ref: https://github.com/yarpc/yarpc-go/blob/ab5cb1600445ed2c2aaf1b025257b84a81c01a90/transport/http/transport.go#L280
	defaultDialTimeout = time.Second * 30
	// HTTP transport uses default handshake timeout of 10s.
	// Ref: https://github.com/golang/go/blob/f78efc0178d51c02beff8a8203910dc0a9c6e953/src/net/http/transport.go#L52
	defaultHandshakeTimeout = time.Second * 10
)

// Params holds parameters needed for creating new TLSDialer.
type Params struct {
	Config *tls.Config
	Dialer func(ctx context.Context, network, address string) (net.Conn, error)
}

// TLSDialer implements context dialer.
type TLSDialer struct {
	config *tls.Config
	dialer func(ctx context.Context, network, address string) (net.Conn, error)
}

// NewTLSDialer returns dialer which creates TLS client connection based on
// the given TLS configuration.
func NewTLSDialer(params Params) *TLSDialer {
	dialer := params.Dialer
	if dialer == nil {
		dialer = (&net.Dialer{
			Timeout: defaultDialTimeout,
		}).DialContext
	}
	return &TLSDialer{
		config: params.Config,
		dialer: dialer,
	}
}

// DialContext returns a TLS client connection after finishing the handshake.
func (t *TLSDialer) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	conn, err := t.dialer(ctx, network, addr)
	if err != nil {
		return nil, err
	}

	tlsConn := tls.Client(conn, t.config)
	ctx, cancel := context.WithTimeout(ctx, defaultHandshakeTimeout)
	defer cancel()
	if err := tlsConn.HandshakeContext(ctx); err != nil {
		// TODO: emit metric to track handshake error.
		return nil, err
	}

	// TODO: emit metric to track successful handshake and TLS version.
	return tlsConn, nil
}
