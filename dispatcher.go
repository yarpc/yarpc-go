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
	"context"
	"fmt"
	"sync"

	"go.uber.org/yarpc/api/middleware"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/internal"
	"go.uber.org/yarpc/internal/clientconfig"
	"go.uber.org/yarpc/internal/errors"
	"go.uber.org/yarpc/internal/request"
	intsync "go.uber.org/yarpc/internal/sync"

	"github.com/opentracing/opentracing-go"
)

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
	Unary  middleware.UnaryOutbound
	Oneway middleware.OnewayOutbound
}

// InboundMiddleware contains the different type of inbound middleware
type InboundMiddleware struct {
	Unary  middleware.UnaryInbound
	Oneway middleware.OnewayInbound
}

// NewDispatcher builds a new Dispatcher using the specified Config.
func NewDispatcher(cfg Config) *Dispatcher {
	if cfg.Name == "" {
		panic("yarpc.NewDispatcher expects a service name")
	}
	if err := internal.ValidateServiceName(cfg.Name); err != nil {
		panic("yarpc.NewDispatcher expects a valid service name: %s" + err.Error())
	}

	return &Dispatcher{
		name:              cfg.Name,
		table:             NewMapRouter(cfg.Name),
		inbounds:          cfg.Inbounds,
		outbounds:         convertOutbounds(cfg.Outbounds, cfg.OutboundMiddleware),
		transports:        collectTransports(cfg.Inbounds, cfg.Outbounds),
		inboundMiddleware: cfg.InboundMiddleware,
	}
}

// convertOutbounds applys outbound middleware and creates validator outbounds
func convertOutbounds(outbounds Outbounds, mw OutboundMiddleware) Outbounds {
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
			unaryOutbound = middleware.ApplyUnaryOutbound(outs.Unary, mw.Unary)
			unaryOutbound = request.UnaryValidatorOutbound{UnaryOutbound: unaryOutbound}
		}

		if outs.Oneway != nil {
			onewayOutbound = middleware.ApplyOnewayOutbound(outs.Oneway, mw.Oneway)
			onewayOutbound = request.OnewayValidatorOutbound{OnewayOutbound: onewayOutbound}
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

// Dispatcher object is used to configure a YARPC application; it is used by
// Clients to send RPCs, and by Procedures to recieve them. This object is what
// enables an application to be transport-agnostic.
type Dispatcher struct {
	table      transport.RouteTable
	name       string
	inbounds   Inbounds
	outbounds  Outbounds
	transports []transport.Transport

	inboundMiddleware InboundMiddleware
}

// Inbounds returns a copy of the list of inbounds for this RPC object.
//
// The Inbounds will be returned in the same order that was used in the
// configuration.
func (d *Dispatcher) Inbounds() Inbounds {
	inbounds := make(Inbounds, len(d.inbounds))
	copy(inbounds, d.inbounds)
	return inbounds
}

// ClientConfig produces a configuration object for an encoding-specific
// outbound RPC client.
//
// For example, pass the returned configuration object to client.New() for any
// generated Thrift client.
func (d *Dispatcher) ClientConfig(service string) transport.ClientConfig {
	if rs, ok := d.outbounds[service]; ok {
		return clientconfig.MultiOutbound(d.name, service, rs)
	}
	panic(noOutboundForService{Service: service})
}

// Procedures returns a list of services and procedures that have been
// registered with this Dispatcher.
func (d *Dispatcher) Procedures() []transport.Procedure {
	return d.table.Procedures()
}

// Choose picks a handler for the given request or returns an error if a
// handler for this request does not exist.
func (d *Dispatcher) Choose(ctx context.Context, req *transport.Request) (transport.HandlerSpec, error) {
	return d.table.Choose(ctx, req)
}

// Register configures the dispatcher's router to route inbound requests to a
// collection of procedure handlers.
func (d *Dispatcher) Register(rs []transport.Procedure) {
	procedures := make([]transport.Procedure, 0, len(rs))

	for _, r := range rs {
		switch r.HandlerSpec.Type() {
		case transport.Unary:
			h := middleware.ApplyUnaryInbound(r.HandlerSpec.Unary(),
				d.inboundMiddleware.Unary)
			r.HandlerSpec = transport.NewUnaryHandlerSpec(h)
		case transport.Oneway:
			h := middleware.ApplyOnewayInbound(r.HandlerSpec.Oneway(),
				d.inboundMiddleware.Oneway)
			r.HandlerSpec = transport.NewOnewayHandlerSpec(h)
		default:
			panic(fmt.Sprintf("unknown handler type %q for service %q, procedure %q",
				r.HandlerSpec.Type(), r.Service, r.Name))
		}

		procedures = append(procedures, r)
	}

	d.table.Register(procedures)
}

// Start Start the RPC allowing it to accept and processing new incoming
// requests.
//
// Blocks until the RPC is ready to start accepting new requests.
//
// Start goes through the Transports, Outbounds and Inbounds and starts them
// *NOTE* there can be problems if we don't start these in a particular order
// The order should be: Transports -> Outbounds -> Inbounds
// If the Outbounds are started before the Transports we might get a network
// request before the Transports are ready.
// If the Inbounds are started before the Outbounds an Inbound request might
// hit an Outbound before that Outbound is ready to take requests
func (d *Dispatcher) Start() error {
	var (
		mu         sync.Mutex
		allStarted []transport.Lifecycle
	)

	start := func(s transport.Lifecycle) func() error {
		return func() error {
			if s == nil {
				return nil
			}

			if err := s.Start(); err != nil {
				return err
			}

			mu.Lock()
			allStarted = append(allStarted, s)
			mu.Unlock()
			return nil
		}
	}

	abort := func(errs []error) error {
		// Failed to start so stop everything that was started.
		wait := intsync.ErrorWaiter{}
		for _, s := range allStarted {
			wait.Submit(s.Stop)
		}
		if newErrors := wait.Wait(); len(newErrors) > 0 {
			errs = append(errs, newErrors...)
		}

		return errors.ErrorGroup(errs)
	}

	// Start Transports
	wait := intsync.ErrorWaiter{}
	for _, t := range d.transports {
		wait.Submit(start(t))
	}
	if errs := wait.Wait(); len(errs) != 0 {
		return abort(errs)
	}

	// Start Outbounds
	wait = intsync.ErrorWaiter{}
	for _, o := range d.outbounds {
		wait.Submit(start(o.Unary))
		wait.Submit(start(o.Oneway))
	}
	if errs := wait.Wait(); len(errs) != 0 {
		return abort(errs)
	}

	// Start Inbounds
	wait = intsync.ErrorWaiter{}
	for _, i := range d.inbounds {
		i.SetRouter(d)
		wait.Submit(start(i))
	}
	if errs := wait.Wait(); len(errs) != 0 {
		return abort(errs)
	}

	return nil
}

// Stop goes through the Transports, Outbounds and Inbounds and stops them
// *NOTE* there can be problems if we don't stop these in a particular order
// The order should be: Inbounds -> Outbounds -> Transports
// If the Outbounds are stopped before the Inbounds we might get a network
// request to a stopped Outbound from a still-going Inbound.
// If the Transports are stopped before the Outbounds the `peers` contained in
// the Outbound might be `deleted` from the Transports perspective and cause
// issues
func (d *Dispatcher) Stop() error {
	var allErrs []error

	// Stop Inbounds
	wait := intsync.ErrorWaiter{}
	for _, i := range d.inbounds {
		wait.Submit(i.Stop)
	}
	if errs := wait.Wait(); len(errs) > 0 {
		allErrs = append(allErrs, errs...)
	}

	// Stop Outbounds
	wait = intsync.ErrorWaiter{}
	for _, o := range d.outbounds {
		if o.Unary != nil {
			wait.Submit(o.Unary.Stop)
		}
		if o.Oneway != nil {
			wait.Submit(o.Oneway.Stop)
		}
	}
	if errs := wait.Wait(); len(errs) > 0 {
		allErrs = append(allErrs, errs...)
	}

	// Stop Transports
	wait = intsync.ErrorWaiter{}
	for _, t := range d.transports {
		wait.Submit(t.Stop)
	}
	if errs := wait.Wait(); len(errs) > 0 {
		allErrs = append(allErrs, errs...)
	}

	if len(allErrs) > 0 {
		return errors.ErrorGroup(allErrs)
	}
	return nil
}
