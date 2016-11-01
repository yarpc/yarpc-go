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
	Name string

	Inbounds       []transport.Inbound
	RemoteServices []RemoteService

	// Filter and Interceptor that will be applied to all outgoing and incoming
	// requests respectively.
	Filter      transport.UnaryFilter
	Interceptor transport.UnaryInterceptor

	Tracer opentracing.Tracer
}

// RemoteService encapsulates a remote service and its outbounds
type RemoteService struct {
	Name string

	UnaryOutbound  transport.UnaryOutbound
	OnewayOutbound transport.OnewayOutbound
}

// NewDispatcher builds a new Dispatcher using the specified Config.
func NewDispatcher(cfg Config) Dispatcher {
	if cfg.Name == "" {
		panic("a service name is required")
	}

	return dispatcher{
		Name:           cfg.Name,
		Registry:       transport.NewMapRegistry(cfg.Name),
		inbounds:       cfg.Inbounds,
		RemoteServices: convertRemoteServices(cfg.RemoteServices, cfg.Filter),
		Interceptor:    cfg.Interceptor,
		deps:           transport.NoDeps.WithTracer(cfg.Tracer),
	}
}

func convertRemoteServices(
	remoteServices []RemoteService,
	filter transport.UnaryFilter) map[string]transport.RemoteService {
	services := make(map[string]transport.RemoteService, len(remoteServices))

	for _, rs := range remoteServices {
		// This ensures that we don't apply filters/validators to the same outbound
		//	more than once. This can be the case if one object implements multiple
		//	outbound types
		seen := make(map[transport.Outbound]transport.Outbound, 2)

		outbound := rs.UnaryOutbound
		onewayOutbound := rs.OnewayOutbound

		// apply filters and create ValidatorOutbounds
		if rs.UnaryOutbound != nil {
			original := rs.UnaryOutbound

			outbound = transport.ApplyFilter(rs.UnaryOutbound, filter)
			outbound = request.ValidatorOutbound{UnaryOutbound: outbound}

			seen[original] = outbound
		}

		if rs.OnewayOutbound != nil {
			original := rs.OnewayOutbound

			// TODO: apply oneway outbound filter
			if o, ok := seen[original]; ok {
				onewayOutbound = o.(transport.OnewayOutbound)
			} else {
				onewayOutbound = request.OnewayValidatorOutbound{
					OnewayOutbound: rs.OnewayOutbound,
				}
			}
		}

		services[rs.Name] = transport.RemoteService{
			Name:           rs.Name,
			UnaryOutbound:  outbound,
			OnewayOutbound: onewayOutbound,
		}
	}

	return services
}

// dispatcher is the standard RPC implementation.
//
// It allows use of multiple Inbounds and Outbounds together.
type dispatcher struct {
	transport.Registry

	Name string

	RemoteServices map[string]transport.RemoteService

	//TODO: get rid of these, can just apply filter in NewDispatcher
	Filter      transport.UnaryFilter
	Interceptor transport.UnaryInterceptor

	inbounds []transport.Inbound

	deps transport.Deps
}

func (d dispatcher) Inbounds() []transport.Inbound {
	inbounds := make([]transport.Inbound, len(d.inbounds))
	copy(inbounds, d.inbounds)
	return inbounds
}

func (d dispatcher) Channel(service string) transport.Channel {
	if rs, ok := d.RemoteServices[service]; ok {
		return transport.MultiOutboundChannel(d.Name, rs)
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

	// TODO record the name of the service whose outbound failed
	for _, o := range d.getUniqueOutbounds() {
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
	for i, r := range rs {
		if r.HandlerSpec.Type == transport.Unary {
			r.HandlerSpec.UnaryHandler = transport.ApplyUnaryInterceptor(r.HandlerSpec.UnaryHandler, d.Interceptor)
			rs[i] = r
		}
		//TODO add oneway interceptors
	}
	d.Registry.Register(rs)
}

func (d dispatcher) Stop() error {
	var wait intsync.ErrorWaiter
	for _, i := range d.inbounds {
		wait.Submit(i.Stop)
	}

	for _, o := range d.getUniqueOutbounds() {
		wait.Submit(o.Stop)
	}

	if errors := wait.Wait(); len(errors) > 0 {
		return errorGroup(errors)
	}

	return nil
}

func (d dispatcher) getUniqueOutbounds() []transport.Outbound {
	var unique []transport.Outbound

	for _, rs := range d.RemoteServices {
		if rs.UnaryOutbound == nil {
			unique = append(unique, rs.OnewayOutbound)
		} else if rs.OnewayOutbound == nil {
			unique = append(unique, rs.UnaryOutbound)
		} else if rs.UnaryOutbound.(transport.Outbound) == rs.OnewayOutbound.(transport.Outbound) {
			unique = append(unique, rs.UnaryOutbound)
		} else {
			unique = append(unique, rs.UnaryOutbound, rs.OnewayOutbound)
		}
	}

	return unique
}
