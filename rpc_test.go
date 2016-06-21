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

package yarpc

import (
	"testing"

	"github.com/yarpc/yarpc-go/transport"
	"github.com/yarpc/yarpc-go/transport/http"
	tch "github.com/yarpc/yarpc-go/transport/tchannel"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uber/tchannel-go"
)

func basicRPC(t *testing.T) RPC {
	ch, err := tchannel.NewChannel("test", nil)
	require.NoError(t, err, "failed to create TChannel")

	return New(Config{
		Name: "test",
		Inbounds: []transport.Inbound{
			tch.NewInbound(ch, tch.ListenAddr(":0")),
			http.NewInbound(":0"),
		},
	})
}

func TestInboundsReturnsACopy(t *testing.T) {
	rpc := basicRPC(t)

	inbounds := rpc.Inbounds()
	require.Len(t, inbounds, 2, "expected two inbounds")
	assert.NotNil(t, inbounds[0], "must not be nil")
	assert.NotNil(t, inbounds[1], "must not be nil")

	// Mutate the list and verify that the next call still returns non-nil
	// results.
	inbounds[0] = nil
	inbounds[1] = nil

	inbounds = rpc.Inbounds()
	require.Len(t, inbounds, 2, "expected two inbounds")
	assert.NotNil(t, inbounds[0], "must not be nil")
	assert.NotNil(t, inbounds[1], "must not be nil")
}

func TestInboundsOrderIsMaintained(t *testing.T) {
	rpc := basicRPC(t)

	// Order must be maintained
	assert.Implements(t,
		(*tch.Inbound)(nil), rpc.Inbounds()[0], "first inbound must be TChannel")
	assert.Implements(t,
		(*http.Inbound)(nil), rpc.Inbounds()[1], "second inbound must be HTTP")
}

func TestInboundsOrderAfterStart(t *testing.T) {
	rpc := basicRPC(t)

	require.NoError(t, rpc.Start(), "failed to start RPC")
	defer rpc.Stop()

	inbounds := rpc.Inbounds()

	tchInbound := inbounds[0].(tch.Inbound)
	assert.NotEqual(t, "0.0.0.0:0", tchInbound.Channel().PeerInfo().HostPort)

	httpInbound := inbounds[1].(http.Inbound)
	assert.NotNil(t, httpInbound.Addr(), "expected an HTTP addr")
}
