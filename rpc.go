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
	"github.com/yarpc/yarpc-go/internal/request"
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
	Channel(service string) transport.Channel

	// Starts the RPC allowing it to accept and processing new incoming
	// requests.
	//
	// Blocks until the RPC is ready to start accepting new requests.
	Start() error

	// Stops the RPC. No new requests will be accepted.
	//
	// Blocks until the RPC has stopped.
	Stop() error
}

// Config specifies the parameters of a new RPC constructed via New.
type Config struct {
	Name      string
	Inbounds  []transport.Inbound
	Outbounds transport.Outbounds

	// Filter and Interceptor that will be applied to all outgoing and incoming
	// requests respectively.
	Filter      transport.Filter
	Interceptor transport.Interceptor

	// TODO FallbackHandler for catch-all endpoints
}

// New builds a new RPC using the specified configuration.
func New(cfg Config) RPC {
	if cfg.Name == "" {
		panic("a service name is required")
	}

	return rpc{
		Name:        cfg.Name,
		Registry:    transport.NewMapRegistry(cfg.Name),
		Inbounds:    cfg.Inbounds,
		Outbounds:   cfg.Outbounds,
		Filter:      cfg.Filter,
		Interceptor: cfg.Interceptor,
	}
}

// rpc is the standard RPC implementation.
//
// It allows use of multiple Inbounds and Outbounds together.
type rpc struct {
	transport.Registry

	Name        string
	Inbounds    []transport.Inbound
	Outbounds   transport.Outbounds
	Filter      transport.Filter
	Interceptor transport.Interceptor
}

func (r rpc) Channel(service string) transport.Channel {
	// TODO keep map[string]*Channel instead of Outbound when New is called. The
	// channels will allow persisting service-specific settings like "always
	// use this TTL for this service."

	if out, ok := r.Outbounds[service]; ok {
		// we can eventually write an outbound that load balances between
		// known outbounds for a service.
		out = transport.ApplyFilter(out, r.Filter)
		return transport.Channel{
			Outbound: request.ValidatorOutbound{Outbound: out},
			Caller:   r.Name,
			Service:  service,
		}
	}
	panic(noOutboundForService{Service: service})
}

func (r rpc) Start() error {
	callServe := func(i transport.Inbound) func() error {
		return func() error {
			return i.Start(r)
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

func (r rpc) Register(service, procedure string, handler transport.Handler) {
	handler = transport.ApplyInterceptor(handler, r.Interceptor)
	r.Registry.Register(service, procedure, handler)
}

func (r rpc) Handle(ctx context.Context, req *transport.Request, rw transport.ResponseWriter) error {
	h, err := r.GetHandler(req.Service, req.Procedure)
	if err != nil {
		return err
	}
	return h.Handle(ctx, req, rw)
}

func (r rpc) Stop() error {
	var wait sync.ErrorWaiter
	for _, i := range r.Inbounds {
		wait.Submit(i.Stop)
	}

	if errors := wait.Wait(); len(errors) > 0 {
		return errorGroup{Errors: errors}
	}

	return nil
}
