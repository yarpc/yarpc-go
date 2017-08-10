// Copyright (c) 2017 Uber Technologies, Inc.
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

func newPeer(address string, t *Transport) (*grpcPeer, error) {
	clientConn, err := grpc.Dial(
		address,
		grpc.WithInsecure(),
		grpc.WithCodec(customCodec{}),
		grpc.WithUserAgent(UserAgent),
	)
	if err != nil {
		return nil, err
	}
	grpcPeer := &grpcPeer{
		Peer:       hostport.NewPeer(hostport.PeerIdentifier(address), t),
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
		ctx := context.Background()
		var cancel context.CancelFunc

		var peerConnectionStatus peer.ConnectionStatus
		var err error
		// will be called the first time since changed is initialized to true
		if changed {
			peerConnectionStatus, err = connectivityStateToPeerConnectionStatus(connectivityState)
			if err != nil {
				if cancel != nil {
					cancel()
				}
				p.monitorStop(err)
				return
			}
			p.Peer.SetStatus(peerConnectionStatus)
		}

		if peerConnectionStatus == peer.Available {
			// reset attempts since we are available
			attempts = 0
		} else {
			attempts++
			// this isn't actually a backoff, what we're saying is that
			// we backoff on TIMING OUT before the next state change
			// before returning
			// https://godoc.org/google.golang.org/grpc#ClientConn.WaitForStateChange
			ctx, cancel = context.WithTimeout(ctx, backoff.Duration(attempts))
		}

		changedC := make(chan bool, 1)
		go func() { changedC <- p.clientConn.WaitForStateChange(ctx, connectivityState) }()

		select {
		case changed = <-changedC:
			if cancel != nil {
				cancel()
			}
			if changed {
				connectivityState = p.clientConn.GetState()
			}
			continue
		case <-p.stoppingC:
		case <-p.t.once.Stopping():
			if cancel != nil {
				cancel()
			}
			p.monitorStop(nil)
			return
		}
	}
}

// this should only be called by monitor()
// if you want to stop outside of monitor(), call stop()
func (p *grpcPeer) monitorStop(err error) {
	p.Peer.SetStatus(peer.Unavailable)
	// Close always returns an error
	_ = p.clientConn.Close()
	p.stoppedC <- err
	close(p.stoppedC)
}

// should only be called once
// maybe do this with once?
func (p *grpcPeer) stop() {
	// this is selected on in monitor()
	p.stoppingC <- struct{}{}
	close(p.stoppingC)
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
