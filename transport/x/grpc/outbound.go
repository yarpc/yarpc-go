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
	"sync"

	"google.golang.org/grpc"

	"go.uber.org/yarpc/api/transport"
	internalsync "go.uber.org/yarpc/internal/sync"
)

var _ transport.UnaryOutbound = (*Outbound)(nil)

// Outbound is a transport.UnaryOutbound.
type Outbound struct {
	once       internalsync.LifecycleOnce
	lock       sync.Mutex
	address    string
	clientConn *grpc.ClientConn
}

// NewSingleOutbound returns a new Outbound for the given adrress.
func NewSingleOutbound(address string) *Outbound {
	return &Outbound{internalsync.Once(), sync.Mutex{}, address, nil}
}

// Start implements transport.Lifecycle#Start.
func (o *Outbound) Start() error {
	return o.once.Start(o.start)
}

// Stop implements transport.Lifecycle#Stop.
func (o *Outbound) Stop() error {
	return o.once.Stop(o.stop)
}

// IsRunning implements transport.Lifecycle#IsRunning.
func (o *Outbound) IsRunning() bool {
	return o.once.IsRunning()
}

// Transports implements transport.Inbound#Transports.
func (o *Outbound) Transports() []transport.Transport {
	return []transport.Transport{}
}

// Call implements transport.UnaryOutbound#Call.
func (o *Outbound) Call(ctx context.Context, request *transport.Request) (*transport.Response, error) {
	return nil, nil
}

func (o *Outbound) start() error {
	// TODO: redial
	clientConn, err := grpc.Dial(
		o.address,
		grpc.WithInsecure(),
		// TODO: want to support default codec
		grpc.WithCodec(noopCodec{}),
	)
	if err != nil {
		return err
	}
	o.lock.Lock()
	defer o.lock.Unlock()
	o.clientConn = clientConn
	return nil
}

func (o *Outbound) stop() error {
	o.lock.Lock()
	defer o.lock.Unlock()
	if o.clientConn != nil {
		return o.clientConn.Close()
	}
	return nil
}
