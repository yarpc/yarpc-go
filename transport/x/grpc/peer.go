package grpc

import (
	"context"

	"go.uber.org/yarpc/api/peer"
	"go.uber.org/yarpc/peer/hostport"
	"go.uber.org/yarpc/yarpcerrors"
	"google.golang.org/grpc"
)

type grpcPeer struct {
	*hostport.Peer
	t          *Transport
	clientConn *grpc.ClientConn
	stoppingC  chan struct{}
	stoppedC   chan error
}

func newPeer(peerIdentifier hostport.PeerIdentifier, t *Transport) (*grpcPeer, error) {
	clientConn, err := grpc.Dial(
		peerIdentifier.Identifier(),
		grpc.WithInsecure(),
		grpc.WithCodec(customCodec{}),
		grpc.WithUserAgent(UserAgent),
	)
	if err != nil {
		return nil, err
	}
	grpcPeer := &grpcPeer{
		Peer:       hostport.NewPeer(peerIdentifier, t),
		t:          t,
		clientConn: clientConn,
		stoppingC:  make(chan struct{}, 1),
		stoppedC:   make(chan error, 1),
	}
	go grpcPeer.monitor()
	return grpcPeer, nil
}

func (p *grpcPeer) monitor() {
	// wait for start so we can be certain that we have a channel
	<-p.t.once.Started()

	attempts := 0
	backoff := p.t.options.backoffStrategy.Backoff()

	connectivityState := p.clientConn.GetState()
	changed := true
	for {
		if changed {
			peerStatus, err := connectivityStateToPeerStatus(connectivityState)
			if err != nil {
				p.stop(err)
				return
			}
			p.Peer.SetStatus(peerStatus)
		}

		ctx := context.Background()
		var cancel context.CancelFunc
		if peerStatus == peer.Available {
			attempts = 0
		} else {
			attempts++
			ctx, cancel = context.WithTimeout(ctx, backoff.Duration(attempts))
		}

		changedC := make(chan bool, 1)
		go func() { changedC <- p.clientConn.WaitForStateChange(ctx, connectivityState) }()

		select {
		case changed := <-changedC:
			if cancel != nil {
				cancel()
			}
			continue
		case <-p.stoppingC:
		case <-p.t.once.Stopping():
			p.stop(nil)
			return
		}
	}
}

func (p *grpcPeer) signalStop() {
	p.stoppingC <- struct{}{}
}

func (p *grpcPeer) stop(err error) {
	p.Peer.SetStatus(peer.Unavailable)
	_ = p.clientConn.Close()
	p.stoppedC <- err
}

func (p *grpcPeer) wait() error {
	return <-p.stoppedC
}

func connectivityStateToPeerStatus(connectivityState grpc.ConnectivityState) (peer.Status, error) {
	switch connectivityState {
	case grpc.Idle, grpc.TransientFailure, grpc.Shutdown:
		return peer.Unavailable, nil
	case grpc.Connecting:
		return peer.Connecting, nil
	case grpc.Available:
		return peer.Available, nil
	default:
		return 0, yarpcerrors.InternalErrorf("unknown grpc.ConnectivityState: %v", connectivityState)
	}
}
