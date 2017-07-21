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

package grpc

import "google.golang.org/grpc"

// Option is an interface shared by TransportOption, InboundOption, and OutboundOption
// allowing either to be recognized by TransportSpec().
type Option interface {
	grpcOption()
}

var _ Option = (TransportOption)(nil)
var _ Option = (InboundOption)(nil)
var _ Option = (OutboundOption)(nil)

// TransportOption is an option for a transport.
type TransportOption func(*transportOptions)

func (TransportOption) grpcOption() {}

// InboundOption is an option for an inbound.
type InboundOption func(*inboundOptions)

func (InboundOption) grpcOption() {}

// OutboundOption is an option for an outbound.
type OutboundOption func(*outboundOptions)

func (OutboundOption) grpcOption() {}

type transportOptions struct{}

func newTransportOptions(options []TransportOption) *transportOptions {
	transportOptions := &transportOptions{}
	for _, option := range options {
		option(transportOptions)
	}
	return transportOptions
}

type inboundOptions struct {
	unaryInterceptor grpc.UnaryServerInterceptor
}

func newInboundOptions(options []InboundOption) *inboundOptions {
	inboundOptions := &inboundOptions{}
	for _, option := range options {
		option(inboundOptions)
	}
	return inboundOptions
}

// TODO: this should cover the tracer interceptor too
// grpc-go only allows one interceptor, so need to handle all cases
// working on this with go-grpc-middleware
func (i *inboundOptions) getUnaryInterceptor() grpc.UnaryServerInterceptor {
	return i.unaryInterceptor
}

type outboundOptions struct{}

func newOutboundOptions(options []OutboundOption) *outboundOptions {
	outboundOptions := &outboundOptions{}
	for _, option := range options {
		option(outboundOptions)
	}
	return outboundOptions
}

// for testing only for now
func withInboundUnaryInterceptor(unaryInterceptor grpc.UnaryServerInterceptor) InboundOption {
	return func(inboundOptions *inboundOptions) {
		inboundOptions.unaryInterceptor = unaryInterceptor
	}
}
