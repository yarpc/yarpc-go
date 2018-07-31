// Copyright (c) 2018 Uber Technologies, Inc.
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

package grpc

import (
	"crypto/tls"

	"go.uber.org/yarpc/api/peer"
)

// WithTLS creates a TLS gRPC transport wrapper.
// Any outbound and peer list constructed with this TLS decorator will feed the
// TLS configuration to each peer used by that peer list and client.
// Use a separate TLS transport for each separate TLS configuration.
//
//   trans := grpc.NewTransport()
//   tlsTrans := trans.WithTLS(&tls.Config{...})
//   list := roundrobin.New(tlsTrans)
//   list.Update(peer.ListUpdates{
//       Additions: []peer.Identifier{
//           hostport.Identify("127.0.0.1:8080"),
//       },
//   })
//   outbound := trans.NewOutbound(list)
//   client := theirserviceencoding.Client(outbound)
//
// For a nil tlsConfig, returns the transport unmodified.
func (t *Transport) WithTLS(tlsConfig *tls.Config) peer.Transport {
	if tlsConfig == nil {
		return t
	}
	return &tlsTransport{
		trans:     t,
		tlsConfig: tlsConfig,
	}
}

// tlsTransport is a decorator for a gRPC transport that threads TLS
// configuration to the transport for every retained and released peer.
type tlsTransport struct {
	trans     *Transport
	tlsConfig *tls.Config
}

var _ peer.Transport = (*tlsTransport)(nil)

// RetainPeer retains the identified peer with TLS.
func (t tlsTransport) RetainPeer(id peer.Identifier, ps peer.Subscriber) (peer.Peer, error) {
	return t.trans.retainPeer(id, t.tlsConfig, ps)
}

// ReleasePeer releases a TLS peer.
func (t tlsTransport) ReleasePeer(id peer.Identifier, ps peer.Subscriber) error {
	return t.trans.ReleasePeer(id, ps)
}
