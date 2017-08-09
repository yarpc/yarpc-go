package grpc

import (
	"go.uber.org/yarpc/peer/hostport"
	"google.golang.org/grpc"
)

type grpcPeer struct {
	*hostport.Peer
	clientConn *grpc.ClientConn
	signalC    chan struct{}
	stoppedC   chan struct{}
}

func newPeer(peerIdentifier hostport.PeerIdentifier, transport *Transport) (*grpcPeer, error) {
	clientConn, err := grpc.Dial(
		peerIdentifier.Identifier(),
		grpc.WithInsecure(),
		grpc.WithCodec(customCodec{}),
		grpc.WithUserAgent(UserAgent),
	)
	if err != nil {
		return nil, err
	}
	return &grpcPeer{
		Peer:       hostport.NewPeer(peerIdentifier, transport),
		clientConn: clientConn,
		signalC:    make(chan struct{}, 1),
		stoppedC:   make(chan struct{}, 1),
	}, nil
}

// TODO close the clientConn when the transport stops or the peer is released.

// TODO NotifyStatusChange whenenver connection status changes or pending
// request count changes.

func (p *grpcPeer) monitor() {

}

func (p *grpcPeer) stop() {
	p.signalC <- struct{}{}
}

func (p *grpcPeer) wait() {
	<-p.stoppedC
}
