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

package redispubsub

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/internal/errors"
	"go.uber.org/yarpc/internal/request"
	"go.uber.org/yarpc/serialize"

	"github.com/opentracing/opentracing-go"
	"go.uber.org/atomic"
)

const transportName = "redispubsub"

// Inbound is a redis inbound that subscribes to a redis channel.
type Inbound struct {
	registry transport.Registry
	tracer   opentracing.Tracer

	client  Client
	timeout time.Duration
	channel string
	started atomic.Bool
}

// NewInbound creates a redis Inbound that satisfies transport.Inbound.
//
// channel - subscription channel name
func NewInbound(client Client, channel string) *Inbound {
	return &Inbound{
		tracer:  opentracing.GlobalTracer(),
		client:  client,
		channel: channel,
	}
}

// Transports returns nil for now
func (i *Inbound) Transports() []transport.Transport {
	// TODO
	return nil
}

// WithTracer configures a tracer on this inbound.
func (i *Inbound) WithTracer(tracer opentracing.Tracer) *Inbound {
	i.tracer = tracer
	return i
}

// SetRegistry configures a registry to handle incoming requests.
// This satisfies the transport.Inbound interface, and would be called
// by a dispatcher when it starts.
func (i *Inbound) SetRegistry(registry transport.Registry) {
	i.registry = registry
}

// Start starts the inbound, reading from the queueKey
func (i *Inbound) Start() error {
	if i.started.Swap(true) {
		return nil
	}

	if i.registry == nil {
		return errors.NoRegistryError{}
	}

	if err := i.client.Start(); err != nil {
		return err
	}

	return i.client.Subscribe(i.channel, i.handleWithLogging)
}

// Stop closes the connection to redis
func (i *Inbound) Stop() error {
	if i.started.Swap(false) {
		return i.client.Stop()
	}
	return nil
}

func (i *Inbound) handleWithLogging(item []byte) {
	if err := i.handle(item); err != nil {
		log.Println("redispubsub error:", err)
	}
}

func (i *Inbound) handle(item []byte) error {
	start := time.Now()

	spanContext, req, err := serialize.FromBytes(i.tracer, item)
	if err != nil {
		return fmt.Errorf("could not deserialize item from bytes: %q", err)
	}

	extractOpenTracingSpan := transport.ExtractOpenTracingSpan{
		ParentSpanContext: spanContext,
		Tracer:            i.tracer,
		TransportName:     transportName,
		StartTime:         start,
	}
	ctx, span := extractOpenTracingSpan.Do(context.Background(), req)
	defer span.Finish()

	v := request.Validator{Request: req}
	req, err = v.Validate(ctx)
	if err != nil {
		return transport.UpdateSpanWithErr(span, err)
	}

	spec, err := i.registry.Choose(ctx, req)
	if err != nil {
		return transport.UpdateSpanWithErr(span, err)
	}

	if spec.Type() != transport.Oneway {
		err = errors.UnsupportedTypeError{Transport: transportName, Type: string(spec.Type())}
		return transport.UpdateSpanWithErr(span, err)
	}

	req, err = v.ValidateOneway(ctx)
	if err != nil {
		return transport.UpdateSpanWithErr(span, err)
	}

	return transport.DispatchOnewayHandler(ctx, spec.Oneway(), req)
}
