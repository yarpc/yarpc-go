package grpc

import (
	"context"

	"go.uber.org/yarpc/api/peer"
	"go.uber.org/yarpc/peer/hostport"
	"go.uber.org/yarpc/yarpcerrors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
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

	var attempts uint
	backoff := p.t.options.backoffStrategy.Backoff()

	connectivityState := p.clientConn.GetState()
	changed := true
	for {
		var peerConnectionStatus peer.ConnectionStatus
		var err error
		// will be called the first time since changed is initialized to true
		if changed {
			peerConnectionStatus, err = connectivityStateToPeerConnectionStatus(connectivityState)
			if err != nil {
				p.stop(err)
				return
			}
			p.Peer.SetStatus(peerConnectionStatus)
		}

		ctx := context.Background()
		var cancel context.CancelFunc
		if peerConnectionStatus == peer.Available {
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

func connectivityStateToPeerConnectionStatus(connectivityState connectivity.State) (peer.ConnectionStatus, error) {
	switch connectivityState {
	case connectivity.Idle, connectivity.TransientFailure, connectivity.Shutdown:
		return peer.Unavailable, nil
	case connectivity.Connecting:
		return peer.Connecting, nil
	case connectivity.Ready:
		return peer.Available, nil
	default:
		return 0, yarpcerrors.InternalErrorf("unknown connectivity.State: %v", connectivityState)
	}
}
