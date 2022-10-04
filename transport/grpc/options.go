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

package grpc

import (
	"context"
	"crypto/tls"
	"math"
	"net"

	opentracing "github.com/opentracing/opentracing-go"
	"go.uber.org/net/metrics"
	"go.uber.org/yarpc/api/backoff"
	"go.uber.org/yarpc/api/transport"
	yarpctls "go.uber.org/yarpc/api/transport/tls"
	intbackoff "go.uber.org/yarpc/internal/backoff"
	"go.uber.org/yarpc/transport/internal/tls/dialer"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"
)

const (
	// defensive programming
	// these are copied from grpc-go but we set them explicitly here
	// in case these change in grpc-go so that yarpc stays consistent
	defaultServerMaxSendMsgSize = math.MaxInt32
	defaultClientMaxSendMsgSize = math.MaxInt32
	// Overriding default server and client maximum request and response
	// receive sizes to 64MB.
	defaultServerMaxRecvMsgSize = 1024 * 1024 * 64
	defaultClientMaxRecvMsgSize = 1024 * 1024 * 64
)

// Option is an interface shared by TransportOption, InboundOption, and OutboundOption
// allowing either to be recognized by TransportSpec().
type Option interface {
	grpcOption()
}

var _ Option = (TransportOption)(nil)
var _ Option = (InboundOption)(nil)
var _ Option = (OutboundOption)(nil)
var _ Option = (DialOption)(nil)

// TransportOption is an option for a transport.
type TransportOption func(*transportOptions)

func (TransportOption) grpcOption() {}

// ServiceName specifices the name of the service used in transport logging
// and metrics.
func ServiceName(name string) TransportOption {
	return func(transportOptions *transportOptions) {
		transportOptions.serviceName = name
	}
}

// BackoffStrategy specifies the backoff strategy for delays between
// connection attempts for each peer.
//
// The default is exponential backoff starting with 10ms fully jittered,
// doubling each attempt, with a maximum interval of 30s.
func BackoffStrategy(backoffStrategy backoff.Strategy) TransportOption {
	return func(transportOptions *transportOptions) {
		transportOptions.backoffStrategy = backoffStrategy
	}
}

// Tracer specifies the tracer to use.
//
// By default, opentracing.GlobalTracer() is used.
func Tracer(tracer opentracing.Tracer) TransportOption {
	return func(transportOptions *transportOptions) {
		transportOptions.tracer = tracer
	}
}

// Logger sets a logger to use for internal logging.
//
// The default is to not write any logs.
func Logger(logger *zap.Logger) TransportOption {
	return func(transportOptions *transportOptions) {
		transportOptions.logger = logger
	}
}

// Meter sets a meter to use for transport metrics.
//
// The default is to not emit any metrics.
func Meter(meter *metrics.Scope) TransportOption {
	return func(transportOptions *transportOptions) {
		transportOptions.meter = meter
	}
}

// ServerMaxRecvMsgSize is the maximum message size the server can receive.
//
// The default is 4MB.
func ServerMaxRecvMsgSize(serverMaxRecvMsgSize int) TransportOption {
	return func(transportOptions *transportOptions) {
		transportOptions.serverMaxRecvMsgSize = serverMaxRecvMsgSize
	}
}

// ServerMaxSendMsgSize is the maximum message size the server can send.
//
// The default is unlimited.
func ServerMaxSendMsgSize(serverMaxSendMsgSize int) TransportOption {
	return func(transportOptions *transportOptions) {
		transportOptions.serverMaxSendMsgSize = serverMaxSendMsgSize
	}
}

// ServerMaxHeaderListSize returns a transport option for configuring maximum
// header list size the server must accept.
//
// The default is 16MB (gRPC default).
func ServerMaxHeaderListSize(serverMaxHeaderListSize uint32) TransportOption {
	return func(transportOptions *transportOptions) {
		transportOptions.serverMaxHeaderListSize = &serverMaxHeaderListSize
	}
}

// ClientMaxRecvMsgSize is the maximum message size the client can receive.
//
// The default is 4MB.
func ClientMaxRecvMsgSize(clientMaxRecvMsgSize int) TransportOption {
	return func(transportOptions *transportOptions) {
		transportOptions.clientMaxRecvMsgSize = clientMaxRecvMsgSize
	}
}

// ClientMaxSendMsgSize is the maximum message size the client can send.
//
// The default is unlimited.
func ClientMaxSendMsgSize(clientMaxSendMsgSize int) TransportOption {
	return func(transportOptions *transportOptions) {
		transportOptions.clientMaxSendMsgSize = clientMaxSendMsgSize
	}
}

// ClientMaxHeaderListSize returns a transport option for configuring maximum
// header list size the client must accept.
//
// The default is 16MB (gRPC default).
func ClientMaxHeaderListSize(clientMaxHeaderListSize uint32) TransportOption {
	return func(transportOptions *transportOptions) {
		transportOptions.clientMaxHeaderListSize = &clientMaxHeaderListSize
	}
}

// InboundOption is an option for an inbound.
type InboundOption func(*inboundOptions)

func (InboundOption) grpcOption() {}

// InboundCredentials returns an InboundOption that sets credentials for incoming
// connections.
func InboundCredentials(creds credentials.TransportCredentials) InboundOption {
	return func(inboundOptions *inboundOptions) {
		inboundOptions.creds = creds
	}
}

// InboundTLSConfiguration returns an InboundOption that provides the TLS confiugration
// used for setting up TLS inbound.
func InboundTLSConfiguration(config *tls.Config) InboundOption {
	return func(inboundOptions *inboundOptions) {
		inboundOptions.tlsConfig = config
	}
}

// InboundTLSMode returns an InboundOption that sets inbound TLS mode.
// It must be noted that TLS configuration must be passed separately using inbound
// option InboundTLSConfiguration.
func InboundTLSMode(mode yarpctls.Mode) InboundOption {
	return func(inboundOptions *inboundOptions) {
		inboundOptions.tlsMode = mode
	}
}

// OutboundOption is an option for an outbound.
type OutboundOption func(*outboundOptions)

func (OutboundOption) grpcOption() {}

// OutboundTLSConfigProvider returns an OutboundOption that provides the
// outbound TLS config provider.
func OutboundTLSConfigProvider(provider yarpctls.OutboundTLSConfigProvider) OutboundOption {
	return func(outboundOptions *outboundOptions) {
		outboundOptions.tlsConfigProvider = provider
	}
}

// DialOption is an option that influences grpc.Dial.
type DialOption func(*dialOptions)

func (DialOption) grpcOption() {}

// DialerCredentials returns a DialOption which configures a
// connection level security credentials (e.g., TLS/SSL).
func DialerCredentials(creds credentials.TransportCredentials) DialOption {
	return func(dialOptions *dialOptions) {
		dialOptions.creds = creds
	}
}

// DialerTLSConfig return a DialOption which configures the TLS config for the
// outbound.
func DialerTLSConfig(config *tls.Config) DialOption {
	return func(dialOptions *dialOptions) {
		dialOptions.tlsConfig = config
	}
}

// DialerDestinationServiceName returns a DialOption which configures the
// destination service name of the dialer. This is used in TLS dialer metrics.
func DialerDestinationServiceName(service string) DialOption {
	return func(dialOptions *dialOptions) {
		dialOptions.destServiceName = service
	}
}

// ContextDialer sets the dialer for creating outbound connections.
//
// See https://godoc.org/google.golang.org/grpc#WithContextDialer for more
// details.
func ContextDialer(f func(context.Context, string) (net.Conn, error)) DialOption {
	return func(dialOptions *dialOptions) {
		dialOptions.contextDialer = f
	}
}

// Compressor sets the compressor to be used by default for gRPC connections
func Compressor(compressor transport.Compressor) DialOption {
	return func(dialOptions *dialOptions) {
		if compressor != nil {
			// We assume that the grpc-go compressor was also globally
			// registered and just use the name.
			// Future implementations may elect to actually use the compressor.
			dialOptions.defaultCompressor = compressor.Name()
		}
	}
}

// KeepaliveParams sets the gRPC keepalive parameters of the outbound
// connection.
// See https://pkg.go.dev/google.golang.org/grpc#WithKeepaliveParams for more
// details.
func KeepaliveParams(params keepalive.ClientParameters) DialOption {
	return func(dialOptions *dialOptions) {
		dialOptions.keepaliveParams = &params
	}
}

type transportOptions struct {
	backoffStrategy         backoff.Strategy
	tracer                  opentracing.Tracer
	logger                  *zap.Logger
	meter                   *metrics.Scope
	serviceName             string
	serverMaxRecvMsgSize    int
	serverMaxSendMsgSize    int
	clientMaxRecvMsgSize    int
	clientMaxSendMsgSize    int
	serverMaxHeaderListSize *uint32
	clientMaxHeaderListSize *uint32
}

func newTransportOptions(options []TransportOption) *transportOptions {
	transportOptions := &transportOptions{
		backoffStrategy:      intbackoff.DefaultExponential,
		serverMaxRecvMsgSize: defaultServerMaxRecvMsgSize,
		serverMaxSendMsgSize: defaultServerMaxSendMsgSize,
		clientMaxRecvMsgSize: defaultClientMaxRecvMsgSize,
		clientMaxSendMsgSize: defaultClientMaxSendMsgSize,
	}
	for _, option := range options {
		option(transportOptions)
	}
	if transportOptions.logger == nil {
		transportOptions.logger = zap.NewNop()
	}
	if transportOptions.tracer == nil {
		transportOptions.tracer = opentracing.GlobalTracer()
	}
	return transportOptions
}

type inboundOptions struct {
	creds credentials.TransportCredentials

	tlsConfig *tls.Config
	tlsMode   yarpctls.Mode
}

func newInboundOptions(options []InboundOption) *inboundOptions {
	inboundOptions := &inboundOptions{}
	for _, option := range options {
		option(inboundOptions)
	}
	return inboundOptions
}

type outboundOptions struct {
	tlsConfigProvider yarpctls.OutboundTLSConfigProvider
}

func newOutboundOptions(options []OutboundOption) *outboundOptions {
	outboundOptions := &outboundOptions{}
	for _, option := range options {
		option(outboundOptions)
	}
	return outboundOptions
}

type dialOptions struct {
	creds             credentials.TransportCredentials
	contextDialer     func(context.Context, string) (net.Conn, error)
	defaultCompressor string
	keepaliveParams   *keepalive.ClientParameters
	tlsConfig         *tls.Config
	destServiceName   string
}

func (d *dialOptions) grpcOptions(t *Transport) []grpc.DialOption {
	credsOption := grpc.WithInsecure()
	if d.creds != nil {
		credsOption = grpc.WithTransportCredentials(d.creds)
	}

	opts := []grpc.DialOption{
		credsOption,
	}

	if d.defaultCompressor != "" {
		opts = append(opts, grpc.WithDefaultCallOptions(grpc.UseCompressor(d.defaultCompressor)))
	}

	if d.keepaliveParams != nil {
		opts = append(opts, grpc.WithKeepaliveParams(*d.keepaliveParams))
	}

	if d.tlsConfig != nil {
		params := dialer.Params{
			Config:        d.tlsConfig,
			Meter:         t.options.meter,
			Logger:        t.options.logger,
			ServiceName:   t.options.serviceName,
			TransportName: TransportName,
			Dest:          d.destServiceName,
		}

		if d.contextDialer != nil {
			params.Dialer = func(ctx context.Context, network, addr string) (net.Conn, error) {
				return d.contextDialer(ctx, addr)
			}
		}
		tlsDialer := dialer.NewTLSDialer(params)
		opts = append(opts, grpc.WithContextDialer(func(ctx context.Context, addr string) (net.Conn, error) {
			return tlsDialer.DialContext(ctx, "tcp", addr)
		}))
	} else if d.contextDialer != nil {
		opts = append(opts, grpc.WithContextDialer(d.contextDialer))
	}

	return opts
}

func newDialOptions(options []DialOption) *dialOptions {
	var dopts dialOptions
	for _, option := range options {
		option(&dopts)
	}
	return &dopts
}
