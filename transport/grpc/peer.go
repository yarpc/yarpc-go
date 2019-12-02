// Copyright (c) 2019 Uber Technologies, Inc.
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
	"go.uber.org/yarpc/peer/abstractpeer"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
)

type grpcPeer struct {
	*abstractpeer.Peer

	t          *Transport
	ctx        context.Context
	cancel     context.CancelFunc
	clientConn *grpc.ClientConn
	changedC   chan struct{}
	stoppedC   chan struct{}
}

func (t *Transport) newPeer(address string, options *dialOptions) (*grpcPeer, error) {
	dialOptions := append([]grpc.DialOption{
		grpc.WithUserAgent(UserAgent),
		grpc.WithDefaultCallOptions(
			grpc.CallCustomCodec(customCodec{}),
			grpc.MaxCallRecvMsgSize(t.options.clientMaxRecvMsgSize),
			grpc.MaxCallSendMsgSize(t.options.clientMaxSendMsgSize),
		),
	}, options.grpcOptions()...)

	clientConn, err := grpc.Dial(address, dialOptions...)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())

	grpcPeer := &grpcPeer{
		Peer:       abstractpeer.NewPeer(abstractpeer.PeerIdentifier(address), t),
		t:          t,
		ctx:        ctx,
		cancel:     cancel,
		clientConn: clientConn,
		changedC:   make(chan struct{}),
		stoppedC:   make(chan struct{}),
	}

	go grpcPeer.monitorConnectionStatus()
	go grpcPeer.monitorPendingRequestCount()

	return grpcPeer, nil
}

func (p *grpcPeer) monitorConnectionStatus() {
	p.setConnectionStatus(peer.Unavailable)
	var grpcStatus connectivity.State
	for {
		grpcStatus = p.clientConn.GetState()
		yarpcStatus := grpcStatusToYARPCStatus(grpcStatus)
		p.setConnectionStatus(yarpcStatus)

		if !p.clientConn.WaitForStateChange(p.ctx, grpcStatus) {
			break
		}
	}
	p.setConnectionStatus(peer.Unavailable)

	// Close always returns an error.
	_ = p.clientConn.Close()
	close(p.stoppedC)
}

func (p *grpcPeer) setConnectionStatus(status peer.ConnectionStatus) {
	p.Peer.SetStatus(status)
	p.Peer.NotifyStatusChanged()
}

func (p *grpcPeer) monitorPendingRequestCount() {
	for {
		select {
		case <-p.ctx.Done():
			return
		case <-p.changedC:
			p.Peer.NotifyStatusChanged()
		}
	}
}

func (p *grpcPeer) notifyPendingRequestCountChanged() {
	// kick the pending request count change channel.
	// monitorPendingRequestCount broadcasts changes to subscribers so
	// StartRequest() and EndRequest() don't reply to peer lists on the stack,
	// possibly causing deadlock.
	select {
	case p.changedC <- struct{}{}:
	default:
	}
}

func (p *grpcPeer) StartRequest() {
	p.Peer.StartRequest()
	p.notifyPendingRequestCountChanged()
}

func (p *grpcPeer) EndRequest() {
	p.Peer.EndRequest()
	p.notifyPendingRequestCountChanged()
}

func (p *grpcPeer) stop() {
	p.cancel()
}

func (p *grpcPeer) wait() {
	<-p.stoppedC
}

func grpcStatusToYARPCStatus(grpcStatus connectivity.State) peer.ConnectionStatus {
	switch grpcStatus {
	case connectivity.Ready:
		return peer.Available
	case connectivity.Connecting:
		return peer.Connecting
	default:
		return peer.Unavailable
	}
}
