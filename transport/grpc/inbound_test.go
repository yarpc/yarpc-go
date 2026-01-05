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

package grpc

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInboundMechanics(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	inbound := NewTransport().NewInbound(listener)

	assert.False(t, inbound.IsRunning())
	assert.Equal(t, errRouterNotSet, inbound.Start())

	inbound = NewTransport().NewInbound(listener)
	inbound.SetRouter(newTestRouter(nil))
	assert.Nil(t, inbound.Addr())
	assert.NoError(t, inbound.Start())
	assert.True(t, inbound.IsRunning())
	assert.NotNil(t, inbound.Addr())
	assert.NoError(t, inbound.Stop())
	assert.Nil(t, inbound.Addr())
}

func TestInboundIntrospection(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	inbound := NewTransport().NewInbound(listener)
	inbound.SetRouter(newTestRouter(nil))

	assert.Equal(t, TransportName, inbound.Introspect().Transport, "unexpected transport name")
	assert.Equal(t, "Stopped", inbound.Introspect().State, "expected 'Stopped' state")
	assert.Empty(t, inbound.Introspect().Endpoint, "unexpected endpoint")

	require.NoError(t, inbound.Start())
	assert.Equal(t, "Started", inbound.Introspect().State, "expected 'Started' state")
	assert.NotEmpty(t, inbound.Introspect().Endpoint)
	assert.Equal(t, inbound.Addr().String(), inbound.Introspect().Endpoint, "unexpected endpoint")

	assert.NoError(t, inbound.Stop())
	assert.Equal(t, "Stopped", inbound.Introspect().State, "expected 'Stopped' state")
	assert.Empty(t, inbound.Introspect().Endpoint, "unexpected endpoint")
}
