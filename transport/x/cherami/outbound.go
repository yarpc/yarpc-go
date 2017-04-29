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

package cherami

import (
	"context"
	"time"

	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/internal/sync"
	"go.uber.org/yarpc/serialize"
	"go.uber.org/yarpc/transport/x/cherami/internal"

	"github.com/opentracing/opentracing-go"
	"github.com/uber/cherami-client-go/client/cherami"
)

// OutboundOptions specifies a Cherami outbound.
type OutboundOptions struct {
	Destination string
}

// Outbound is a outbound that uses Cherami as the transport.
type Outbound struct {
	transport     *Transport
	opts          OutboundOptions
	publisher     cherami.Publisher
	tracer        opentracing.Tracer
	client        cherami.Client
	clientFactory internal.ClientFactory

	once sync.LifecycleOnce
}

type receipt struct{ cherami.PublisherReceipt }

func (r receipt) String() string {
	return r.Receipt
}

// NewOutbound builds a new cherami outbound.
func (t *Transport) NewOutbound(opts OutboundOptions) *Outbound {
	return &Outbound{
		once:          sync.Once(),
		transport:     t,
		opts:          opts,
		tracer:        t.tracer,
		client:        t.client,
		clientFactory: t.clientFactory,
	}
}

// Transports returns the transport that the outbound uses.
func (o *Outbound) Transports() []transport.Transport {
	return []transport.Transport{o.transport}
}

// IsRunning returns whether the outbound is still running.
func (o *Outbound) IsRunning() bool {
	return o.once.IsRunning()
}

// Start starts the outbound.
func (o *Outbound) Start() error {
	return o.once.Start(o.start)
}

func (o *Outbound) start() error {
	var err error
	o.publisher, err = o.clientFactory.GetPublisher(o.client, o.opts.Destination)
	return err
}

// Stop ends the connection to Cherami.
func (o *Outbound) Stop() error {
	return o.once.Stop(o.stop)
}

func (o *Outbound) stop() error {
	o.publisher.Close()
	return nil
}

// setClientFactory sets a cherami client factory, used for testing.
func (o *Outbound) setClientFactory(factory internal.ClientFactory) {
	o.clientFactory = factory
}

// CallOneway makes a oneway request using Cherami.
func (o *Outbound) CallOneway(ctx context.Context, req *transport.Request) (transport.Ack, error) {
	if err := o.once.WhenRunning(ctx); err != nil {
		return nil, err
	}

	createOpenTracingSpan := transport.CreateOpenTracingSpan{
		Tracer:        o.tracer,
		TransportName: transportName,
		StartTime:     time.Now(),
	}
	_, span := createOpenTracingSpan.Do(ctx, req)
	defer span.Finish()

	marshalledRPC, err := serialize.ToBytes(o.tracer, span.Context(), req)
	if err != nil {
		return nil, transport.UpdateSpanWithErr(span, err)
	}

	msg := &cherami.PublisherMessage{Data: marshalledRPC}
	r := o.publisher.Publish(msg)

	if r.Error != nil {
		return nil, transport.UpdateSpanWithErr(span, r.Error)
	}

	return receipt{*r}, nil
}
