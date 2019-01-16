// Copyright (c) 2019 Uber Technologies, Inc.
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

	"go.uber.org/multierr"
	"go.uber.org/net/metrics"
	"go.uber.org/yarpc/api/middleware"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/internal"
	"go.uber.org/yarpc/internal/inboundmiddleware"
	"go.uber.org/yarpc/internal/observability"
	"go.uber.org/yarpc/internal/outboundmiddleware"
	"go.uber.org/yarpc/internal/request"
	"go.uber.org/yarpc/pkg/lifecycle"
	"go.uber.org/zap"
)

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
	Stream middleware.StreamOutbound
}

// InboundMiddleware contains the different types of inbound middlewares.
type InboundMiddleware struct {
	Unary  middleware.UnaryInbound
	Oneway middleware.OnewayInbound
	Stream middleware.StreamInbound
}

// RouterMiddleware wraps the Router middleware
type RouterMiddleware middleware.Router

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
		panic("yarpc.NewDispatcher expects a valid service name: " + err.Error())
	}

	logger := cfg.Logging.logger(cfg.Name)
	extractor := cfg.Logging.extractor()

	meter, stopMeter := cfg.Metrics.scope(cfg.Name, logger)
	cfg = addObservingMiddleware(cfg, meter, logger, extractor)

	return &Dispatcher{
		name:              cfg.Name,
		table:             middleware.ApplyRouteTable(NewMapRouter(cfg.Name), cfg.RouterMiddleware),
		inbounds:          cfg.Inbounds,
		outbounds:         convertOutbounds(cfg.Outbounds, cfg.OutboundMiddleware),
		transports:        collectTransports(cfg.Inbounds, cfg.Outbounds),
		inboundMiddleware: cfg.InboundMiddleware,
		log:               logger,
		meter:             meter,
		stopMeter:         stopMeter,
		once:              lifecycle.NewOnce(),
	}
}

func addObservingMiddleware(cfg Config, meter *metrics.Scope, logger *zap.Logger, extractor observability.ContextExtractor) Config {
	if cfg.DisableAutoObservabilityMiddleware {
		return cfg
	}

	observer := observability.NewMiddleware(observability.Config{
		Logger:                logger,
		Scope:                 meter,
		ContextExtractor:      extractor,
		ApplicationErrorLevel: cfg.Logging.Levels.ApplicationError,
	})

	cfg.InboundMiddleware.Unary = inboundmiddleware.UnaryChain(observer, cfg.InboundMiddleware.Unary)
	cfg.InboundMiddleware.Oneway = inboundmiddleware.OnewayChain(observer, cfg.InboundMiddleware.Oneway)
	cfg.InboundMiddleware.Stream = inboundmiddleware.StreamChain(observer, cfg.InboundMiddleware.Stream)

	cfg.OutboundMiddleware.Unary = outboundmiddleware.UnaryChain(cfg.OutboundMiddleware.Unary, observer)
	cfg.OutboundMiddleware.Oneway = outboundmiddleware.OnewayChain(cfg.OutboundMiddleware.Oneway, observer)
	cfg.OutboundMiddleware.Stream = outboundmiddleware.StreamChain(cfg.OutboundMiddleware.Stream, observer)

	return cfg
}

// convertOutbounds applies outbound middleware and creates validator outbounds
func convertOutbounds(outbounds Outbounds, mw OutboundMiddleware) Outbounds {
	outboundSpecs := make(Outbounds, len(outbounds))

	for outboundKey, outs := range outbounds {
		if outs.Unary == nil && outs.Oneway == nil && outs.Stream == nil {
			panic(fmt.Sprintf("no outbound set for outbound key %q in dispatcher", outboundKey))
		}

		var (
			unaryOutbound  transport.UnaryOutbound
			onewayOutbound transport.OnewayOutbound
			streamOutbound transport.StreamOutbound
		)
		serviceName := outboundKey

		// apply outbound middleware and create ValidatorOutbounds
		if outs.Unary != nil {
			unaryOutbound = middleware.ApplyUnaryOutbound(outs.Unary, mw.Unary)
			unaryOutbound = request.UnaryValidatorOutbound{UnaryOutbound: unaryOutbound}
		}

		if outs.Oneway != nil {
			onewayOutbound = middleware.ApplyOnewayOutbound(outs.Oneway, mw.Oneway)
			onewayOutbound = request.OnewayValidatorOutbound{OnewayOutbound: onewayOutbound}
		}

		if outs.Stream != nil {
			streamOutbound = middleware.ApplyStreamOutbound(outs.Stream, mw.Stream)
			streamOutbound = request.StreamValidatorOutbound{StreamOutbound: streamOutbound}
		}

		if outs.ServiceName != "" {
			serviceName = outs.ServiceName
		}

		outboundSpecs[outboundKey] = transport.Outbounds{
			ServiceName: serviceName,
			Unary:       unaryOutbound,
			Oneway:      onewayOutbound,
			Stream:      streamOutbound,
		}
	}

	return outboundSpecs
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
		if stream := outbound.Stream; stream != nil {
			for _, transport := range stream.Transports() {
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

	log       *zap.Logger
	meter     *metrics.Scope
	stopMeter context.CancelFunc

	once *lifecycle.Once
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

// Outbounds returns a copy of the list of outbounds for this RPC object.
func (d *Dispatcher) Outbounds() Outbounds {
	outbounds := make(Outbounds, len(d.outbounds))
	for k, v := range d.outbounds {
		outbounds[k] = v
	}
	return outbounds
}

// ClientConfig provides the configuration needed to talk to the given
// service through an outboundKey. This configuration may be directly
// passed into encoding-specific RPC clients.
//
// 	keyvalueClient := json.New(dispatcher.ClientConfig("keyvalue"))
//
// This function panics if the outboundKey is not known.
func (d *Dispatcher) ClientConfig(outboundKey string) transport.ClientConfig {
	return d.MustOutboundConfig(outboundKey)
}

// MustOutboundConfig provides the configuration needed to talk to the given
// service through an outboundKey. This configuration may be directly
// passed into encoding-specific RPC clients.
//
// 	keyvalueClient := json.New(dispatcher.MustOutboundConfig("keyvalue"))
//
// This function panics if the outboundKey is not known.
func (d *Dispatcher) MustOutboundConfig(outboundKey string) *transport.OutboundConfig {
	if oc, ok := d.OutboundConfig(outboundKey); ok {
		return oc
	}
	panic(fmt.Sprintf("no configured outbound transport for outbound key %q", outboundKey))
}

// OutboundConfig provides the configuration needed to talk to the given
// service through an outboundKey. This configuration may be directly
// passed into encoding-specific RPC clients.
//
//  outboundConfig, ok := dispatcher.OutboundConfig("keyvalue")
//  if !ok {
//    // do something
//  }
// 	keyvalueClient := json.New(outboundConfig)
func (d *Dispatcher) OutboundConfig(outboundKey string) (oc *transport.OutboundConfig, ok bool) {
	if out, ok := d.outbounds[outboundKey]; ok {
		return &transport.OutboundConfig{
			CallerName: d.name,
			Outbounds:  out,
		}, true
	}
	return nil, false
}

// InboundMiddleware returns the middleware applied to all inbound handlers.
// Router middleware and fallback handlers can use the InboundMiddleware to
// wrap custom handlers.
func (d *Dispatcher) InboundMiddleware() InboundMiddleware {
	return d.inboundMiddleware
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
		case transport.Streaming:
			h := middleware.ApplyStreamInbound(r.HandlerSpec.Stream(),
				d.inboundMiddleware.Stream)
			r.HandlerSpec = transport.NewStreamHandlerSpec(h)
		default:
			panic(fmt.Sprintf("unknown handler type %q for service %q, procedure %q",
				r.HandlerSpec.Type(), r.Service, r.Name))
		}

		procedures = append(procedures, r)
		d.log.Info("Registration succeeded.", zap.Object("registeredProcedure", r))
	}

	d.table.Register(procedures)
}

// Start starts the Dispatcher, allowing it to accept and process new incoming
// requests. This starts all inbounds and outbounds configured on this
// Dispatcher.
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
//
// Start and PhasedStart are mutually exclusive. See the PhasedStart
// documentation for details.
func (d *Dispatcher) Start() error {
	starter := &PhasedStarter{
		dispatcher: d,
		log:        d.log,
	}
	return d.once.Start(func() error {
		d.log.Info("starting dispatcher")
		starter.setRouters()
		if err := starter.StartTransports(); err != nil {
			return err
		}
		if err := starter.StartOutbounds(); err != nil {
			return err
		}
		if err := starter.StartInbounds(); err != nil {
			return err
		}
		d.log.Info("dispatcher startup complete")
		return nil
	})
}

// PhasedStart is a more granular alternative to Start, and is intended only
// for advanced users. Rather than starting all transports, inbounds, and
// outbounds at once, it lets the user start them separately.
//
// Start and PhasedStart are mutually exclusive. If Start is called first,
// PhasedStart is a no-op and returns the same error (if any) that Start
// returned. If PhasedStart is called first, Start is a no-op and always
// returns a nil error; the caller is responsible for using the PhasedStarter
// to complete startup.
func (d *Dispatcher) PhasedStart() (*PhasedStarter, error) {
	starter := &PhasedStarter{
		dispatcher: d,
		log:        d.log,
	}
	if err := d.once.Start(func() error {
		starter.log.Info("beginning phased dispatcher start")
		starter.setRouters()
		return nil
	}); err != nil {
		return nil, err
	}
	return starter, nil
}

// Stop stops the Dispatcher, shutting down all inbounds, outbounds, and
// transports. This function returns after everything has been stopped.
//
// Stop and PhasedStop are mutually exclusive. See the PhasedStop
// documentation for details.
func (d *Dispatcher) Stop() error {
	stopper := &PhasedStopper{
		dispatcher: d,
		log:        d.log,
	}
	return d.once.Stop(func() error {
		d.log.Info("shutting down dispatcher")
		return multierr.Combine(
			stopper.StopInbounds(),
			stopper.StopOutbounds(),
			stopper.StopTransports(),
		)
	})
}

// PhasedStop is a more granular alternative to Stop, and is intended only for
// advanced users. Rather than stopping all inbounds, outbounds, and
// transports at once, it lets the user stop them separately.
//
// Stop and PhasedStop are mutually exclusive. If Stop is called first,
// PhasedStop is a no-op and returns the same error (if any) that Stop
// returned. If PhasedStop is called first, Stop is a no-op and always returns
// a nil error; the caller is responsible for using the PhasedStopper to
// complete shutdown.
func (d *Dispatcher) PhasedStop() (*PhasedStopper, error) {
	if err := d.once.Stop(func() error { return nil }); err != nil {
		return nil, err
	}
	return &PhasedStopper{
		dispatcher: d,
		log:        d.log,
	}, nil
}

// Router returns the procedure router.
func (d *Dispatcher) Router() transport.Router {
	return d.table
}

// Name returns the name of the dispatcher.
func (d *Dispatcher) Name() string {
	return d.name
}
