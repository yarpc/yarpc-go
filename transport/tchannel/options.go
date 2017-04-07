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
	"net"

	"github.com/opentracing/opentracing-go"
)

// transportConfig is suitable for conveying options to TChannel transport
// constructors.
// At time of writing, there is only a ChannelTransport constructor, which
// supports options like WithChannel that only apply to this constructor form.
// The transportConfig should also be suitable, albeit with extraneous properties,
// if used for NewTransport, which will return a Transport suitable for YARPC
// peer lists.
// TODO update above when NewTransport is real.
type transportConfig struct {
	ch       Channel
	tracer   opentracing.Tracer
	addr     string
	listener net.Listener
	name     string
}

// TransportOption customizes the behavior of a TChannel Transport.
type TransportOption func(*transportConfig)

// Tracer specifies the request tracer used for RPCs passing through the
// TChannel transport.
func Tracer(tracer opentracing.Tracer) TransportOption {
	return func(t *transportConfig) {
		t.tracer = tracer
	}
}

// WithChannel specifies the TChannel Channel to use to send and receive YARPC
// requests. The instance may already have handlers registered against it;
// these will be left unchanged.
//
// If this option is not passed, the Transport will build and manage its own
// Channel. The behavior of that Tchannel may be customized using the
// ListenAddr and ServiceName options.
func WithChannel(ch Channel) TransportOption {
	return func(t *transportConfig) {
		t.ch = ch
	}
}

// ListenAddr specifies the port the TChannel should listen on.  This defaults
// to ":0" (all interfaces, OS-assigned port).
//
// 	transport := NewChannelTransport(ServiceName("myservice"), ListenAddr(":4040"))
//
// This option has no effect if WithChannel was used and the TChannel was
// already listening.
//
// This is deprecated in favor of Listener.
// If Listener is also specified, this option will be ignored.
func ListenAddr(addr string) TransportOption {
	return func(t *transportConfig) {
		t.addr = addr
	}
}

// Listener specifies the listener that TChannel should listen on.
func Listener(listener net.Listener) TransportOption {
	return func(t *transportConfig) {
		t.listener - listener
	}
}

// ServiceName informs the NewChannelTransport constructor which service
// name to use if it needs to construct a root Channel object, as when called
// without the WithChannel option.

// ServiceName specifies the name of the current service for the YARPC-owned
// TChannel Channel. If the WithChannel option is not specified, the TChannel
// Transport will build its own TChannel Chanel and use this name for that
// Channel.
//
// This option MUST be specified if WithChannel was not used. Note that this
// is the name of the LOCAL service, not the service you are trying to send
// requests to.
//
// This option has no effect if WithChannel was used.
func ServiceName(name string) TransportOption {
	return func(t *transportConfig) {
		t.name = name
	}
}
