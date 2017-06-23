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

package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/opentracing/opentracing-go"
	"go.uber.org/multierr"
	"go.uber.org/yarpc"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/internal/introspection"
	"go.uber.org/yarpc/internal/sync"
	"go.uber.org/yarpc/serialize"
)

const transportName = "redis"

const maxConnectRetries = 100

var connectRetryDelay = 10 * time.Millisecond

// Inbound is a redis inbound that reads from the given queueKey. This will
// wait for an item in the queue or until the timout is reached before trying
// to read again.
type Inbound struct {
	router transport.Router
	tracer opentracing.Tracer

	client        Client
	timeout       time.Duration
	queueKey      string
	processingKey string

	stop chan struct{}

	once sync.LifecycleOnce
}

// NewInbound creates a redis Inbound that satisfies transport.Inbound.
//
// queueKey - key for the queue in redis
// processingKey - key for the list we'll store items we've popped from the queue
// timeout - how long the inbound will block on reading from redis
func NewInbound(client Client, queueKey, processingKey string, timeout time.Duration) *Inbound {
	return &Inbound{
		tracer: opentracing.GlobalTracer(),
		once:   sync.Once(),

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

// WithRouter configures a router to handle incoming requests,
// as a chained method for convenience.
func (i *Inbound) WithRouter(router transport.Router) *Inbound {
	i.router = router
	return i
}

// SetRouter configures a router to handle incoming requests.
// This satisfies the transport.Inbound interface, and would be called
// by a dispatcher when it starts.
func (i *Inbound) SetRouter(router transport.Router) {
	i.router = router
}

// Start starts the inbound, reading from the queueKey
func (i *Inbound) Start() error {
	return i.once.Start(i.start)
}

func (i *Inbound) start() error {
	if i.router == nil {
		return yarpc.InternalErrorf("no router configured for transport inbound")
	}

	var err error
	for attempt := 0; attempt < maxConnectRetries; attempt++ {
		err = i.client.Start()
		if err == nil {
			break
		}
		time.Sleep(connectRetryDelay)
	}
	if err != nil {
		return err
	}

	go i.startLoop()
	return nil
}

func (i *Inbound) startLoop() {
	for {
		select {
		case <-i.stop:
			return
		default:
			// TODO: log error
			_ = i.handle()
		}
	}
}

// Stop ends the connection to redis
func (i *Inbound) Stop() error {
	return i.once.Stop(i.stopClient)
}

func (i *Inbound) stopClient() error {
	close(i.stop)
	return i.client.Stop()
}

// IsRunning returns whether the inbound is still processing requests.
func (i *Inbound) IsRunning() bool {
	return i.once.IsRunning()
}

func (i *Inbound) handle() (err error) {
	// TODO: logging
	item, err := i.client.BRPopLPush(i.queueKey, i.processingKey, i.timeout)
	if err != nil {
		return err
	}
	defer func() {
		err = multierr.Append(err, i.client.LRem(i.queueKey, item))
	}()

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

	if err := transport.ValidateRequest(req); err != nil {
		return transport.UpdateSpanWithErr(span, err)
	}

	spec, err := i.router.Choose(ctx, req)
	if err != nil {
		return transport.UpdateSpanWithErr(span, err)
	}

	if spec.Type() != transport.Oneway {
		err = yarpc.UnimplementedErrorf("transport:%s type:%s", transportName, spec.Type().String())
		return transport.UpdateSpanWithErr(span, err)
	}

	return transport.DispatchOnewayHandler(ctx, spec.Oneway(), req)
}

// Introspect returns the state of the inbound for introspection purposes.
func (i *Inbound) Introspect() introspection.InboundStatus {
	return introspection.InboundStatus{
		Transport: "redis",
		Endpoint: fmt.Sprintf("%s (queue: %s)",
			i.client.Endpoint(), i.queueKey),
		State: i.client.ConnectionState(),
	}
}
