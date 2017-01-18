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
	"log"
	"time"

	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/internal/errors"
	"go.uber.org/yarpc/internal/sync"
	"go.uber.org/yarpc/serialize"

	"github.com/opentracing/opentracing-go"
	"github.com/uber/cherami-client-go/client/cherami"
)

const (
	transportName = "cherami"

	defaultPrefetchCount = 10

	defaultCheramiTimeoutInSec = 15
)

// InboundConfig defines the config in order to create a Inbound
// if frontend is provided, we'll connect to the provided frontend
// otherwise, hyperbahn will be used to connect to cherami
// PrefetchCount controls the number of messages to buffer locally.
// Inbounds which process messages very fast may want to specify larger value
// for PrefetchCount for faster throughput.  On the flip side larger values for
// PrefetchCount will result in more messages being buffered locally causing high memory foot print

type InboundConfig struct {
	Destination         string
	ConsumerGroup       string
	Frontend            string
	Port                int
	PrefetchCount       int
	CheramiTimeoutInSec int
}

// Inbound is a inbound that uses cherami as the transport
type Inbound struct {
	config         InboundConfig
	consumer       cherami.Consumer
	router         transport.Router
	tracer         opentracing.Tracer
	cheramiFactory CheramiFactory

	once sync.LifecycleOnce
}

// NewInbound builds a new cherami inbound
func NewInbound(config InboundConfig) *Inbound {
	if config.PrefetchCount == 0 {
		config.PrefetchCount = defaultPrefetchCount
	}
	if config.CheramiTimeoutInSec == 0 {
		config.CheramiTimeoutInSec = defaultCheramiTimeoutInSec
	}
	return &Inbound{
		config:         config,
		tracer:         opentracing.GlobalTracer(),
		cheramiFactory: NewCheramiFactory(),
	}
}

// Transports returns nil for now
func (i *Inbound) Transports() []transport.Transport {
	return nil
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

// Start starts the inbound, reading and handling messages from cherami
func (i *Inbound) Start() error {
	return i.once.Start(i.start)
}

func (i *Inbound) start() error {
	if i.router == nil {
		return errors.ErrNoRouter
	}

	var client cherami.Client
	var err error
	if len(i.config.Frontend) > 0 {
		client, err = i.cheramiFactory.GetClientWithFrontEnd(i.config.Frontend, i.config.Port)
	} else {
		client, err = i.cheramiFactory.GetClientWithHyperbahn()
	}
	if err != nil {
		return err
	}

	consumer, ch, err := i.cheramiFactory.GetConsumer(client, i.config.Destination, i.config.ConsumerGroup, i.config.PrefetchCount, i.config.CheramiTimeoutInSec)
	if err != nil {
		return err
	}

	i.consumer = consumer

	go func() {
		for delivery := range ch {
			// checksum verification before accessing message payload data
			if !delivery.VerifyChecksum() {
				log.Printf("checksum verification failed for ack_token: %s, asking for redelivery\n", delivery.GetDeliveryToken())
				delivery.Nack()
				continue
			}

			msg := delivery.GetMessage()

			if err = i.handleMsg(msg.Payload.Data); err == nil {
				if err = delivery.Ack(); err != nil {
					log.Printf("ack failed for ack_token: %s\n", delivery.GetDeliveryToken())
				}
			} else {
				if err = delivery.Nack(); err != nil {
					log.Printf("nack failed for ack_token: %s\n", delivery.GetDeliveryToken())
				}
			}
		}
	}()
	return nil
}

// Stop ends the connection to cherami
func (i *Inbound) Stop() error {
	return i.once.Stop(i.stop)
}

func (i *Inbound) stop() error {
	i.consumer.Close()
	return nil
}

// SetCheramiFactory sets a cherami factory, used for testing
func (i *Inbound) SetCheramiFactory(factory CheramiFactory) {
	i.cheramiFactory = factory
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
		err = errors.UnsupportedTypeError{Transport: transportName, Type: string(spec.Type())}
		return transport.UpdateSpanWithErr(span, err)
	}

	return transport.DispatchOnewayHandler(ctx, spec.Oneway(), req)
}
