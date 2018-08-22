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

package yarpctest

import (
	"context"
	"fmt"

	"go.uber.org/yarpc/api/peer"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/pkg/lifecycle"
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

// OutboundCallable is a function that will be called for for an outbound's
// `Call` method.
type OutboundCallable func(ctx context.Context, req *transport.Request) (*transport.Response, error)

// OutboundCallOverride returns an option to set the "callOverride" for a
// FakeTransport.NewOutbound.
// This can be used to set the functionality for the FakeOutbound's `Call`
// function.
func OutboundCallOverride(callable OutboundCallable) FakeOutboundOption {
	return func(o *FakeOutbound) {
		o.callOverride = callable
	}
}

// NewOutbound returns a FakeOutbound with a given peer chooser and options.
func (t *FakeTransport) NewOutbound(c peer.Chooser, opts ...FakeOutboundOption) *FakeOutbound {
	o := &FakeOutbound{
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
	once         *lifecycle.Once
	transport    *FakeTransport
	chooser      peer.Chooser
	nopOption    string
	callOverride OutboundCallable
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

// Call pretends to send a unary RPC, but actually just returns an error.
func (o *FakeOutbound) Call(ctx context.Context, req *transport.Request) (*transport.Response, error) {
	if o.callOverride != nil {
		return o.callOverride(ctx, req)
	}
	return nil, fmt.Errorf(`no outbound callable specified on the fake outbound`)
}

// CallStream pretends to send a Stream RPC, but actually just returns an error.
func (o *FakeOutbound) CallStream(ctx context.Context, req *transport.StreamRequest) (*transport.ClientStream, error) {
	return nil, fmt.Errorf(`fake outbound does not support call stream`)
}
