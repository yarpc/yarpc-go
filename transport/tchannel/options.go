// Copyright (c) 2022 Uber Technologies, Inc.
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
	"context"
	"crypto/tls"
	"net"
	"time"

	"github.com/opentracing/opentracing-go"
	"github.com/uber/tchannel-go"
	"go.uber.org/net/metrics"
	backoffapi "go.uber.org/yarpc/api/backoff"
	"go.uber.org/yarpc/internal/backoff"
	"go.uber.org/zap"
)

// Option allows customizing the YARPC TChannel transport.
// TransportSpec() accepts any TransportOption, and may in the future also
// accept inbound and outbound options.
type Option interface {
	tchannelOption()
}

var _ Option = (TransportOption)(nil)

// transportOptions is suitable for conveying options to TChannel transport
// constructors (NewTransport and NewChannelTransport).
// At time of writing, there is only a ChannelTransport constructor, which
// supports options like WithChannel that only apply to this constructor form.
// The transportOptions should also be suitable, albeit with extraneous properties,
// if used for NewTransport, which will return a Transport suitable for YARPC
// peer lists.
// TODO update above when NewTransport is real.
type transportOptions struct {
	ch                             Channel
	tracer                         opentracing.Tracer
	logger                         *zap.Logger
	meter                          *metrics.Scope
	addr                           string
	listener                       net.Listener
	dialer                         func(ctx context.Context, network, hostPort string) (net.Conn, error)
	name                           string
	connTimeout                    time.Duration
	connBackoffStrategy            backoffapi.Strategy
	originalHeaders                bool
	nativeTChannelMethods          NativeTChannelMethods
	excludeServiceHeaderInResponse bool
	muxTLSConfig                   *tls.Config
}

// newTransportOptions constructs the default transport options struct
func newTransportOptions() transportOptions {
	return transportOptions{
		tracer:              opentracing.GlobalTracer(),
		connTimeout:         defaultConnTimeout,
		connBackoffStrategy: backoff.DefaultExponential,
	}
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

// Logger sets a logger to use for internal logging.
//
// The default is to not write any logs.
func Logger(logger *zap.Logger) TransportOption {
	return func(t *transportOptions) {
		t.logger = logger
	}
}

// Meter sets a meter to use for transport metrics.
//
// The default is to not emit any transport metrics.
func Meter(meter *metrics.Scope) TransportOption {
	return func(t *transportOptions) {
		t.meter = meter
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

// Listener sets a net.Listener to use for the channel. This only applies to
// NewTransport (will not work with NewChannelTransport).
//
// The default is to depend on the ListenAddr (no-op).
func Listener(l net.Listener) TransportOption {
	return func(t *transportOptions) {
		t.listener = l
	}
}

// Dialer sets a dialer function for outbound calls.
//
// The function signature matches the net.Dialer DialContext method.
// https://golang.org/pkg/net/#Dialer.DialContext
func Dialer(dialer func(ctx context.Context, network, hostPort string) (net.Conn, error)) TransportOption {
	return func(t *transportOptions) {
		t.dialer = dialer
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

// ConnTimeout specifies the time that TChannel will wait for a
// connection attempt to any retained peer.
//
// The default is half of a second.
func ConnTimeout(d time.Duration) TransportOption {
	return func(options *transportOptions) {
		options.connTimeout = d
	}
}

// ConnBackoff specifies the connection backoff strategy for delays between
// connection attempts for each peer.
//
// ConnBackoff accepts a function that creates new backoff instances.
// The transport uses this to make referentially independent backoff instances
// that will not be shared across goroutines.
//
// The backoff instance is a function that accepts connection attempts and
// returns a duration.
//
// The default is exponential backoff starting with 10ms fully jittered,
// doubling each attempt, with a maximum interval of 30s.
func ConnBackoff(s backoffapi.Strategy) TransportOption {
	return func(options *transportOptions) {
		options.connBackoffStrategy = s
	}
}

// OriginalHeaders specifies whether to forward headers without canonicalizing them
func OriginalHeaders() TransportOption {
	return func(options *transportOptions) {
		options.originalHeaders = true
	}
}

// NativeTChannelMethods interface exposes methods which returns all
// the native TChannel methods and list of method names whose requests must
// be handled by the provided TChannel handlers.
type NativeTChannelMethods interface {
	// Methods returns a map of all the native handlers by method name.
	Methods() map[string]tchannel.Handler

	// SkipMethodNames returns a list of method names whose requests must be
	// handled by the provide TChannel handlers.
	SkipMethodNames() []string
}

// WithNativeTChannelMethods specifies the list of methods whose requests must
// be handled by the provided native TChannel method handlers.
//
// Requests with other methods will be handled by the Yarpc router.
// This is useful for gradual migration.
func WithNativeTChannelMethods(nativeMethods NativeTChannelMethods) TransportOption {
	return func(option *transportOptions) {
		option.nativeTChannelMethods = nativeMethods
	}
}

// ExcludeServiceHeaderInResponse stop adding the $rpc$-service response header for inbounds
func ExcludeServiceHeaderInResponse() TransportOption {
	return func(option *transportOptions) {
		option.excludeServiceHeaderInResponse = true
	}
}

// InboundMuxTLS returns a TransportOption that creates muxed listener which
// accepts inbound plaintext and TLS connections with the given TLS config.
func InboundMuxTLS(tlsConfig *tls.Config) TransportOption {
	return func(option *transportOptions) {
		option.muxTLSConfig = tlsConfig
	}
}
