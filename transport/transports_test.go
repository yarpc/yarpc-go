// Copyright (c) 2020 Uber Technologies, Inc.
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

package transport_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/yarpc"
	"go.uber.org/yarpc/api/middleware"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/yarpctest"
)

func TestFirstOutboundMiddleware(t *testing.T) {
	// This ensures that the meta middleware is applied before all user specified
	// middleware. If the meta middleware is installed properly, user middleware
	// should see the name of the transport, instead of an empty string.

	const (
		transportName = "transport-name"
		serviceName   = "service-name"
	)

	newOutboundConfig := func(outboundMiddleware yarpc.OutboundMiddleware) *transport.OutboundConfig {
		outbound := yarpctest.NewFakeTransport().
			NewOutbound(nil, yarpctest.OutboundName(transportName))

		dispatcher := yarpc.NewDispatcher(yarpc.Config{
			Name: serviceName,
			Outbounds: yarpc.Outbounds{
				serviceName: {
					ServiceName: serviceName,
					Unary:       outbound,
					Oneway:      outbound,
					Stream:      outbound,
				},
			},
			OutboundMiddleware: outboundMiddleware,
		})
		return dispatcher.MustOutboundConfig(serviceName)
	}

	t.Run("unary", func(t *testing.T) {
		var gotTransportName string

		out := newOutboundConfig(yarpc.OutboundMiddleware{
			Unary: middleware.UnaryOutboundFunc(func(ctx context.Context, req *transport.Request, next transport.UnaryOutbound) (*transport.Response, error) {
				gotTransportName = req.Transport
				return next.Call(ctx, req)
			}),
		}).GetUnaryOutbound()

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		// fields required to satisfy validator outbound
		req := &transport.Request{
			Service:   "foo",
			Caller:    "foo",
			Procedure: "foo",
			Encoding:  transport.Encoding("foo"),
			Transport: "", // unset
		}

		_, _ = out.Call(ctx, req)
		assert.Equal(t, transportName, gotTransportName)
	})

	t.Run("oneway", func(t *testing.T) {
		var gotTransportName string

		out := newOutboundConfig(yarpc.OutboundMiddleware{
			Oneway: middleware.OnewayOutboundFunc(func(ctx context.Context, req *transport.Request, next transport.OnewayOutbound) (transport.Ack, error) {
				gotTransportName = req.Transport
				return next.CallOneway(ctx, req)
			}),
		}).GetOnewayOutbound()

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		// fields required to satisfy validator outbound
		req := &transport.Request{
			Service:   "foo",
			Caller:    "foo",
			Procedure: "foo",
			Encoding:  transport.Encoding("foo"),
			Transport: "", // unset
		}

		_, _ = out.CallOneway(ctx, req)
		assert.Equal(t, transportName, gotTransportName)
	})

	t.Run("stream", func(t *testing.T) {
		var gotTransportName string

		out := newOutboundConfig(yarpc.OutboundMiddleware{
			Stream: middleware.StreamOutboundFunc(func(ctx context.Context, req *transport.StreamRequest, next transport.StreamOutbound) (*transport.ClientStream, error) {
				gotTransportName = req.Meta.Transport
				return next.CallStream(ctx, req)
			}),
		}).Outbounds.Stream

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		// fields required to satisfy validator outbound
		req := &transport.StreamRequest{
			Meta: &transport.RequestMeta{
				Service:   "foo",
				Caller:    "foo",
				Procedure: "foo",
				Encoding:  transport.Encoding("foo"),
				Transport: "", // unset
			},
		}

		_, _ = out.CallStream(ctx, req)
		assert.Equal(t, transportName, gotTransportName)
	})
}
