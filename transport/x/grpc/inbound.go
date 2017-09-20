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
	"net"
	"sync"

	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/pkg/lifecycle"
	"go.uber.org/yarpc/yarpcerrors"
	"google.golang.org/grpc"
)

var (
	errRouterNotSet = yarpcerrors.Newf(yarpcerrors.CodeInternal, "router not set")

	_ transport.Inbound = (*Inbound)(nil)
)

// Inbound is a grpc transport.Inbound.
type Inbound struct {
	once     *lifecycle.Once
	lock     sync.Mutex
	t        *Transport
	listener net.Listener
	options  *inboundOptions
	router   transport.Router
	server   *grpc.Server
}

// newInbound returns a new Inbound for the given listener.
func newInbound(t *Transport, listener net.Listener, options ...InboundOption) *Inbound {
	return &Inbound{
		once:     lifecycle.NewOnce(),
		t:        t,
		listener: listener,
		options:  newInboundOptions(options),
	}
}

// Start implements transport.Lifecycle#Start.
func (i *Inbound) Start() error {
	return i.once.Start(i.start)
}

// Stop implements transport.Lifecycle#Stop.
func (i *Inbound) Stop() error {
	return i.once.Stop(i.stop)
}

// IsRunning implements transport.Lifecycle#IsRunning.
func (i *Inbound) IsRunning() bool {
	return i.once.IsRunning()
}

// SetRouter implements transport.Inbound#SetRouter.
func (i *Inbound) SetRouter(router transport.Router) {
	i.lock.Lock()
	defer i.lock.Unlock()
	i.router = router
}

// Transports implements transport.Inbound#Transports.
func (i *Inbound) Transports() []transport.Transport {
	return []transport.Transport{i.t}
}

func (i *Inbound) start() error {
	i.lock.Lock()
	defer i.lock.Unlock()
	if i.router == nil {
		return errRouterNotSet
	}

	handler := newHandler(i)

	server := grpc.NewServer(
		grpc.CustomCodec(customCodec{}),
		grpc.UnknownServiceHandler(handler.handle),
		grpc.MaxRecvMsgSize(i.t.options.serverMaxRecvMsgSize),
		grpc.MaxSendMsgSize(i.t.options.serverMaxSendMsgSize),
	)

	go func() {
		// TODO there should be some mechanism to block here
		// there is a race because the listener gets set in the grpc
		// Server implementation and we should be able to block
		// until Serve initialization is done
		//
		// It would be even better if we could do this outside the
		// lock in i
		//
		// TODO Server always returns a non-nil error but should
		// we do something with some or all errors?
		_ = server.Serve(i.listener)
	}()
	i.server = server
	return nil
}

func (i *Inbound) stop() error {
	i.lock.Lock()
	defer i.lock.Unlock()
	if i.server != nil {
		i.server.GracefulStop()
	}
	return nil
}

type noopGrpcStruct struct{}
