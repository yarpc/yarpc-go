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

	"go.uber.org/yarpc/api/middleware"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/internal"
	"go.uber.org/yarpc/internal/clientconfig"
	"go.uber.org/yarpc/internal/errors"
	"go.uber.org/yarpc/internal/introspection"
	"go.uber.org/yarpc/internal/request"
	intsync "go.uber.org/yarpc/internal/sync"

	"github.com/opentracing/opentracing-go"
)

// Config specifies the parameters of a new Dispatcher constructed via
// NewDispatcher.
type Config struct {
	// Name of the service. This is the name used by other services when
	// making requests to this service.
	Name string

	// Inbounds define how this service receives incoming requests from other
	// services.
	//
	// This may be nil if this service does not receive any requests.
	Inbounds Inbounds

	// Outbounds defines how this service makes requests to other services.
	//
	// This may be nil if this service does not send any requests.
	Outbounds Outbounds

	// Inbound and Outbound Middleware that will be applied to all incoming
	// and outgoing requests respectively.
	//
	// These may be nil if there is no middleware to apply.
	InboundMiddleware  InboundMiddleware
	OutboundMiddleware OutboundMiddleware

	// Tracer is deprecated. The dispatcher does nothing with this propery.
	Tracer opentracing.Tracer
}

// Inbounds contains a list of inbound transports. Each inbound transport
// specifies a source through which incoming requests are received.
type Inbounds []transport.Inbound

// Outbounds provides access to outbounds for a remote service. Outbounds
// define how requests are sent from this service to the remote service.
type Outbounds map[string]transport.Outbounds

// OutboundMiddleware contains the different types of outbound middlewares.
type OutboundMiddleware struct {
	Unary  middleware.UnaryOutbound
	Oneway middleware.OnewayOutbound
}

// InboundMiddleware contains the different types of inbound middlewares.
type InboundMiddleware struct {
	Unary  middleware.UnaryInbound
	Oneway middleware.OnewayInbound
}

// NewDispatcher builds a new Dispatcher using the specified Config. At
// minimum, a service name must be specified.
//
// Invalid configurations or errors in constructing the Dispatcher will cause
// panics.
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

// Dispatcher encapsulates a YARPC application. It acts as the entry point to
// send and receive YARPC requests in a transport and encoding agnostic way.
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

// ClientConfig provides the configuration needed to talk to the given
// service. This configuration may be directly passed into encoding-specific
// RPC clients.
//
// 	keyvalueClient := json.New(dispatcher.ClientConfig("keyvalue"))
//
// This function panics if the service name is not known.
func (d *Dispatcher) ClientConfig(service string) transport.ClientConfig {
	if rs, ok := d.outbounds[service]; ok {
		return clientconfig.MultiOutbound(d.name, service, rs)
	}
	panic(noOutboundForService{Service: service})
}

// Register registers zero or more procedures with this dispatcher. Incoming
// requests to these procedures will be routed to the handlers specified in
// the given Procedures.
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

// Start starts the Dispatcher, allowing it to accept and processing new
// incoming requests.
//
// This starts all inbounds and outbounds configured on this Dispatcher.
//
// This function returns immediately after everything has been started.
// Servers should add a `select {}` to block to process all incoming requests.
//
// 	if err := dispatcher.Start(); err != nil {
// 		log.Fatal(err)
// 	}
// 	defer dispatcher.Stop()
//
// 	select {}
func (d *Dispatcher) Start() error {
	// NOTE: These MUST be started in the order transports, outbounds, and
	// then inbounds.
	//
	// If the outbounds are started before the transports, we might get a
	// network request before the transports are ready.
	//
	// If the inbounds are started before the outbounds, an inbound request
	// might result in an outbound call before the outbound is ready.

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
		i.SetRouter(d.table)
		wait.Submit(start(i))
	}
	if errs := wait.Wait(); len(errs) != 0 {
		return abort(errs)
	}

	addDispatcherToDebugPages(d)
	return nil
}

// Stop stops the Dispatcher.
//
// This stops all outbounds and inbounds owned by this Dispatcher.
//
// This function returns after everything has been stopped.
func (d *Dispatcher) Stop() error {
	// NOTE: These MUST be stopped in the order inbounds, outbounds, and then
	// transports.
	//
	// If the outbounds are stopped before the inbounds, we might receive a
	// request which needs to use a stopped outbound from a still-going
	// inbound.
	//
	// If the transports are stopped before the outbounds, the peers contained
	// in the outbound might be deleted from the transport's perspective and
	// cause issues.
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

	removeDispatcherFromDebugPages(d)
	return nil
}

type dispatcherStatus struct {
	Name       string
	ID         string
	Procedures []transport.Procedure
	Inbounds   []introspection.InboundStatus
	Outbounds  []introspection.OutboundStatus
}

func (d *Dispatcher) introspect() dispatcherStatus {
	var inbounds []introspection.InboundStatus
	for _, i := range d.inbounds {
		var status introspection.InboundStatus
		if i, ok := i.(introspection.IntrospectableInbound); ok {
			status = i.Introspect()
		} else {
			status = introspection.InboundStatus{
				Transport: "Introspection not supported",
			}
		}
		inbounds = append(inbounds, status)
	}
	var outbounds []introspection.OutboundStatus
	for destService, o := range d.outbounds {
		var status introspection.OutboundStatus
		if o.Unary != nil {
			if o, ok := o.Unary.(introspection.IntrospectableOutbound); ok {
				status = o.Introspect()
			} else {
				status.Transport = "Introspection not supported"
			}
			status.Type = "unary"
		}
		if o.Oneway != nil {
			if o, ok := o.Oneway.(introspection.IntrospectableOutbound); ok {
				status = o.Introspect()
			} else {
				status.Transport = "Introspection not supported"
			}
			status.Type = "oneway"
		}
		status.Service = destService
		outbounds = append(outbounds, status)
	}
	return dispatcherStatus{
		Name:       d.name,
		ID:         fmt.Sprintf("%p", d),
		Procedures: d.table.Procedures(),
		Inbounds:   inbounds,
		Outbounds:  outbounds,
	}
}
