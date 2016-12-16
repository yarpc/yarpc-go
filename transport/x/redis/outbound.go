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
	"sync"
	"time"

	"go.uber.org/atomic"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/internal/errors"
	"go.uber.org/yarpc/serialize"

	"github.com/opentracing/opentracing-go"
)

var errOutboundNotStarted = errors.ErrOutboundNotStarted("redis.Outbound")

// Outbound is a redis OnewayOutbound that puts an RPC into the given queue key
type Outbound struct {
	client   Client
	tracer   opentracing.Tracer
	queueKey string

	started   atomic.Bool
	startOnce sync.Once
	startErr  error
	stopped   atomic.Bool
	stopOnce  sync.Once
	stopErr   error
}

// NewOnewayOutbound creates a redis Outbound that satisfies transport.OnewayOutbound
// queueKey - key for the queue in redis
func NewOnewayOutbound(client Client, queueKey string) *Outbound {
	return &Outbound{
		client:   client,
		tracer:   opentracing.GlobalTracer(),
		queueKey: queueKey,
	}
}

// Transports returns nil for now
func (o *Outbound) Transports() []transport.Transport {
	// TODO
	return nil
}

// WithTracer configures a tracer for the outbound
func (o *Outbound) WithTracer(tracer opentracing.Tracer) *Outbound {
	o.tracer = tracer
	return o
}

// Start creates connection to the redis instance
func (o *Outbound) Start() error {
	o.startOnce.Do(func() {
		o.started.Store(true)
		o.startErr = o.client.Start()
	})

	return o.startErr
}

// Stop stops the redis connection
func (o *Outbound) Stop() error {
	o.stopOnce.Do(func() {
		o.stopped.Store(true)
		o.stopErr = o.client.Stop()
	})

	return o.stopErr
}

// IsRunning returns whether the Outbound is running.
func (o *Outbound) IsRunning() bool {
	return o.started.Load() && !o.stopped.Load()
}

// CallOneway makes a oneway request using redis
func (o *Outbound) CallOneway(ctx context.Context, req *transport.Request) (transport.Ack, error) {
	if !o.started.Load() {
		return nil, errOutboundNotStarted
	}

	createOpenTracingSpan := transport.CreateOpenTracingSpan{
		Tracer:        o.tracer,
		TransportName: transportName,
		StartTime:     time.Now(),
	}
	ctx, span := createOpenTracingSpan.Do(ctx, req)
	defer span.Finish()

	marshalledRPC, err := serialize.ToBytes(o.tracer, span.Context(), req)
	if err != nil {
		return nil, transport.UpdateSpanWithErr(span, err)
	}

	err = o.client.LPush(o.queueKey, marshalledRPC)
	ack := time.Now()

	if err != nil {
		return nil, transport.UpdateSpanWithErr(span, err)
	}

	return ack, nil
}
