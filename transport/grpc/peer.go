// Copyright (c) 2021 Uber Technologies, Inc.
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
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
)

type grpcPeer struct {
	*abstractpeer.Peer

	t          *Transport
	ctx        context.Context
	cancel     context.CancelFunc
	clientConn *grpc.ClientConn
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
		stoppedC:   make(chan struct{}),
	}

	go grpcPeer.monitorConnectionStatus()

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
	p.t.options.logger.Debug(
		"peer status change",
		zap.String("status", status.String()),
		zap.String("peer", p.Peer.Identifier()),
		zap.String("transport", "grpc"),
	)
	p.Peer.SetStatus(status)
	p.Peer.NotifyStatusChanged()
}

// StartRequest and EndRequest are no-ops now.
// They previously aggregated pending request count from all subscibed peer
// lists and distributed change notifications.
// This was fraught with concurrency hazards so we moved pending request count
// tracking into the lists themselves.

func (p *grpcPeer) StartRequest() {}

func (p *grpcPeer) EndRequest() {}

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
