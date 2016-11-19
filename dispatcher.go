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

	"go.uber.org/yarpc/internal/channel"
	"go.uber.org/yarpc/internal/request"
	intsync "go.uber.org/yarpc/internal/sync"
	"go.uber.org/yarpc/transport"

	"github.com/opentracing/opentracing-go"
)

// Dispatcher object is used to configure a YARPC application; it is used by
// Clients to send RPCs, and by Procedures to recieve them. This object is what
// enables an application to be transport-agnostic.
type Dispatcher interface {
	transport.Registrar
	transport.ChannelProvider

	// Inbounds returns a copy of the list of inbounds for this RPC object.
	//
	// The Inbounds will be returned in the same order that was used in the
	// configuration.
	Inbounds() Inbounds

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
	Name string

	Inbounds  Inbounds
	Outbounds Outbounds

	// Filter and Interceptor that will be applied to all outgoing and incoming
	// requests respectively.
	Filter      transport.Filter
	Interceptor transport.Interceptor

	Tracer opentracing.Tracer
}

// Inbounds contains a list of inbound transports
type Inbounds []transport.Inbound

// Outbounds encapsulates a service and its outbounds
type Outbounds map[string]transport.Outbounds

// NewDispatcher builds a new Dispatcher using the specified Config.
func NewDispatcher(cfg Config) Dispatcher {
	if cfg.Name == "" {
		panic("a service name is required")
	}

	return dispatcher{
		Name:        cfg.Name,
		Registrar:   transport.NewMapRegistry(cfg.Name),
		inbounds:    cfg.Inbounds,
		outbounds:   convertOutbounds(cfg.Outbounds, cfg.Filter),
		Interceptor: cfg.Interceptor,
		deps:        transport.NoDeps.WithTracer(cfg.Tracer),
	}
}

// convertOutbounds applys filters and creates validator outbounds
func convertOutbounds(outbounds Outbounds, filter transport.Filter) Outbounds {
	//TODO(apb): ensure we're not given the same underlying outbound for each RPC type
	convertedOutbounds := make(Outbounds, len(outbounds))

	for service, outs := range outbounds {
		var (
			unaryOutbound  transport.UnaryOutbound
			onewayOutbound transport.OnewayOutbound
		)

		// apply filters and create ValidatorOutbounds
		if outs.Unary != nil {
			unaryOutbound = transport.ApplyFilter(outs.Unary, filter)
			unaryOutbound = request.UnaryValidatorOutbound{UnaryOutbound: unaryOutbound}
		}

		// TODO(apb): apply oneway outbound filter
		if outs.Oneway != nil {
			onewayOutbound = request.OnewayValidatorOutbound{OnewayOutbound: outs.Oneway}
		}

		convertedOutbounds[service] = transport.Outbounds{
			Unary:  unaryOutbound,
			Oneway: onewayOutbound,
		}
	}

	return convertedOutbounds
}

// dispatcher is the standard RPC implementation.
//
// It allows use of multiple Inbounds and Outbounds together.
type dispatcher struct {
	transport.Registrar

	Name string

	inbounds  Inbounds
	outbounds Outbounds

	Interceptor transport.Interceptor

	deps transport.Deps
}

func (d dispatcher) Inbounds() Inbounds {
	inbounds := make(Inbounds, len(d.inbounds))
	copy(inbounds, d.inbounds)
	return inbounds
}

func (d dispatcher) Channel(service string) transport.Channel {
	if rs, ok := d.outbounds[service]; ok {
		return channel.MultiOutbound(d.Name, service, rs)
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
			if o == nil {
				return nil
			}

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

	// TODO record the name of the service whose outbound failed
	for _, o := range d.outbounds {
		wait.Submit(startOutbound(o.Unary))
		wait.Submit(startOutbound(o.Oneway))
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

	d.Registrar.Register(registrants)
}

func (d dispatcher) Stop() error {
	var wait intsync.ErrorWaiter
	for _, i := range d.inbounds {
		wait.Submit(i.Stop)
	}

	for _, o := range d.outbounds {
		if o.Unary != nil {
			wait.Submit(o.Unary.Stop)
		}
		if o.Oneway != nil {
			wait.Submit(o.Oneway.Stop)
		}
	}

	if errors := wait.Wait(); len(errors) > 0 {
		return errorGroup(errors)
	}

	return nil
}
