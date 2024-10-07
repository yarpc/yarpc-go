// Copyright (c) 2024 Uber Technologies, Inc.
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

package transportinterceptor

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/yarpc/api/transport"
)

// TestNopUnaryOutbound ensures NopUnaryOutbound returns nil responses and no error.
func TestNopUnaryOutbound(t *testing.T) {
	outbound := NopUnaryOutbound

	resp, err := outbound.Call(context.Background(), &transport.Request{})
	assert.NoError(t, err)
	assert.Nil(t, resp)

	assert.False(t, outbound.IsRunning())
	assert.Nil(t, outbound.Transports())

	assert.NoError(t, outbound.Start())
	assert.NoError(t, outbound.Stop())
}

// TestNopOnewayOutbound ensures NopOnewayOutbound calls return nil acks and no error.
func TestNopOnewayOutbound(t *testing.T) {
	outbound := NopOnewayOutbound

	ack, err := outbound.CallOneway(context.Background(), &transport.Request{})
	assert.NoError(t, err)
	assert.Nil(t, ack)

	assert.False(t, outbound.IsRunning())
	assert.Nil(t, outbound.Transports())

	assert.NoError(t, outbound.Start())
	assert.NoError(t, outbound.Stop())
}

// TestNopStreamOutbound ensures NopStreamOutbound calls return nil responses and no error.
func TestNopStreamOutbound(t *testing.T) {
	outbound := NopStreamOutbound

	stream, err := outbound.CallStream(context.Background(), &transport.StreamRequest{})
	assert.NoError(t, err)
	assert.Nil(t, stream)

	assert.False(t, outbound.IsRunning())
	assert.Nil(t, outbound.Transports())

	assert.NoError(t, outbound.Start())
	assert.NoError(t, outbound.Stop())
}

// TestUnaryOutboundFunc tests if the function gets called correctly.
func TestUnaryOutboundFunc(t *testing.T) {
	called := false
	outbound := UnaryOutboundFunc(func(ctx context.Context, req *transport.Request) (*transport.Response, error) {
		called = true
		return &transport.Response{}, nil
	})

	resp, err := outbound.Call(context.Background(), &transport.Request{})
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.True(t, called)

	assert.NoError(t, outbound.Start())
	assert.NoError(t, outbound.Stop())
	assert.False(t, outbound.IsRunning())
	assert.Nil(t, outbound.Transports())
}

// TestOnewayOutboundFunc tests if the oneway function gets called correctly.
func TestOnewayOutboundFunc(t *testing.T) {
	called := false
	outbound := OnewayOutboundFunc(func(ctx context.Context, req *transport.Request) (transport.Ack, error) {
		called = true
		return nil, nil // Return nil since Ack is an interface
	})

	ack, err := outbound.CallOneway(context.Background(), &transport.Request{})
	assert.NoError(t, err)
	assert.Nil(t, ack)
	assert.True(t, called)

	assert.NoError(t, outbound.Start())
	assert.NoError(t, outbound.Stop())
	assert.False(t, outbound.IsRunning())
	assert.Nil(t, outbound.Transports())
}

// TestStreamOutboundFunc tests if the stream function gets called correctly.
func TestStreamOutboundFunc(t *testing.T) {
	called := false
	outbound := StreamOutboundFunc(func(ctx context.Context, req *transport.StreamRequest) (*transport.ClientStream, error) {
		called = true
		return &transport.ClientStream{}, nil
	})

	stream, err := outbound.CallStream(context.Background(), &transport.StreamRequest{})
	assert.NoError(t, err)
	assert.NotNil(t, stream)
	assert.True(t, called)

	assert.NoError(t, outbound.Start())
	assert.NoError(t, outbound.Stop())
	assert.False(t, outbound.IsRunning())
	assert.Nil(t, outbound.Transports())
}
