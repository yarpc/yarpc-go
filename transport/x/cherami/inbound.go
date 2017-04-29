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
	"log"
	"time"

	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/internal/errors"
	"go.uber.org/yarpc/internal/sync"
	"go.uber.org/yarpc/serialize"
	"go.uber.org/yarpc/transport/x/cherami/internal"

	"github.com/opentracing/opentracing-go"
	"github.com/uber/cherami-client-go/client/cherami"
	"go.uber.org/multierr"
)

const (
	transportName = "cherami"

	defaultPrefetchCount = 10
)

// InboundOptions defines the config in order to create a Inbound.
//
// PrefetchCount controls the number of messages to buffer locally. Inbounds
// which process messages very fast may want to specify larger value for
// PrefetchCount for faster throughput. On the flip side larger values for
// PrefetchCount will result in more messages being buffered locally causing
// high memory footprint.
type InboundOptions struct {
	Destination   string
	ConsumerGroup string
	PrefetchCount int
}

// Inbound receives Oneway YARPC requests over Cherami.
type Inbound struct {
	transport     *Transport
	opts          InboundOptions
	consumer      cherami.Consumer
	router        transport.Router
	tracer        opentracing.Tracer
	client        cherami.Client
	clientFactory internal.ClientFactory

	once sync.LifecycleOnce
}

// NewInbound builds a new Cherami inbound.
func (t *Transport) NewInbound(opts InboundOptions) *Inbound {
	if opts.PrefetchCount == 0 {
		opts.PrefetchCount = defaultPrefetchCount
	}
	return &Inbound{
		once:          sync.Once(),
		transport:     t,
		opts:          opts,
		tracer:        t.tracer,
		client:        t.client,
		clientFactory: t.clientFactory,
	}
}

// Transports returns the transport that the inbound uses.
func (i *Inbound) Transports() []transport.Transport {
	return []transport.Transport{i.transport}
}

// SetRouter configures a router to handle incoming requests.
// This satisfies the transport.Inbound interface, and would be called
// by a dispatcher when it starts.
func (i *Inbound) SetRouter(router transport.Router) {
	i.router = router
}

// IsRunning returns whether the inbound is still processing requests.
func (i *Inbound) IsRunning() bool {
	return i.once.IsRunning()
}

// Start starts the inbound, reads and handle messages from Cherami.
func (i *Inbound) Start() error {
	return i.once.Start(i.start)
}

func (i *Inbound) start() error {
	if i.router == nil {
		return errors.ErrNoRouter
	}

	consumer, ch, err := i.clientFactory.GetConsumer(i.client, internal.ConsumerConfig{
		Destination:   i.opts.Destination,
		ConsumerGroup: i.opts.ConsumerGroup,
		PrefetchCount: i.opts.PrefetchCount,
	})
	if err != nil {
		return err
	}

	i.consumer = consumer

	go i.loop(ch)
	return nil
}

func (i *Inbound) loop(ch chan cherami.Delivery) {
	for delivery := range ch {
		// checksum verification before accessing message payload data
		if !delivery.VerifyChecksum() {
			log.Printf("checksum verification failed for ack_token: %s, asking for redelivery\n", delivery.GetDeliveryToken())
			if err := delivery.Nack(); err != nil {
				log.Printf("nack failed for ack_token: %s\n", delivery.GetDeliveryToken())
			}
			continue
		}

		msg := delivery.GetMessage()
		if err := i.handleMsg(msg.Payload.Data); err != nil {
			err = multierr.Append(err, delivery.Nack())
			log.Printf("handle message failure: %v\n", err)
			continue
		}

		if err := delivery.Ack(); err != nil {
			log.Printf("ack failed for ack_token: %s\n", delivery.GetDeliveryToken())
		}
	}
}

// Stop ends the connection to Cherami.
func (i *Inbound) Stop() error {
	return i.once.Stop(i.stop)
}

func (i *Inbound) stop() error {
	i.consumer.Close()
	return nil
}

// setClientFactory sets a cherami client factory, used for testing
func (i *Inbound) setClientFactory(factory internal.ClientFactory) {
	i.clientFactory = factory
}

func (i *Inbound) handleMsg(msg []byte) error {
	start := time.Now()
	spanContext, req, err := serialize.FromBytes(i.tracer, msg)
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
		err = errors.UnsupportedTypeError{Transport: transportName, Type: spec.Type().String()}
		return transport.UpdateSpanWithErr(span, err)
	}

	return transport.DispatchOnewayHandler(ctx, spec.Oneway(), req)
}
