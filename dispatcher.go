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
	"sync"

	"go.uber.org/yarpc/internal/request"
	intsync "go.uber.org/yarpc/internal/sync"
	"go.uber.org/yarpc/transport"

	"github.com/opentracing/opentracing-go"
)

// Dispatcher object is used to configure a YARPC application; it is used by
// Clients to send RPCs, and by Procedures to recieve them. This object is what
// enables an application to be transport-agnostic.
type Dispatcher interface {
	transport.Registry
	transport.ChannelProvider

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

	Tracer opentracing.Tracer
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
		deps:        transport.NoDeps.WithTracer(cfg.Tracer),
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
	deps     transport.Deps
}

func (d dispatcher) Inbounds() []transport.Inbound {
	inbounds := make([]transport.Inbound, len(d.inbounds))
	copy(inbounds, d.inbounds)
	return inbounds
}

func (d dispatcher) Channel(service string) transport.Channel {
	if out, ok := d.Outbounds[service]; ok {
		out = transport.ApplyFilter(out, d.Filter)
		out = request.ValidatorOutbound{UnaryOutbound: out}
		return transport.IdentityChannel(d.Name, service, out)
	}
	panic(noOutboundForService{Service: service})
}

func (d dispatcher) Start() error {
	var (
		mu               sync.Mutex
		startedInbounds  []transport.Inbound
		startedOutbounds []transport.Outbound
	)

	service := transport.ServiceDetail{
		Name:     d.Name,
		Registry: d,
	}

	startInbound := func(i transport.Inbound) func() error {
		return func() error {
			if err := i.Start(service, d.deps); err != nil {
				return err
			}

			mu.Lock()
			startedInbounds = append(startedInbounds, i)
			mu.Unlock()
			return nil
		}
	}

	startOutbound := func(o transport.Outbound) func() error {
		return func() error {
			if err := o.Start(d.deps); err != nil {
				return err
			}

			mu.Lock()
			startedOutbounds = append(startedOutbounds, o)
			mu.Unlock()
			return nil
		}
	}

	var wait intsync.ErrorWaiter
	for _, i := range d.inbounds {
		wait.Submit(startInbound(i))
	}

	for _, o := range d.Outbounds {
		// TODO record the name of the service whose outbound failed
		wait.Submit(startOutbound(o))
	}

	errors := wait.Wait()
	if len(errors) == 0 {
		return nil
	}

	// Failed to start so stop everything that was started.
	wait = intsync.ErrorWaiter{}
	for _, i := range startedInbounds {
		wait.Submit(i.Stop)
	}
	for _, o := range startedOutbounds {
		wait.Submit(o.Stop)
	}

	if newErrors := wait.Wait(); len(newErrors) > 0 {
		errors = append(errors, newErrors...)
	}

	return errorGroup(errors)
}

func (d dispatcher) Register(rs []transport.Registrant) {
	registrants := make([]transport.Registrant, 0, len(rs))

	for _, r := range rs {
		switch r.HandlerSpec.Type() {
		case transport.Unary:
			h := transport.ApplyInterceptor(r.HandlerSpec.Unary(), d.Interceptor)
			r.HandlerSpec = transport.NewUnaryHandlerSpec(h)
		case transport.Oneway:
			//TODO(apb): add oneway interceptors https://github.com/yarpc/yarpc-go/issues/413
		default:
			panic(fmt.Sprintf("unknown handler type %q for service %q, procedure %q",
				r.HandlerSpec.Type(), r.Service, r.Procedure))
		}

		registrants = append(registrants, r)
	}

	d.Registry.Register(registrants)
}

func (d dispatcher) Stop() error {
	var wait intsync.ErrorWaiter
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
