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
	"github.com/yarpc/yarpc-go/internal/sync"
	"github.com/yarpc/yarpc-go/transport"

	"golang.org/x/net/context"
)

// Dispatcher object is used to configure a YARPC application; it is used by
// Clients to send RPCs, and by Procedures to recieve them. This object is what
// enables an application to be transport-agnostic.
type Dispatcher interface {
	transport.Handler
	transport.Registry

	// Retrieves a new Outbound transport that will make requests to the given
	// service.
	//
	// This panics if the given service is unknown.
	Channel(service string) transport.Channel

	// Inbounds returns a copy of the list of inbounds for this RPC object.
	//
	// The Inbounds will be returned in the same order that was used in the
	// configuration.
	Inbounds() []transport.Inbound

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

// NewDispatcher builds a new Dispatcher using the specified Config.
func NewDispatcher(cfg Config) Dispatcher {
	if cfg.Name == "" {
		panic("a service name is required")
	}

	return dispatcher{
		Name:        cfg.Name,
		Registry:    transport.NewMapRegistry(cfg.Name),
		inbounds:    cfg.Inbounds,
		Outbounds:   cfg.Outbounds,
		Filter:      cfg.Filter,
		Interceptor: cfg.Interceptor,
	}
}

// dispatcher is the standard RPC implementation.
//
// It allows use of multiple Inbounds and Outbounds together.
type dispatcher struct {
	transport.Registry

	Name        string
	Outbounds   transport.Outbounds
	Filter      transport.Filter
	Interceptor transport.Interceptor

	inbounds []transport.Inbound
}

func (d dispatcher) Inbounds() []transport.Inbound {
	inbounds := make([]transport.Inbound, len(d.inbounds))
	copy(inbounds, d.inbounds)
	return inbounds
}

func (d dispatcher) Channel(service string) transport.Channel {
	// TODO keep map[string]*Channel instead of Outbound when New is called. The
	// channels will allow persisting service-specific settings like "always
	// use this TTL for this service."

	if out, ok := d.Outbounds[service]; ok {
		// we can eventually write an outbound that load balances between
		// known outbounds for a service.
		out = transport.ApplyFilter(out, d.Filter)
		return transport.Channel{
			Outbound: request.ValidatorOutbound{Outbound: out},
			Caller:   d.Name,
			Service:  service,
		}
	}
	panic(noOutboundForService{Service: service})
}

func (d dispatcher) Start() error {
	startInbound := func(i transport.Inbound) func() error {
		return func() error {
			return i.Start(d)
		}
	}

	var wait sync.ErrorWaiter
	for _, i := range d.inbounds {
		wait.Submit(startInbound(i))
	}

	for _, o := range d.Outbounds {
		// TODO record the name of the service whose outbound failed
		wait.Submit(o.Start)
	}

	if errors := wait.Wait(); len(errors) > 0 {
		return errorGroup(errors)
	}
	return nil
}

func (d dispatcher) Register(service, procedure string, handler transport.Handler) {
	handler = transport.ApplyInterceptor(handler, d.Interceptor)
	d.Registry.Register(service, procedure, handler)
}

func (d dispatcher) Handle(ctx context.Context, req *transport.Request, rw transport.ResponseWriter) error {
	h, err := d.GetHandler(req.Service, req.Procedure)
	if err != nil {
		return err
	}
	return h.Handle(ctx, req, rw)
}

func (d dispatcher) Stop() error {
	var wait sync.ErrorWaiter
	for _, i := range d.inbounds {
		wait.Submit(i.Stop)
	}
	for _, o := range d.Outbounds {
		wait.Submit(o.Stop)
	}

	if errors := wait.Wait(); len(errors) > 0 {
		return errorGroup(errors)
	}

	return nil
}
