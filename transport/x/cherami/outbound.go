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

package cherami

import (
	"context"
	"time"

	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/internal/errors"
	"go.uber.org/yarpc/internal/sync"
	"go.uber.org/yarpc/serialize"
	"go.uber.org/yarpc/transport/x/cherami/internal"

	"github.com/opentracing/opentracing-go"
	"github.com/uber/cherami-client-go/client/cherami"
)

var errOutboundNotStarted = errors.ErrOutboundNotStarted("cherami.Outbound")

// OutboundConfig defines the config in order to create a Outbound.
type OutboundConfig struct {
	Destination string
}

// Outbound is a outbound that uses cherami as the transport
type Outbound struct {
	config        OutboundConfig
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

// NewOutbound builds a new cherami outbound
func (t *Transport) NewOutbound(config OutboundConfig) *Outbound {
	return &Outbound{
		config:        config,
		tracer:        t.tracer,
		client:        t.client,
		clientFactory: t.clientFactory,
	}
}

// Transports returns nil for now
func (o *Outbound) Transports() []transport.Transport {
	return nil
}

// IsRunning returns whether the outbound is still running.
func (o *Outbound) IsRunning() bool {
	return o.once.IsRunning()
}

// Start starts the outbound
func (o *Outbound) Start() error {
	return o.once.Start(o.start)
}

func (o *Outbound) start() error {
	var err error
	o.publisher, err = o.clientFactory.GetPublisher(o.client, o.config.Destination)
	return err
}

// Stop ends the connection to cherami
func (o *Outbound) Stop() error {
	return o.once.Stop(o.stop)
}

func (o *Outbound) stop() error {
	o.publisher.Close()
	return nil
}

// SetClientFactory sets a cherami client factory, used for testing
func (o *Outbound) SetClientFactory(factory internal.ClientFactory) {
	o.clientFactory = factory
}

// CallOneway makes a oneway request using cherami
func (o *Outbound) CallOneway(ctx context.Context, req *transport.Request) (transport.Ack, error) {
	if !o.IsRunning() {
		return nil, errOutboundNotStarted
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
