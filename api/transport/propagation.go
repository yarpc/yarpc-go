// Copyright (c) 2024 Uber Technologies, Inc.
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
	"context"
	"time"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	opentracinglog "github.com/opentracing/opentracing-go/log"
	"go.uber.org/yarpc/yarpcerrors"
)

// CreateOpenTracingSpan creates a new context with a started span
type CreateOpenTracingSpan struct {
	Tracer        opentracing.Tracer
	TransportName string
	StartTime     time.Time
	ExtraTags     opentracing.Tags
}

const (
	// TracingTagStatusCode is the span tag key for the YAPRC status code.
	TracingTagStatusCode = "rpc.yarpc.status_code"
)

// Do creates a new context that has a reference to the started span.
// This should be called before a Outbound makes a call
func (c *CreateOpenTracingSpan) Do(
	ctx context.Context,
	req *Request,
) (context.Context, opentracing.Span) {
	var parent opentracing.SpanContext
	if parentSpan := opentracing.SpanFromContext(ctx); parentSpan != nil {
		parent = parentSpan.Context()
	}

	tags := opentracing.Tags{
		"rpc.caller":    req.Caller,
		"rpc.service":   req.Service,
		"rpc.encoding":  req.Encoding,
		"rpc.transport": c.TransportName,
	}
	for k, v := range c.ExtraTags {
		tags[k] = v
	}
	span := c.Tracer.StartSpan(
		req.Procedure,
		opentracing.StartTime(c.StartTime),
		opentracing.ChildOf(parent),
		tags,
	)
	ext.PeerService.Set(span, req.Service)
	ext.SpanKindRPCClient.Set(span)

	ctx = opentracing.ContextWithSpan(ctx, span)
	return ctx, span
}

// ExtractOpenTracingSpan derives a context and associated span
type ExtractOpenTracingSpan struct {
	ParentSpanContext opentracing.SpanContext
	Tracer            opentracing.Tracer
	TransportName     string
	StartTime         time.Time
	ExtraTags         opentracing.Tags
}

// Do derives a new context from SpanContext. The created context has a
// reference to the started span. parentSpanCtx may be nil.
// This should be called before a Inbound handles a request
func (e *ExtractOpenTracingSpan) Do(
	ctx context.Context,
	req *Request,
) (context.Context, opentracing.Span) {
	tags := opentracing.Tags{
		"rpc.caller":    req.Caller,
		"rpc.service":   req.Service,
		"rpc.encoding":  req.Encoding,
		"rpc.transport": e.TransportName,
	}
	for k, v := range e.ExtraTags {
		tags[k] = v
	}
	span := e.Tracer.StartSpan(
		req.Procedure,
		opentracing.StartTime(e.StartTime),
		tags,
		// parentSpanCtx may be nil
		// this implies ChildOf
		ext.RPCServerOption(e.ParentSpanContext),
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
		span.LogFields(opentracinglog.String("event", err.Error()))
	}
	return err
}

// UpdateSpanWithApplicationErr sets the error tag and status code on a span for application error.
// The error message is intentionally omitted to prevent exposing
// personally identifiable information (PII) in tracing systems.
// Returns the given error
func UpdateSpanWithApplicationErr(span opentracing.Span, err error, errCode yarpcerrors.Code) error {
	if err != nil {
		span.SetTag("error", true)
		span.SetTag(TracingTagStatusCode, errCode)
		span.LogFields(
			opentracinglog.String("event", "error"),
		)
	}
	return err
}
