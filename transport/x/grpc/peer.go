package grpc

import (
	"go.uber.org/yarpc/api/peer"
	"go.uber.org/yarpc/peer/hostport"
	"google.golang.org/grpc"
)

type grpcPeer struct {
	*hostport.Peer
	clientConn *grpc.ClientConn
}

func newPeer(address string, transport *Transport) (*grpcPeer, error) {
	clientConn, err := grpc.Dial(
		address,
		grpc.WithInsecure(),
		grpc.WithCodec(customCodec{}),
		grpc.WithUserAgent(UserAgent),
	)
	if err != nil {
		return nil, err
	}
	return &grpcPeer{
		Peer:       hostport.NewPeer(hostport.PeerIdentifier(address), transport),
		clientConn: clientConn,
	}, nil
}

// TODO close the clientConn when the transport stops or the peer is released.

// TODO NotifyStatusChange whenenver connection status changes or pending
// request count changes.

func (p *grpcPeer) Status() peer.Status {
	return peer.Status{
		PendingRequestCount: 0,
		ConnectionStatus:    peer.Available,
	}
}

func (p *grpcPeer) StartRequest() {
	// TODO pending request count
}

func (p *grpcPeer) EndRequest() {
}
