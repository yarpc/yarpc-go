// Copyright (c) 2016 Uber Technologies, Inc.
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

package yarpc

import (
	"fmt"

	"github.com/yarpc/yarpc-go/sync"
	"github.com/yarpc/yarpc-go/transport"

	"golang.org/x/net/context"
)

// RPC TODO
type RPC interface {
	transport.Handler
	transport.Registry

	// Retrieves a new Outbound transport that will make requests to the given
	// service.
	//
	// This panics if the given service is unknown.
	Channel(service string) transport.Outbound
	// TODO do we really want to panic on unknown services?

	// Starts the RPC allowing it to accept and processing new incoming
	// requests.
	//
	// Blocks until the RPC is stopped.
	Serve() error

	// Closes the RPC. No new requests will be accepted.
	//
	// Blocks until the RPC has stopped.
	Close() error
}

// Config specifies the parameters of a new RPC constructed via New.
type Config struct {
	Name      string
	Inbounds  []transport.Inbound
	Outbounds transport.Outbounds

	// TODO FallbackHandler for catch-all endpoints
}

// New builds a new RPC using the specified configuration.
func New(cfg Config) RPC {
	return rpc{
		Name:      cfg.Name,
		Registry:  make(transport.MapRegistry),
		Inbounds:  cfg.Inbounds,
		Outbounds: cfg.Outbounds,
	}
}

// rpc is the standard RPC implementation.
//
// It allows use of multiple Inbounds and Outbounds together.
type rpc struct {
	transport.Registry

	Name      string
	Inbounds  []transport.Inbound
	Outbounds transport.Outbounds
}

func (r rpc) Channel(service string) transport.Outbound {
	if out, ok := r.Outbounds[service]; ok {
		// we can eventually write an outbound that load balances between
		// known outbounds for a service.
		return out
	}
	panic(fmt.Sprintf("unknown service %q", service))
}

func (r rpc) Serve() error {
	callServe := func(i transport.Inbound) func() error {
		return func() error {
			return i.Serve(r)
		}
	}

	var wait sync.ErrorWaiter
	for _, i := range r.Inbounds {
		wait.Submit(callServe(i))
	}

	if errors := wait.Wait(); len(errors) > 0 {
		return errorGroup{Errors: errors}
	}

	return nil
}

func (r rpc) Handle(ctx context.Context, req *transport.Request) (*transport.Response, error) {
	h, err := r.GetHandler(req.Procedure)
	if err != nil {
		return nil, err
	}
	return h.Handle(ctx, req)
}

func (r rpc) Close() error {
	var wait sync.ErrorWaiter
	for _, i := range r.Inbounds {
		wait.Submit(i.Close)
	}

	if errors := wait.Wait(); len(errors) > 0 {
		return errorGroup{Errors: errors}
	}

	return nil
}
