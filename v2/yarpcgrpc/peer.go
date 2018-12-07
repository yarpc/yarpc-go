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

package yarpcgrpc

import (
	"context"
	"fmt"
	"sync"

	"go.uber.org/yarpc/v2"
	"go.uber.org/yarpc/v2/yarpcerror"
	"go.uber.org/yarpc/v2/yarpcpeer"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
)

type grpcPeer struct {
	*yarpcpeer.AbstractPeer

	dialer     *dialerInternals
	clientConn *grpc.ClientConn
	stoppingC  chan struct{}
	stoppedC   chan error
	lock       sync.Mutex
	stopping   bool
	stopped    bool
	stoppedErr error
}

func (d *dialerInternals) newPeer(id yarpc.Identifier) (*grpcPeer, error) {
	clientConn, err := grpc.Dial(id.Identifier(), d.dialOptions...)
	if err != nil {
		return nil, err
	}
	grpcPeer := &grpcPeer{
		AbstractPeer: yarpcpeer.NewAbstractPeer(id),
		dialer:       d,
		clientConn:   clientConn,
		stoppingC:    make(chan struct{}, 1),
		stoppedC:     make(chan error, 1),
	}
	go grpcPeer.monitor()
	return grpcPeer, nil
}

func (p *grpcPeer) monitor() {
	var attempts uint
	backoff := p.dialer.backoff.Backoff()

	connectivityState := p.clientConn.GetState()
	changed := true
	for {
		var peerConnectionStatus yarpc.ConnectionStatus
		var err error
		// will be called the first time since changed is initialized to true
		if changed {
			peerConnectionStatus, err = connectivityStateToPeerConnectionStatus(connectivityState)
			if err != nil {
				p.monitorStop(err)
				return
			}
			p.AbstractPeer.SetStatus(peerConnectionStatus)
		}

		var ctx context.Context
		var cancel context.CancelFunc
		if peerConnectionStatus == yarpc.Available {
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

// this should only be called by monitor()
func (p *grpcPeer) monitorStop(err error) {
	p.AbstractPeer.SetStatus(yarpc.Unavailable)
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

func connectivityStateToPeerConnectionStatus(connectivityState connectivity.State) (yarpc.ConnectionStatus, error) {
	switch connectivityState {
	case connectivity.Idle, connectivity.TransientFailure, connectivity.Shutdown, connectivity.Connecting:
		return yarpc.Unavailable, nil
	case connectivity.Ready:
		return yarpc.Available, nil
	default:
		return 0, yarpcerror.New(yarpcerror.CodeInternal, fmt.Sprintf("unknown connectivity.State: %v", connectivityState))
	}
}
