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
	"strings"
	"sync"
	"time"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	opentracinglog "github.com/opentracing/opentracing-go/log"
)

const (
	tchannelTracingKeyPrefix      = "$tracing$"
	tchannelTracingKeyMappingSize = 100
)

// CreateOpenTracingSpan creates a new context with a started span
type CreateOpenTracingSpan struct {
	Tracer        opentracing.Tracer
	TransportName string
	StartTime     time.Time
	ExtraTags     opentracing.Tags
}

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

// GetPropagationFormat returns the opentracing propagation depends on transport.
// For TChannel, the format is opentracing.TextMap
// For HTTP and gRPC, the format is opentracing.HTTPHeaders
func GetPropagationFormat(transport string) opentracing.BuiltinFormat {
	if transport == "tchannel" {
		return opentracing.TextMap
	}
	return opentracing.HTTPHeaders
}

// PropagationCarrier is an interface to combine both reader and writer interface
type PropagationCarrier interface {
	opentracing.TextMapReader
	opentracing.TextMapWriter
}

// GetPropagationCarrier get the propagation carrier depends on the transport.
// The carrier is used for accessing the transport headers.
// For TChannel, a special carrier is used. For details, see comments of TChannelHeadersCarrier
func GetPropagationCarrier(headers map[string]string, transport string) PropagationCarrier {
	if transport == "tchannel" {
		return TChannelHeadersCarrier(headers)
	}
	return opentracing.TextMapCarrier(headers)
}

// TChannelHeadersCarrier is a dedicated carrier for TChannel.
// When writing the tracing headers into headers, the $tracing$ prefix is added to each tracing header key.
// When reading the tracing headers from headers, the $tracing$ prefix is removed from each tracing header key.
type TChannelHeadersCarrier map[string]string

var _ PropagationCarrier = TChannelHeadersCarrier{}

// ForeachKey iterates over all tracing headers in the carrier, applying the provided
// handler function to each header after stripping the $tracing$ prefix from the keys.
func (c TChannelHeadersCarrier) ForeachKey(handler func(string, string) error) error {
	for k, v := range c {
		if !strings.HasPrefix(k, tchannelTracingKeyPrefix) {
			continue
		}
		noPrefixKey := tchannelTracingKeyDecoding.mapAndCache(k)
		if err := handler(noPrefixKey, v); err != nil {
			return err
		}
	}
	return nil
}

// Set adds a tracing header to the carrier, prefixing the key with $tracing$ before storing it.
func (c TChannelHeadersCarrier) Set(key, value string) {
	prefixedKey := tchannelTracingKeyEncoding.mapAndCache(key)
	c[prefixedKey] = value
}

// tchannelTracingKeysMapping is to optimize the efficiency of tracing header key manipulations.
// The implementation is forked from tchannel-go: https://github.com/uber/tchannel-go/blob/dev/tracing_keys.go#L36
type tchannelTracingKeysMapping struct {
	sync.RWMutex
	mapping map[string]string
	mapper  func(key string) string
}

var tchannelTracingKeyEncoding = &tchannelTracingKeysMapping{
	mapping: make(map[string]string),
	mapper: func(key string) string {
		return tchannelTracingKeyPrefix + key
	},
}

var tchannelTracingKeyDecoding = &tchannelTracingKeysMapping{
	mapping: make(map[string]string),
	mapper: func(key string) string {
		return key[len(tchannelTracingKeyPrefix):]
	},
}

func (m *tchannelTracingKeysMapping) mapAndCache(key string) string {
	m.RLock()
	v, ok := m.mapping[key]
	m.RUnlock()
	if ok {
		return v
	}
	m.Lock()
	defer m.Unlock()
	if v, ok := m.mapping[key]; ok {
		return v
	}
	mappedKey := m.mapper(key)
	if len(m.mapping) < tchannelTracingKeyMappingSize {
		m.mapping[key] = mappedKey
	}
	return mappedKey
}
