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

import (
	"github.com/opentracing/opentracing-go"
	"google.golang.org/grpc/grpclog"
)

// InboundOption is an option for an inbound.
type InboundOption func(*inboundOptions)

// OutboundOption is an option for an outbound.
type OutboundOption func(*outboundOptions)

// WithInboundTracer specifies the tracer to use for an inbound.
func WithInboundTracer(tracer opentracing.Tracer) InboundOption {
	return func(inboundOptions *inboundOptions) {
		inboundOptions.tracer = tracer
	}
}

// WithInboundOnewayErrorHandler specifiec the error handler to use for an inbound.
//
// The default is to call grpclog.Print(err).
func WithInboundOnewayErrorHandler(onewayErrorHandler func(error)) InboundOption {
	return func(inboundOptions *inboundOptions) {
		inboundOptions.onewayErrorHandler = onewayErrorHandler
	}
}

// WithOutboundTracer specifies the tracer to use for an outbound.
func WithOutboundTracer(tracer opentracing.Tracer) OutboundOption {
	return func(outboundOptions *outboundOptions) {
		outboundOptions.tracer = tracer
	}
}

type inboundOptions struct {
	tracer             opentracing.Tracer
	onewayErrorHandler func(error)
}

func newInboundOptions(options []InboundOption) *inboundOptions {
	inboundOptions := &inboundOptions{}
	for _, option := range options {
		option(inboundOptions)
	}
	return inboundOptions
}

func (i *inboundOptions) getTracer() opentracing.Tracer {
	if i.tracer == nil {
		return opentracing.GlobalTracer()
	}
	return i.tracer
}

func (i *inboundOptions) getOnewayErrorHandler() func(error) {
	if i.onewayErrorHandler == nil {
		return defaultOnewayErrorHandler
	}
	return i.onewayErrorHandler
}

type outboundOptions struct {
	tracer opentracing.Tracer
}

func newOutboundOptions(options []OutboundOption) *outboundOptions {
	outboundOptions := &outboundOptions{}
	for _, option := range options {
		option(outboundOptions)
	}
	return outboundOptions
}

func (o *outboundOptions) getTracer() opentracing.Tracer {
	if o.tracer == nil {
		return opentracing.GlobalTracer()
	}
	return o.tracer
}

func defaultOnewayErrorHandler(err error) {
	if err != nil {
		grpclog.Print(err)
	}
}
