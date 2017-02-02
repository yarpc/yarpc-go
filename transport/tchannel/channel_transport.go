// Copyright (c) 2017 Uber Technologies, Inc.
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
	"errors"

	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/internal/sync"

	"github.com/opentracing/opentracing-go"
	"github.com/uber/tchannel-go"
)

var errChannelOrServiceNameIsRequired = errors.New(
	"cannot instantiate tchannel.ChannelTransport: " +
		"please provide a service name with the ServiceName option " +
		"or an existing Channel with the WithChannel option")

// NewChannelTransport is a YARPC transport that facilitates sending and
// receiving YARPC requests through TChannel. It uses a shared TChannel
// Channel for both, incoming and outgoing requests, ensuring reuse of
// connections and other resources.
//
// Either the local service name (with the ServiceName option) or a user-owned
// TChannel (with the WithChannel option) MUST be specified.
//
// ChannelTransport uses the underlying TChannel Channel for load balancing
// and peer managament.
// Use NewTransport and its NewOutbound to support YARPC peer.Choosers.
func NewChannelTransport(opts ...TransportOption) (*ChannelTransport, error) {
	var config transportConfig
	config.tracer = opentracing.GlobalTracer()
	for _, opt := range opts {
		opt(&config)
	}

	// Attempt to construct a channel on behalf of the caller if none given.
	// Defer the error until Start since NewChannelTransport does not have
	// an error return.
	var err error
	ch := config.ch

	if ch == nil {
		if config.name == "" {
			err = errChannelOrServiceNameIsRequired
		} else {
			opts := tchannel.ChannelOptions{Tracer: config.tracer}
			ch, err = tchannel.NewChannel(config.name, &opts)
		}
	}

	return &ChannelTransport{
		ch:     ch,
		addr:   config.addr,
		tracer: config.tracer,
	}, err
}

// ChannelTransport maintains TChannel peers and creates inbounds and outbounds for
// TChannel.
// If you have a YARPC peer.Chooser, use the unqualified tchannel.Transport
// instead.
type ChannelTransport struct {
	ch     Channel
	name   string
	addr   string
	tracer opentracing.Tracer
	router transport.Router

	once sync.LifecycleOnce
}

// Channel returns the underlying TChannel "Channel" instance.
func (t *ChannelTransport) Channel() Channel {
	return t.ch
}

// ListenAddr exposes the listen address of the transport.
func (t *ChannelTransport) ListenAddr() string {
	return t.addr
}

// Start starts the TChannel transport. This starts making connections and
// accepting inbound requests. All inbounds must have been assigned a router
// to accept inbound requests before this is called.
func (t *ChannelTransport) Start() error {
	return t.once.Start(t.start)
}

func (t *ChannelTransport) start() error {

	if t.router != nil {
		// Set up handlers. This must occur after construction because the
		// dispatcher, or its equivalent, calls SetRouter before Start.
		// This also means that SetRouter should be called on every inbound
		// before calling Start on any transport or inbound.
		sc := t.ch.GetSubChannel(t.ch.ServiceName())
		existing := sc.GetHandlers()
		sc.SetHandler(handler{existing: existing, router: t.router, tracer: t.tracer})
	}

	if t.ch.State() == tchannel.ChannelListening {
		// Channel.Start() was called before RPC.Start(). We still want to
		// update the Handler and what t.addr means, but nothing else.
		t.addr = t.ch.PeerInfo().HostPort
		return nil
	}

	// Default to ListenIP if addr wasn't given.
	addr := t.addr
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

	if err := t.ch.ListenAndServe(addr); err != nil {
		return err
	}

	t.addr = t.ch.PeerInfo().HostPort
	return nil
}

// Stop stops the TChannel transport. It starts rejecting incoming requests
// and draining connections before closing them.
// In a future version of YARPC, Stop will block until the underlying channel
// has closed completely.
func (t *ChannelTransport) Stop() error {
	return t.once.Stop(t.stop)
}

func (t *ChannelTransport) stop() error {
	t.ch.Close()
	return nil
}

// IsRunning returns whether the ChannelTransport is running.
func (t *ChannelTransport) IsRunning() bool {
	return t.once.IsRunning()
}
