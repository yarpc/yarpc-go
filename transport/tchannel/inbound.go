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
	"net"

	"go.uber.org/yarpc/internal/errors"
	"go.uber.org/yarpc/transport"

	"github.com/opentracing/opentracing-go"
	"github.com/uber/tchannel-go"
)

// NewInbound builds a new TChannel inbound from the given Channel. Existing
// methods registered on the channel remain registered and are preferred when
// a call is received.
func NewInbound(ch Channel) *Inbound {
	return &Inbound{
		ch:     ch,
		tracer: opentracing.GlobalTracer(),
	}
}

// Inbound is a TChannel Inbound.
type Inbound struct {
	ch       Channel
	addr     string
	listener net.Listener
	registry transport.Registry
	tracer   opentracing.Tracer
}

// WithListenAddr changes the address on which the TChannel server will listen
// for connections. By default, the server listens on an OS-assigned port.
//
// This option has no effect if the Chanel provided to NewInbound is already
// listening for connections when Start() is called.
func (i *Inbound) WithListenAddr(addr string) *Inbound {
	i.addr = addr
	return i
}

// WithTracer configures a tracer on the TChannel inbound.
func (i *Inbound) WithTracer(tracer opentracing.Tracer) *Inbound {
	i.tracer = tracer
	return i
}

// WithRegistry configures a registry to handle incoming requests,
// as a chained method for convenience.
func (i *Inbound) WithRegistry(registry transport.Registry) *Inbound {
	i.registry = registry
	return i
}

// SetRegistry configures a registry to handle incoming requests.
// This satisfies the transport.Inbound interface, and would be called
// by a dispatcher when it starts.
func (i *Inbound) SetRegistry(registry transport.Registry) {
	i.registry = registry
}

// Channel returns the underlying Channel for this Inbound.
func (i *Inbound) Channel() Channel {
	return i.ch
}

// Start starts the TChannel inbound transport. This immediately opens a listen
// socket.
func (i *Inbound) Start() error {
	if i.registry == nil {
		return errors.NoRegistryError{}
	}

	sc := i.ch.GetSubChannel(i.ch.ServiceName())
	existing := sc.GetHandlers()
	sc.SetHandler(handler{existing: existing, Registry: i.registry, tracer: i.tracer})

	if i.ch.State() == tchannel.ChannelListening {
		// Channel.Start() was called before RPC.Start(). We still want to
		// update the Handler and what i.addr means, but nothing else.
		i.addr = i.ch.PeerInfo().HostPort
		return nil
	}

	// Default to ListenIP if addr wasn't given.
	addr := i.addr
	if addr == "" {
		listenIP, err := tchannel.ListenIP()
		if err != nil {
			return err
		}

		addr = listenIP.String() + ":0"
		// TODO(abg): Find a way to export this to users
	}

	// TODO(abg): If addr was just the port (":4040"), we want to use
	// ListenIP() + ":4040" rather than just ":4040".

	if err := i.ch.ListenAndServe(addr); err != nil {
		return err
	}

	i.addr = i.ch.PeerInfo().HostPort
	return nil
}

// Stop stops the TChannel inbound transport. This immediately stops listening
// for incoming connections. Existing connections begin to drain.
// New inbound requests are rejected. When there are no further pending
// requests, TChannel closes the connection.
func (i *Inbound) Stop() error {
	i.ch.Close()
	return nil
}
