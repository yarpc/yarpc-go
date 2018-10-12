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
	"math"
	"sync"

	"github.com/opentracing/opentracing-go"
	"go.uber.org/multierr"
	"go.uber.org/yarpc/v2"
	"go.uber.org/yarpc/v2/yarpcbackoff"
	"go.uber.org/yarpc/v2/yarpcpeer"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const (
	// defensive programming: these are copied from grpc-go but we set them
	// explicitly here in case these change in grpc-go so that YARPC stays
	// consistent.
	defaultClientMaxRecvMsgSize = 1024 * 1024 * 4
	defaultClientMaxSendMsgSize = math.MaxInt32
)

var _ yarpc.Dialer = (*Dialer)(nil)

// Dialer keeps track of all gRPC peers.
type Dialer struct {
	// ClientMaxRecvMsgSize is the maximum message size the client can receive.
	//
	// The default is 4MB.
	ClientMaxRecvMsgSize int
	// ClientMaxSendMsgSize is the maximum message size the client can send.
	//
	// The default is math.MaxInt32.
	ClientMaxSendMsgSize int

	// Credentials specifies connection level security credentials (e.g.,
	// TLS/SSL) for outbound connections.
	Credentials credentials.TransportCredentials

	// BackoffStrategy specifies the backoff strategy for delays between
	// connection attempts for each peer.
	//
	// The default is exponential backoff starting with 10ms fully jittered,
	// doubling each attempt, with a maximum interval of 30s.
	Backoff yarpc.BackoffStrategy

	// Tracer configures a logger for the dialer.
	Logger *zap.Logger

	// Tracer configures a tracer for the dialer.
	Tracer opentracing.Tracer

	internal *dialerInternals
}

type dialerInternals struct {
	lock          sync.Mutex
	addressToPeer map[string]*grpcPeer

	dialOptions []grpc.DialOption
	backoff     yarpc.BackoffStrategy

	logger *zap.Logger
	tracer opentracing.Tracer
}

// Start starts the gRPC dialer.
func (d *Dialer) Start(context.Context) error {
	d.internal = &dialerInternals{
		addressToPeer: make(map[string]*grpcPeer),
		backoff:       yarpcbackoff.DefaultExponential,
		tracer:        opentracing.GlobalTracer(),
		logger:        zap.NewNop(),
	}

	if d.Backoff != nil {
		d.internal.backoff = d.Backoff
	}
	if d.Logger != nil {
		d.internal.logger = d.Logger
	}
	if d.Tracer != nil {
		d.internal.tracer = d.Tracer
	}

	d.setDialOptions()
	return nil
}

func (d *Dialer) setDialOptions() {
	credentialDialOption := grpc.WithInsecure()
	if d.Credentials != nil {
		credentialDialOption = grpc.WithTransportCredentials(d.Credentials)
	}

	defaultCallOptions := []grpc.CallOption{grpc.CallCustomCodec(customCodec{})}

	clientMaxRecvMsgSize := defaultClientMaxRecvMsgSize
	if d.ClientMaxRecvMsgSize != 0 {
		clientMaxRecvMsgSize = d.ClientMaxRecvMsgSize
	}
	defaultCallOptions = append(defaultCallOptions, grpc.MaxCallRecvMsgSize(clientMaxRecvMsgSize))

	clientMaxSendMsgSize := defaultClientMaxSendMsgSize
	if d.ClientMaxSendMsgSize != 0 {
		clientMaxSendMsgSize = d.ClientMaxSendMsgSize
	}
	defaultCallOptions = append(defaultCallOptions, grpc.MaxCallSendMsgSize(clientMaxSendMsgSize))

	d.internal.dialOptions = []grpc.DialOption{
		credentialDialOption,
		grpc.WithUserAgent(UserAgent),
		grpc.WithDefaultCallOptions(defaultCallOptions...),
	}
}

// Stop stops the gRPC dialer.
func (d *Dialer) Stop(ctx context.Context) error {
	return d.internal.stop(ctx)
}

func (d *dialerInternals) stop(context.Context) error {
	d.lock.Lock()
	defer d.lock.Unlock()
	for _, grpcPeer := range d.addressToPeer {
		grpcPeer.stop()
	}
	var err error
	for _, grpcPeer := range d.addressToPeer {
		err = multierr.Append(err, grpcPeer.wait())
	}
	return err
}

// RetainPeer retains the identified peer, passing dial options.
func (d *Dialer) RetainPeer(id yarpc.Identifier, sub yarpc.Subscriber) (yarpc.Peer, error) {
	if d.internal == nil {
		return nil, fmt.Errorf("yarpcgrpc.Dialer.RetainPeer must be called after Start")
	}
	return d.internal.retainPeer(id, sub)
}

func (d *dialerInternals) retainPeer(id yarpc.Identifier, sub yarpc.Subscriber) (yarpc.Peer, error) {
	d.lock.Lock()
	defer d.lock.Unlock()
	address := id.Identifier()
	p, ok := d.addressToPeer[address]
	if !ok {
		var err error
		p, err = d.newPeer(id)
		if err != nil {
			return nil, err
		}
		d.addressToPeer[address] = p
	}
	p.Subscribe(sub)
	return p, nil
}

// ReleasePeer releases the identified peer.
func (d *Dialer) ReleasePeer(id yarpc.Identifier, sub yarpc.Subscriber) error {
	if d.internal == nil {
		return fmt.Errorf("yarpcgrpc.Dialer.ReleasePeer must be called after Start")
	}
	return d.internal.releasePeer(id, sub)
}

func (d *dialerInternals) releasePeer(id yarpc.Identifier, sub yarpc.Subscriber) error {
	d.lock.Lock()
	defer d.lock.Unlock()
	address := id.Identifier()
	p, ok := d.addressToPeer[address]
	if !ok {
		return yarpcpeer.ErrDialerHasNoReferenceToPeer{
			DialerName:     "grpc.Dialer",
			PeerIdentifier: address,
		}
	}
	if err := p.Unsubscribe(sub); err != nil {
		return err
	}
	if p.NumSubscribers() == 0 {
		delete(d.addressToPeer, address)
		p.stop()
		return p.wait()
	}
	return nil
}
