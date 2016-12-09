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

package tchannel

import (
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/internal/debug"
	"go.uber.org/yarpc/internal/errors"

	"github.com/opentracing/opentracing-go"
)

// ChannelInbound is a TChannel Inbound backed by a pre-existing TChannel
// Channel.
type ChannelInbound struct {
	ch        Channel
	addr      string
	registry  transport.Registry
	tracer    opentracing.Tracer
	transport *ChannelTransport
}

// NewInbound returns a new TChannel inbound backed by a shared TChannel
// transport.  The returned ChannelInbound does not support peer.Chooser
// and uses TChannel's own internal load balancing peer selection.
func (t *ChannelTransport) NewInbound() *ChannelInbound {
	return &ChannelInbound{
		ch:        t.ch,
		tracer:    t.tracer,
		transport: t,
	}
}

// SetRegistry configures a registry to handle incoming requests.
// This satisfies the transport.Inbound interface, and would be called
// by a dispatcher when it starts.
func (i *ChannelInbound) SetRegistry(registry transport.Registry) {
	i.registry = registry
}

// Transports returns a slice containing the ChannelInbound's underlying
// ChannelTransport.
func (i *ChannelInbound) Transports() []transport.Transport {
	return []transport.Transport{i.transport}
}

// Channel returns the underlying Channel for this Inbound.
func (i *ChannelInbound) Channel() Channel {
	return i.ch
}

// Start starts a TChannel inbound. This arranges for registration of the
// request handler based on the registry given by SetRegistry.
//
// Start does not start listening sockets. That occurs when you Start the
// underlying ChannelTransport.
func (i *ChannelInbound) Start() error {
	if i.registry == nil {
		return errors.NoRegistryError{}
	}

	// Set up handlers. This must occur after construction because the
	// dispatcher, or its equivalent, calls SetRegistry before Start.
	// This also means that starting inbounds should block starting the transport.
	sc := i.ch.GetSubChannel(i.ch.ServiceName())
	existing := sc.GetHandlers()
	sc.SetHandler(handler{existing: existing, Registry: i.registry, tracer: i.tracer})

	return nil
}

// Stop stops the TChannel outbound. This currently does nothing.
func (i *ChannelInbound) Stop() error {
	return nil
}

func (i *ChannelInbound) Debug() debug.Inbound {
	c := i.transport.Channel()
	return debug.Inbound{
		Transport: "tchannel",
		Endpoint:  i.transport.ListenAddr(),
		State:     c.State().String(),
	}
}
