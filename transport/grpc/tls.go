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
