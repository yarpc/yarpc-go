package grpc

import (
	"crypto/tls"

	"go.uber.org/yarpc/api/peer"
)

type tlsIdentifier struct {
	id        peer.Identifier
	tlsConfig *tls.Config
}

func (id *tlsIdentifier) Identifier() string {
	return id.id.Identifier()
}

type tlsTransport struct {
	trans     *Transport
	tlsConfig *tls.Config
}

func (t tlsTransport) RetainPeer(id peer.Identifier, sub peer.Subscriber) (peer.Peer, error) {
	return t.trans.RetainPeer(&tlsIdentifier{
		id:        id,
		tlsConfig: t.tlsConfig,
	}, sub)
}

func (t tlsTransport) ReleasePeer(id peer.Identifier, sub peer.Subscriber) error {
	return t.trans.ReleasePeer(&tlsIdentifier{
		id:        id,
		tlsConfig: t.tlsConfig,
	}, sub)
}
