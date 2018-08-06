// Copyright (c) 2018 Uber Technologies, Inc.
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
	"math"

	"go.uber.org/yarpc/api/backoff"
	intbackoff "go.uber.org/yarpc/internal/backoff"

	"github.com/opentracing/opentracing-go"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const (
	// defensive programming
	// these are copied from grpc-go but we set them explicitly here
	// in case these change in grpc-go so that yarpc stays consistent
	defaultServerMaxRecvMsgSize = 1024 * 1024 * 4
	defaultServerMaxSendMsgSize = math.MaxInt32
	defaultClientMaxRecvMsgSize = 1024 * 1024 * 4
	defaultClientMaxSendMsgSize = math.MaxInt32
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

// OutboundOption is an option for an outbound.
type OutboundOption func(*outboundOptions)

func (OutboundOption) grpcOption() {}

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

type transportOptions struct {
	backoffStrategy      backoff.Strategy
	tracer               opentracing.Tracer
	logger               *zap.Logger
	serverMaxRecvMsgSize int
	serverMaxSendMsgSize int
	clientMaxRecvMsgSize int
	clientMaxSendMsgSize int
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
	if transportOptions.tracer == nil {
		transportOptions.tracer = opentracing.NoopTracer{}
	}
	return transportOptions
}

type inboundOptions struct {
	creds credentials.TransportCredentials
}

func newInboundOptions(options []InboundOption) *inboundOptions {
	inboundOptions := &inboundOptions{}
	for _, option := range options {
		option(inboundOptions)
	}
	return inboundOptions
}

type outboundOptions struct{}

func newOutboundOptions(options []OutboundOption) *outboundOptions {
	outboundOptions := &outboundOptions{}
	for _, option := range options {
		option(outboundOptions)
	}
	return outboundOptions
}

type dialOptions struct {
	creds credentials.TransportCredentials
}

func (d *dialOptions) grpcOptions() []grpc.DialOption {
	var credsOption grpc.DialOption
	if d == nil || d.creds == nil {
		credsOption = grpc.WithInsecure()
	} else {
		credsOption = grpc.WithTransportCredentials(d.creds)
	}
	return []grpc.DialOption{credsOption}
}

func newDialOptions(options []DialOption) *dialOptions {
	var dopts dialOptions
	for _, option := range options {
		option(&dopts)
	}
	return &dopts
}
