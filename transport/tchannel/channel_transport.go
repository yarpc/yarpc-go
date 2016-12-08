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
	"fmt"

	"github.com/opentracing/opentracing-go"
	"github.com/uber/tchannel-go"
)

// NewChannelTransport creates a ChannelTransport, suitable for creating inbounds
// and outbounds with an existing, shared channel.
//
// The ChannelTransport uses the underlying TChannel load balancing and peer
// management, so it is not suitable for use with a peer.Chooser.
// A future version of YARPC will add a NewTransport constructor that returns
// a transport suitable for custom peer selection.
func NewChannelTransport(opts ...TransportOption) *ChannelTransport {
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
	if config.ch == nil {
		if config.name == "" {
			err = fmt.Errorf("can't instantiate TChannelChannelTransport without channel or service name option")
		} else {
			ch, err = tchannel.NewChannel(config.name, &tchannel.ChannelOptions{
				Tracer: config.tracer,
			})
		}
	}

	return &ChannelTransport{
		ch:     ch,
		err:    err,
		addr:   config.addr,
		tracer: config.tracer,
	}
}

// ChannelTransport maintains TChannel peers and creates inbounds and outbounds for
// TChannel.
//
// In a future version, the channel will be suitable for managing peers in a
// peer.List or other peer.Chooser.
type ChannelTransport struct {
	ch     Channel
	name   string
	err    error
	addr   string
	tracer opentracing.Tracer
}

// Channel returns the underlying TChannel "Channel" instance.
func (t *ChannelTransport) Channel() Channel {
	return t.ch
}

// ListenAddr exposes the listen address of the transport.
func (t *ChannelTransport) ListenAddr() string {
	return t.addr
}

// Start starts a TChannel transport, opening listening sockets and accepting
// inbound requests, and opening connections to retained peers.
//
// All inbounds must have been assigned a registry to accept inbound requests.
func (t *ChannelTransport) Start() error {
	// Return error deferred from constructor for the construction of a TChannel.
	if t.err != nil {
		return t.err
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

// Stop stops a TChannel transport, closing listening sockets, rejecting
// inbound requests, draining pending requests, and closing all connections.
//
// Stop blocks until the program can gracefully exit.
func (t *ChannelTransport) Stop() error {
	t.ch.Close()
	return nil
}
