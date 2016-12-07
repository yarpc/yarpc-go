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

import "github.com/opentracing/opentracing-go"

// transportConfig is suitable for conveying options to TChannel transport
// constructors.
// At time of writing, there is only a ChannelTransport constructor, which
// supports options like WithChannel that only apply to this constructor form.
// The transportConfig should also be suitable, albeit with extraneous properties,
// if used for NewTransport, which will return a Transport suitable for YARPC
// peer lists.
// TODO update above when NewTransport is real.
type transportConfig struct {
	ch     Channel
	tracer opentracing.Tracer
	addr   string
	name   string
}

// TransportOption is for variadic arguments to NewChannelTransport.
//
// TransportOption will eventually also be suitable for passing to NewTransport.
type TransportOption func(*transportConfig)

// Tracer is an option that configures the tracer for a TChannel transport.
func Tracer(tracer opentracing.Tracer) TransportOption {
	return func(t *transportConfig) {
		t.tracer = tracer
	}
}

// WithChannel informs NewChannelTransport that it should reuse an existing
// underlying TChannel Channel instance. This instance may already have
// handlers and be listening before this transport starts. Otherwise,
// The TransportChannel will listen on start, albeit with the default address
// ":0" (all interfaces, any port).
func WithChannel(ch Channel) TransportOption {
	return func(t *transportConfig) {
		t.ch = ch
	}
}

// ListenAddr informs a transport constructor what address (in the form of
// host:port) to listen on. This option does not apply to NewChannelTransport
// if it is called with WithChannel and a channel that is already listening.
func ListenAddr(addr string) TransportOption {
	return func(t *transportConfig) {
		t.addr = addr
	}
}

// ServiceName informs the NewChannelTransport constructor which service
// name to use if it needs to construct a root Channel object, as when called
// without the WithChannel option.
func ServiceName(name string) TransportOption {
	return func(t *transportConfig) {
		t.name = name
	}
}
