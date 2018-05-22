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
	"context"
	"sync"

	"go.uber.org/yarpc/api/peer"
	"go.uber.org/yarpc/peer/hostport"
	"go.uber.org/yarpc/yarpcerrors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials"
)

type grpcPeer struct {
	*hostport.Peer
	t          *Transport
	clientConn *grpc.ClientConn
	stoppingC  chan struct{}
	stoppedC   chan error
	lock       sync.Mutex
	stopping   bool
	stopped    bool
	stoppedErr error
}

func newPeer(address string, t *Transport) (*grpcPeer, error) {
	dialOptions := []grpc.DialOption{
		grpc.WithUserAgent(UserAgent),
		grpc.WithDefaultCallOptions(
			grpc.CallCustomCodec(customCodec{}),
			grpc.MaxCallRecvMsgSize(t.options.clientMaxRecvMsgSize),
			grpc.MaxCallSendMsgSize(t.options.clientMaxSendMsgSize),
		),
	}
	if t.options.clientTLSConfig != nil {
		dialOptions = append(dialOptions, grpc.WithTransportCredentials(credentials.NewTLS(t.options.clientTLSConfig)))
	} else if t.options.clientTLS {
		dialOptions = append(dialOptions, grpc.WithTransportCredentials(credentials.NewClientTLSFromCert(nil, "")))
	} else {
		dialOptions = append(dialOptions, grpc.WithInsecure())
	}
	clientConn, err := grpc.Dial(address, dialOptions...)
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
	if !p.monitorStart() {
		p.monitorStop(nil)
		return
	}

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
				p.monitorStop(err)
				return
			}
			p.Peer.SetStatus(peerConnectionStatus)
		}

		var ctx context.Context
		var cancel context.CancelFunc
		if peerConnectionStatus == peer.Available {
			attempts = 0
			ctx = context.Background()
		} else {
			attempts++
			ctx, cancel = context.WithTimeout(context.Background(), backoff.Duration(attempts))
		}

		newConnectivityState, loop := p.monitorLoopWait(ctx, cancel, connectivityState)
		if !loop {
			p.monitorStop(nil)
			return
		}
		changed = connectivityState != newConnectivityState
		connectivityState = newConnectivityState
	}
}

// return true if the transport is started
// return false is monitor was stopped in the meantime
// this should only be called by monitor()
func (p *grpcPeer) monitorStart() bool {
	select {
	// wait for start so we can be certain that we have a channel
	case <-p.t.once.Started():
		return true
	case <-p.stoppingC:
		return false
	}
}

// this should only be called by monitor()
func (p *grpcPeer) monitorStop(err error) {
	p.Peer.SetStatus(peer.Unavailable)
	// Close always returns an error
	_ = p.clientConn.Close()
	p.stoppedC <- err
	close(p.stoppedC)
}

// this should only be called by monitor()
// this does not correlate to wait() at all
//
// return true to continue looping
func (p *grpcPeer) monitorLoopWait(ctx context.Context, cancel context.CancelFunc, connectivityState connectivity.State) (connectivity.State, bool) {
	changedC := make(chan bool, 1)
	go func() { changedC <- p.clientConn.WaitForStateChange(ctx, connectivityState) }()

	loop := false
	select {
	case changed := <-changedC:
		if cancel != nil {
			cancel()
		}
		if changed {
			connectivityState = p.clientConn.GetState()
		}
		loop = true
	case <-p.stoppingC:
	case <-p.t.once.Stopping():
		if cancel != nil {
			cancel()
		}
	}
	return connectivityState, loop
}

func (p *grpcPeer) stop() {
	p.lock.Lock()
	defer p.lock.Unlock()
	if !p.stopping {
		// this is selected on in monitor()
		p.stoppingC <- struct{}{}
		close(p.stoppingC)
		p.stopping = true
	}
}

func (p *grpcPeer) wait() error {
	p.lock.Lock()
	defer p.lock.Unlock()
	if p.stopped {
		return p.stoppedErr
	}
	p.stoppedErr = <-p.stoppedC
	p.stopped = true
	return p.stoppedErr
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
		return 0, yarpcerrors.Newf(yarpcerrors.CodeInternal, "unknown connectivity.State: %v", connectivityState)
	}
}
