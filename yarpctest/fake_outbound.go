// Copyright (c) 2024 Uber Technologies, Inc.
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

package yarpctest

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"

	"go.uber.org/yarpc/api/peer"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/api/transport/transporttest"
	"go.uber.org/yarpc/pkg/lifecycle"
)

var (
	_ transport.Namer          = (*FakeOutbound)(nil)
	_ transport.UnaryOutbound  = (*FakeOutbound)(nil)
	_ transport.OnewayOutbound = (*FakeOutbound)(nil)
	_ transport.StreamOutbound = (*FakeOutbound)(nil)
)

// FakeOutboundOption is an option for FakeTransport.NewOutbound.
type FakeOutboundOption func(*FakeOutbound)

// NopOutboundOption returns an option to set the "nopOption" for a
// FakeTransport.NewOutbound.
// The nopOption has no effect exists only to verify that the option was
// passed, via `FakeOutbound.NopOption()`.
func NopOutboundOption(nopOption string) FakeOutboundOption {
	return func(o *FakeOutbound) {
		o.nopOption = nopOption
	}
}

// OutboundName sets the name of the "fake" outbound.
func OutboundName(name string) FakeOutboundOption {
	return func(o *FakeOutbound) {
		o.name = name
	}
}

// OutboundCallable is a function that will be called for for an outbound's
// `Call` method.
type OutboundCallable func(ctx context.Context, req *transport.Request) (*transport.Response, error)

// OutboundOnewayCallable is a function that will be called for for an outbound's
// `Call` method.
type OutboundOnewayCallable func(context.Context, *transport.Request) (transport.Ack, error)

// OutboundStreamCallable is a function that will be called for for an outbound's
// `Call` method.
type OutboundStreamCallable func(context.Context, *transport.StreamRequest) (*transport.ClientStream, error)

// OutboundRouter returns an option to set the router for outbound requests.
// This connects the outbound to the inbound side of a handler for testing
// purposes.
func OutboundRouter(router transport.Router) FakeOutboundOption {
	return func(o *FakeOutbound) {
		o.router = router
	}
}

// OutboundCallOverride returns an option to set the "callOverride" for a
// FakeTransport.NewOutbound.
// This can be used to set the functionality for the FakeOutbound's `Call`
// function.
func OutboundCallOverride(callable OutboundCallable) FakeOutboundOption {
	return func(o *FakeOutbound) {
		o.callOverride = callable
	}
}

// OutboundCallOnewayOverride returns an option to set the "callOverride" for a
// FakeTransport.NewOutbound.
//
// This can be used to set the functionality for the FakeOutbound's `CallOneway`
// function.
func OutboundCallOnewayOverride(callable OutboundOnewayCallable) FakeOutboundOption {
	return func(o *FakeOutbound) {
		o.callOnewayOverride = callable
	}
}

// OutboundCallStreamOverride returns an option to set the "callOverride" for a
// FakeTransport.NewOutbound.
//
// This can be used to set the functionality for the FakeOutbound's `CallStream`
// function.
func OutboundCallStreamOverride(callable OutboundStreamCallable) FakeOutboundOption {
	return func(o *FakeOutbound) {
		o.callStreamOverride = callable
	}
}

// NewOutbound returns a FakeOutbound with a given peer chooser and options.
func (t *FakeTransport) NewOutbound(c peer.Chooser, opts ...FakeOutboundOption) *FakeOutbound {
	o := &FakeOutbound{
		name:      "fake",
		once:      lifecycle.NewOnce(),
		transport: t,
		chooser:   c,
	}
	for _, opt := range opts {
		opt(o)
	}
	return o
}

// FakeOutbound is a unary outbound for the FakeTransport. It is fake.
type FakeOutbound struct {
	name      string
	once      *lifecycle.Once
	transport *FakeTransport
	chooser   peer.Chooser
	nopOption string
	router    transport.Router

	callOverride       OutboundCallable
	callOnewayOverride OutboundOnewayCallable
	callStreamOverride OutboundStreamCallable
}

// TransportName is "fake".
func (o *FakeOutbound) TransportName() string {
	return "fake"
}

// Chooser returns theis FakeOutbound's peer chooser.
func (o *FakeOutbound) Chooser() peer.Chooser {
	return o.chooser
}

// NopOption returns this FakeOutbound's nopOption. It is fake.
func (o *FakeOutbound) NopOption() string {
	return o.nopOption
}

// Start starts the fake outbound and its chooser.
func (o *FakeOutbound) Start() error {
	return o.once.Start(o.chooser.Start)
}

// Stop stops the fake outbound and its chooser.
func (o *FakeOutbound) Stop() error {
	return o.once.Stop(o.chooser.Stop)
}

// IsRunning returns whether the fake outbound is running.
func (o *FakeOutbound) IsRunning() bool {
	return o.once.IsRunning()
}

// Transports returns the FakeTransport that owns this outbound.
func (o *FakeOutbound) Transports() []transport.Transport {
	return []transport.Transport{o.transport}
}

// Call mimicks sending a oneway RPC.
//
// By default, this returns a error.
// The OutboundCallOverride option supplies an alternate implementation.
// Alternately, the OutboundRouter option may allow this function to route a
// unary request to a unary handler.
func (o *FakeOutbound) Call(ctx context.Context, req *transport.Request) (*transport.Response, error) {
	if o.callOverride != nil {
		return o.callOverride(ctx, req)
	}

	if o.router == nil {
		return nil, errors.New(`no outbound callable specified on the fake outbound`)
	}

	handler, err := o.router.Choose(ctx, req)
	if err != nil {
		return nil, err
	}

	unaryHandler := handler.Unary()
	if unaryHandler == nil {
		return nil, fmt.Errorf(`procedure %q for encoding %q does not handle unary requests`, req.Procedure, req.Encoding)
	}

	resWriter := &transporttest.FakeResponseWriter{}
	if err := unaryHandler.Handle(ctx, req, resWriter); err != nil {
		return nil, err
	}
	return &transport.Response{
		ApplicationError: resWriter.IsApplicationError,
		Headers:          resWriter.Headers,
		Body:             ioutil.NopCloser(&resWriter.Body),
	}, nil
}

// CallOneway mimicks sending a oneway RPC.
//
// By default, this returns an error.
// The OutboundCallOnewayOverride supplies an alternate implementation.
// Atlernately, the OutboundRouter options may route the request through to a
// oneway handler.
func (o *FakeOutbound) CallOneway(ctx context.Context, req *transport.Request) (transport.Ack, error) {
	if o.callOnewayOverride != nil {
		return o.callOnewayOverride(ctx, req)
	}

	if o.router == nil {
		return nil, errors.New(`fake outbound does not support call oneway`)
	}

	handler, err := o.router.Choose(ctx, req)
	if err != nil {
		return nil, err
	}

	onewayHandler := handler.Oneway()
	if onewayHandler == nil {
		return nil, fmt.Errorf(`procedure %q for encoding %q does not handle oneway requests`, req.Procedure, req.Encoding)
	}

	return nil, onewayHandler.HandleOneway(ctx, req)
}

// CallStream mimicks sending a streaming RPC.
//
// By default, this returns an error.
// The OutboundCallStreamOverride option provides a hook to change the behavior.
// Alternately, the OutboundRouter option may route the request through to a
// streaming handler.
func (o *FakeOutbound) CallStream(ctx context.Context, streamReq *transport.StreamRequest) (*transport.ClientStream, error) {
	if o.callStreamOverride != nil {
		return o.callStreamOverride(ctx, streamReq)
	}

	if o.router == nil {
		return nil, errors.New(`fake outbound does not support call stream`)
	}

	req := streamReq.Meta.ToRequest()
	handler, err := o.router.Choose(ctx, req)
	if err != nil {
		return nil, err
	}

	streamHandler := handler.Stream()
	if streamHandler == nil {
		return nil, fmt.Errorf(`procedure %q for encoding %q does not handle streaming requests`, req.Procedure, req.Encoding)
	}

	clientStream, serverStream, finish, err := transporttest.MessagePipe(ctx, streamReq)
	if err != nil {
		return nil, err
	}

	go func() {
		finish(streamHandler.HandleStream(serverStream))
	}()
	return clientStream, nil
}
