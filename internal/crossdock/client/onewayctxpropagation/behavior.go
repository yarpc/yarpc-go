// Copyright (c) 2026 Uber Technologies, Inc.
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

package onewayctxpropagation

import (
	"context"
	"time"

	"github.com/crossdock/crossdock-go"
	opentracing "github.com/opentracing/opentracing-go"
	"go.uber.org/yarpc"
	"go.uber.org/yarpc/encoding/raw"
	"go.uber.org/yarpc/internal/crossdock/client/dispatcher"
)

// Run starts the behavior, testing oneway context propagation
func Run(t crossdock.T) {
	assert := crossdock.Assert(t)
	fatals := crossdock.Fatals(t)

	baggage := map[string]string{
		"hello": "world",
		"foo":   "bar",
	}

	// create handler
	callBackHandler, serverCalledBack := newCallBackHandler()
	dispatcher, callBackAddr := dispatcher.CreateOnewayDispatcher(t, callBackHandler)
	defer dispatcher.Stop()

	client := raw.New(dispatcher.ClientConfig("oneway-server"))

	// make call
	ctx, cancel := context.WithTimeout(newContextWithBaggage(baggage), time.Second)
	defer cancel()
	ack, err := client.CallOneway(
		ctx, "echo/raw", []byte{}, yarpc.WithHeader("callBackAddr", callBackAddr))

	fatals.NoError(err, "call to oneway/raw failed: %v", err)
	fatals.NotNil(ack, "ack is nil")

	// wait for server to call us back
	gotBaggage := <-serverCalledBack
	assert.Equal(baggage, gotBaggage, "server baggage: %s", gotBaggage)
}

// newCallBackHandler creates a oneway handler that fills a channel
// with the received body
func newCallBackHandler() (raw.OnewayHandler, <-chan map[string]string) {
	serverCalledBack := make(chan map[string]string)
	return func(ctx context.Context, body []byte) error {
		serverCalledBack <- extractBaggage(ctx)
		return nil
	}, serverCalledBack
}

func newContextWithBaggage(baggage map[string]string) context.Context {
	span := opentracing.GlobalTracer().StartSpan("add baggage")
	for k, v := range baggage {
		span.SetBaggageItem(k, v)
	}
	return opentracing.ContextWithSpan(context.Background(), span)
}

func extractBaggage(ctx context.Context) map[string]string {
	baggage := make(map[string]string)

	span := opentracing.SpanFromContext(ctx)
	if span == nil {
		return baggage
	}
	spanContext := span.Context()
	if spanContext == nil {
		return baggage
	}

	spanContext.ForeachBaggageItem(func(k, v string) bool {
		baggage[k] = v
		return true
	})

	return baggage
}
