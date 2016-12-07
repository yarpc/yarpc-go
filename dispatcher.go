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

	"go.uber.org/yarpc/internal/clientconfig"
	"go.uber.org/yarpc/internal/errors"
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
	transport.ClientConfigProvider

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

	// Inbound and Outbound Middleware that will be applied to all incoming and
	// outgoing requests respectively.
	InboundMiddleware  InboundMiddleware
	OutboundMiddleware OutboundMiddleware

	// Tracer is deprecated. The dispatcher does nothing with this propery.
	Tracer opentracing.Tracer
}

// Inbounds contains a list of inbound transports
type Inbounds []transport.Inbound

// Outbounds encapsulates a service and its outbounds
type Outbounds map[string]transport.Outbounds

// OutboundMiddleware contains the different type of outbound middleware
type OutboundMiddleware struct {
	Unary  transport.UnaryOutboundMiddleware
	Oneway transport.OnewayOutboundMiddleware
}

// InboundMiddleware contains the different type of inbound middleware
type InboundMiddleware struct {
	Unary  transport.UnaryInboundMiddleware
	Oneway transport.OnewayInboundMiddleware
}

// NewDispatcher builds a new Dispatcher using the specified Config.
func NewDispatcher(cfg Config) Dispatcher {
	if cfg.Name == "" {
		panic("a service name is required")
	}

	return dispatcher{
		Name:              cfg.Name,
		Registrar:         transport.NewMapRegistry(cfg.Name),
		inbounds:          cfg.Inbounds,
		outbounds:         convertOutbounds(cfg.Outbounds, cfg.OutboundMiddleware),
		transports:        collectTransports(cfg.Inbounds, cfg.Outbounds),
		InboundMiddleware: cfg.InboundMiddleware,
	}
}

// convertOutbounds applys outbound middleware and creates validator outbounds
func convertOutbounds(outbounds Outbounds, middleware OutboundMiddleware) Outbounds {
	convertedOutbounds := make(Outbounds, len(outbounds))

	for service, outs := range outbounds {
		if outs.Unary == nil && outs.Oneway == nil {
			panic(fmt.Sprintf("no outbound set for service %q in dispatcher", service))
		}

		var (
			unaryOutbound  transport.UnaryOutbound
			onewayOutbound transport.OnewayOutbound
		)

		// apply outbound middleware and create ValidatorOutbounds
		if outs.Unary != nil {
			unaryOutbound = transport.ApplyUnaryOutboundMiddleware(outs.Unary, middleware.Unary)
			unaryOutbound = request.UnaryValidatorOutbound{UnaryOutbound: unaryOutbound}
		}

		if outs.Oneway != nil {
			onewayOutbound = transport.ApplyOnewayOutboundMiddleware(outs.Oneway, middleware.Oneway)
			onewayOutbound = request.OnewayValidatorOutbound{OnewayOutbound: outs.Oneway}
		}

		convertedOutbounds[service] = transport.Outbounds{
			Unary:  unaryOutbound,
			Oneway: onewayOutbound,
		}
	}

	return convertedOutbounds
}

// collectTransports iterates over all inbounds and outbounds and collects all
// of their unique underlying transports. Multiple inbounds and outbounds may
// share a transport, and we only want the dispatcher to manage their lifecycle
// once.
func collectTransports(inbounds Inbounds, outbounds Outbounds) []transport.Transport {
	// Collect all unique transports from inbounds and outbounds.
	transports := make(map[transport.Transport]struct{})
	for _, inbound := range inbounds {
		for _, transport := range inbound.Transports() {
			transports[transport] = struct{}{}
		}
	}
	for _, outbound := range outbounds {
		if unary := outbound.Unary; unary != nil {
			for _, transport := range unary.Transports() {
				transports[transport] = struct{}{}
			}
		}
		if oneway := outbound.Oneway; oneway != nil {
			for _, transport := range oneway.Transports() {
				transports[transport] = struct{}{}
			}
		}
	}
	keys := make([]transport.Transport, 0, len(transports))
	for key := range transports {
		keys = append(keys, key)
	}
	return keys
}

// dispatcher is the standard RPC implementation.
//
// It allows use of multiple Inbounds and Outbounds together.
type dispatcher struct {
	transport.Registrar

	Name string

	inbounds   Inbounds
	outbounds  Outbounds
	transports []transport.Transport

	InboundMiddleware InboundMiddleware
}

func (d dispatcher) Inbounds() Inbounds {
	inbounds := make(Inbounds, len(d.inbounds))
	copy(inbounds, d.inbounds)
	return inbounds
}

func (d dispatcher) ClientConfig(service string) transport.ClientConfig {
	if rs, ok := d.outbounds[service]; ok {
		return clientconfig.MultiOutbound(d.Name, service, rs)
	}
	panic(noOutboundForService{Service: service})
}

func (d dispatcher) Start() error {
	var (
		mu                sync.Mutex
		startedTransports []transport.Transport
		startedInbounds   []transport.Inbound
		startedOutbounds  []transport.Outbound
	)

	startInbound := func(i transport.Inbound) func() error {
		return func() error {
			if err := i.Start(); err != nil {
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

			if err := o.Start(); err != nil {
				return err
			}

			mu.Lock()
			startedOutbounds = append(startedOutbounds, o)
			mu.Unlock()
			return nil
		}
	}

	startTransport := func(t transport.Transport) func() error {
		return func() error {
			if err := t.Start(); err != nil {
				return err
			}

			mu.Lock()
			startedTransports = append(startedTransports, t)
			mu.Unlock()
			return nil
		}
	}

	abort := func(errs []error) error {
		// Failed to start so stop everything that was started.
		wait := intsync.ErrorWaiter{}
		for _, i := range startedInbounds {
			wait.Submit(i.Stop)
		}
		for _, o := range startedOutbounds {
			wait.Submit(o.Stop)
		}
		for _, t := range startedTransports {
			wait.Submit(t.Stop)
		}

		if newErrors := wait.Wait(); len(newErrors) > 0 {
			errs = append(errs, newErrors...)
		}

		return errors.ErrorGroup(errs)
	}

	// Start inbounds and outbounds in parallel

	var wait intsync.ErrorWaiter
	for _, i := range d.inbounds {
		i.SetRegistry(d)
		wait.Submit(startInbound(i))
	}

	// TODO record the name of the service whose outbound failed
	for _, o := range d.outbounds {
		wait.Submit(startOutbound(o.Unary))
		wait.Submit(startOutbound(o.Oneway))
	}

	// Synchronize
	errs := wait.Wait()
	if len(errs) != 0 {
		return abort(errs)
	}

	// Start transports
	wait = intsync.ErrorWaiter{}
	for _, t := range d.transports {
		wait.Submit(startTransport(t))
	}

	// Synchronize
	errs = wait.Wait()
	if len(errs) != 0 {
		return abort(errs)
	}

	return nil
}

func (d dispatcher) Register(rs []transport.Registrant) {
	registrants := make([]transport.Registrant, 0, len(rs))

	for _, r := range rs {
		switch r.HandlerSpec.Type() {
		case transport.Unary:
			h := transport.ApplyUnaryInboundMiddleware(r.HandlerSpec.Unary(),
				d.InboundMiddleware.Unary)
			r.HandlerSpec = transport.NewUnaryHandlerSpec(h)
		case transport.Oneway:
			h := transport.ApplyOnewayInboundMiddleware(r.HandlerSpec.Oneway(),
				d.InboundMiddleware.Oneway)
			r.HandlerSpec = transport.NewOnewayHandlerSpec(h)
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

	for _, t := range d.transports {
		wait.Submit(t.Stop)
	}

	if errs := wait.Wait(); len(errs) > 0 {
		return errors.ErrorGroup(errs)
	}

	return nil
}
