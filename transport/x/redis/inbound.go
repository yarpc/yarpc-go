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

package redis

import (
	"context"
	"time"

	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/internal/errors"
	"go.uber.org/yarpc/internal/request"
	"go.uber.org/yarpc/serialize"

	"github.com/opentracing/opentracing-go"
)

const transportName = "redis"

// Inbound is a redis inbound that reads from the given queueKey. This will
// wait for an item in the queue or until the timout is reached before trying
// to read again.
type Inbound struct {
	registry transport.Registry
	tracer   opentracing.Tracer

	client        Client
	timeout       time.Duration
	queueKey      string
	processingKey string

	stop chan struct{}
}

// NewInbound creates a redis Inbound that satisfies transport.Inbound.
//
// queueKey - key for the queue in redis
// processingKey - key for the list we'll store items we've popped from the queue
// timeout - how long the inbound will block on reading from redis
func NewInbound(client Client, queueKey, processingKey string, timeout time.Duration) *Inbound {
	return &Inbound{
		tracer: opentracing.GlobalTracer(),

		client:        client,
		timeout:       timeout,
		queueKey:      queueKey,
		processingKey: processingKey,

		stop: make(chan struct{}),
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

// WithRegistry configures a registry to handle incoming requests,
// as a chained method for convenience.
func (i *Inbound) WithRegistry(registry transport.Registry) *Inbound {
	i.registry = registry
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
	if i.registry == nil {
		return errors.NoRegistryError{}
	}

	err := i.client.Start()
	if err != nil {
		return err
	}

	go i.start()
	return nil
}

func (i *Inbound) start() {
	for {
		select {
		case <-i.stop:
			return
		default:
			// TODO: logging
			i.handle()
		}
	}
}

// Stop ends the connection to redis
func (i *Inbound) Stop() error {
	close(i.stop)
	return i.client.Stop()
}

func (i *Inbound) handle() error {
	// TODO: logging
	item, err := i.client.BRPopLPush(i.queueKey, i.processingKey, i.timeout)
	if err != nil {
		return err
	}
	defer i.client.LRem(i.queueKey, item)

	start := time.Now()

	spanContext, req, err := serialize.FromBytes(i.tracer, item)
	if err != nil {
		return err
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
	if err := v.ValidateCommon(ctx); err != nil {
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

	if err := v.ValidateOneway(ctx); err != nil {
		return transport.UpdateSpanWithErr(span, err)
	}

	return transport.DispatchOnewayHandler(ctx, spec.Oneway(), req)
}
