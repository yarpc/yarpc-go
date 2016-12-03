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

package transport

import (
	"bytes"
	"context"
	"time"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
)

// CreateOpentracingSpan creates a new context that has a reference to the
// started span.
// This should be called before a Outbound makes a call
func CreateOpentracingSpan(
	ctx context.Context,
	req *Request,
	tracer opentracing.Tracer,
	transportName string,
	start time.Time,
) (context.Context, opentracing.Span) {
	var parent opentracing.SpanContext
	if parentSpan := opentracing.SpanFromContext(ctx); parentSpan != nil {
		parent = parentSpan.Context()
	}

	span := tracer.StartSpan(
		req.Procedure,
		opentracing.StartTime(start),
		opentracing.ChildOf(parent),
		opentracing.Tags{
			"rpc.caller":    req.Caller,
			"rpc.service":   req.Service,
			"rpc.encoding":  req.Encoding,
			"rpc.transport": transportName,
		},
	)
	ext.PeerService.Set(span, req.Service)
	ext.SpanKindRPCClient.Set(span)

	ctx = opentracing.ContextWithSpan(ctx, span)
	return ctx, span
}

// MarshalSpanContext marshals a span.Context() into bytes
func MarshalSpanContext(tracer opentracing.Tracer, spanContext opentracing.SpanContext) ([]byte, error) {
	carrier := bytes.NewBuffer([]byte{})
	err := tracer.Inject(spanContext, opentracing.Binary, carrier)
	return carrier.Bytes(), err
}

// UnmarshalSpanContext coverts bytes into a span.Context()
func UnmarshalSpanContext(tracer opentracing.Tracer, spanContextBytes []byte) (opentracing.SpanContext, error) {
	carrier := bytes.NewBuffer(spanContextBytes)
	spanContext, err := tracer.Extract(opentracing.Binary, carrier)
	// If no SpanContext was given, we return nil instead of erroring
	// ExtractOpenTracingSpan safely accepts nil
	if err == opentracing.ErrSpanContextNotFound {
		return nil, nil
	}
	return spanContext, err
}

// ExtractOpenTracingSpan derives a new context from SpanContext. The created
// context has a reference to the started span. parentSpanCtx may be nil.
// This should be called before a Inbound handles a request
func ExtractOpenTracingSpan(
	ctx context.Context,
	parentSpanCtx opentracing.SpanContext,
	req *Request,
	tracer opentracing.Tracer,
	transportName string,
	start time.Time,
) (context.Context, opentracing.Span) {
	span := tracer.StartSpan(
		req.Procedure,
		opentracing.StartTime(start),
		opentracing.Tags{
			"rpc.caller":    req.Caller,
			"rpc.service":   req.Service,
			"rpc.encoding":  req.Encoding,
			"rpc.transport": transportName,
		},
		// parentSpanCtx may be nil
		// this implies ChildOf
		ext.RPCServerOption(parentSpanCtx),
	)
	ext.PeerService.Set(span, req.Caller)
	ext.SpanKindRPCServer.Set(span)

	ctx = opentracing.ContextWithSpan(ctx, span)
	return ctx, span
}

// UpdateSpanWithErr sets the error tag on a span, if an error is given.
// Returns the given error
func UpdateSpanWithErr(span opentracing.Span, err error) error {
	if err != nil {
		span.SetTag("error", true)
		span.LogEvent(err.Error())
	}
	return err
}
