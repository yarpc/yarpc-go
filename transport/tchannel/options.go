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

import "github.com/opentracing/opentracing-go"

// Option allows customizing the YARPC TChannel transport.
// TransportSpec() accepts any TransportOption, and may in the future also
// accept inbound and outbound options.
type Option interface {
	tchannelOption()
}

var _ Option = (TransportOption)(nil)

// transportOptions is suitable for conveying options to TChannel transport
// constructors (NewTransport and NewChannelTransport).
type transportOptions struct {
	ch     Channel
	tracer opentracing.Tracer
	addr   string
	name   string
}

// TransportOption customizes the behavior of a TChannel Transport.
type TransportOption func(*transportOptions)

// TransportOption makes all TransportOptions recognizeable as Option so
// TransportSpec will accept them.
func (TransportOption) tchannelOption() {}

// Tracer specifies the request tracer used for RPCs passing through the
// TChannel transport.
func Tracer(tracer opentracing.Tracer) TransportOption {
	return func(t *transportOptions) {
		t.tracer = tracer
	}
}

// WithChannel specifies the TChannel Channel to use to send and receive YARPC
// requests. The instance may already have handlers registered against it;
// these will be left unchanged.
//
// If this option is not passed, the Transport will build and manage its own
// Channel. The behavior of that TChannel may be customized using the
// ListenAddr and ServiceName options.
//
// This option is disallowed for NewTransport and transports constructed with
// the YARPC configuration system.
func WithChannel(ch Channel) TransportOption {
	return func(options *transportOptions) {
		options.ch = ch
	}
}

// ListenAddr specifies the port the TChannel should listen on.  This defaults
// to ":0" (all interfaces, OS-assigned port).
//
// 	transport := NewChannelTransport(ServiceName("myservice"), ListenAddr(":4040"))
//
// This option has no effect if WithChannel was used and the TChannel was
// already listening, and it is disallowed for transports constructed with the
// YARPC configuration system.
func ListenAddr(addr string) TransportOption {
	return func(options *transportOptions) {
		options.addr = addr
	}
}

// ServiceName informs the NewChannelTransport constructor which service
// name to use if it needs to construct a root Channel object, as when called
// without the WithChannel option.
//
// ServiceName specifies the name of the current service for the YARPC-owned
// TChannel Channel. If the WithChannel option is not specified, the TChannel
// Transport will build its own TChannel Chanel and use this name for that
// Channel.
//
// This option has no effect if WithChannel was used, and it is disallowed for
// transports constructed with the YARPC configuration system.
func ServiceName(name string) TransportOption {
	return func(options *transportOptions) {
		options.name = name
	}
}
