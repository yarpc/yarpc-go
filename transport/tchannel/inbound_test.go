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

package tchannel

import (
	"testing"
	"time"

	"go.uber.org/yarpc/transport/transporttest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uber/tchannel-go"
	"github.com/uber/tchannel-go/json"
)

func TestInboundStartNew(t *testing.T) {
	tests := []struct {
		withInbound func(*tchannel.Channel, func(*Inbound))
	}{
		{
			func(ch *tchannel.Channel, f func(*Inbound)) {
				i := NewInboundFromChannel(ch)
				i.SetRegistry(new(transporttest.MockRegistry))
				// Can't do Equal because we want to match the pointer, not a
				// DeepEqual.
				assert.True(t, ch == i.Channel(), "channel does not match")
				require.NoError(t, i.Start())
				defer i.Stop()

				f(i)
			},
		},
		{
			func(ch *tchannel.Channel, f func(*Inbound)) {
				i := NewInboundFromChannel(ch).WithListenAddr(":0")
				i.SetRegistry(new(transporttest.MockRegistry))
				assert.True(t, ch == i.Channel(), "channel does not match")
				require.NoError(t, i.Start())
				defer i.Stop()

				f(i)
			},
		},
	}

	for _, tt := range tests {
		ch, err := tchannel.NewChannel("foo", nil)
		require.NoError(t, err)
		tt.withInbound(ch, func(i *Inbound) {
			assert.Equal(t, tchannel.ChannelListening, ch.State())
			assert.NoError(t, i.Stop())
			assert.Equal(t, tchannel.ChannelClosed, ch.State())
		})
	}
}

func TestInboundStartAlreadyListening(t *testing.T) {
	ch, err := tchannel.NewChannel("foo", nil)
	require.NoError(t, err)

	require.NoError(t, ch.ListenAndServe(":0"))
	assert.Equal(t, tchannel.ChannelListening, ch.State())

	i := NewInboundFromChannel(ch)

	i.SetRegistry(new(transporttest.MockRegistry))
	require.NoError(t, i.Start())
	defer i.Stop()

	assert.NoError(t, i.Stop())
	assert.Equal(t, tchannel.ChannelClosed, ch.State())
}

func TestInboundStopWithoutStarting(t *testing.T) {
	ch, err := tchannel.NewChannel("foo", nil)
	require.NoError(t, err)

	i := NewInboundFromChannel(ch)
	assert.NoError(t, i.Stop())
}

func TestInboundInvalidAddress(t *testing.T) {
	ch, err := tchannel.NewChannel("foo", nil)
	require.NoError(t, err)
	i := NewInboundFromChannel(ch).WithListenAddr("not valid")
	i.SetRegistry(new(transporttest.MockRegistry))
	assert.Error(t, i.Start())
}

func TestInboundExistingMethods(t *testing.T) {
	// Create a channel with an existing "echo" method.
	ch, err := tchannel.NewChannel("foo", nil)
	require.NoError(t, err)
	json.Register(ch, json.Handlers{
		"echo": func(ctx json.Context, req map[string]string) (map[string]string, error) {
			return req, nil
		},
	}, nil)

	i := NewInboundFromChannel(ch)
	i.SetRegistry(new(transporttest.MockRegistry))
	require.NoError(t, i.Start())
	defer i.Stop()

	// Make a call to the "echo" method which should call our pre-registered method.
	ctx, cancel := json.NewContext(time.Second)
	defer cancel()

	var resp map[string]string
	arg := map[string]string{"k": "v"}

	svc := ch.ServiceName()
	peer := ch.Peers().GetOrAdd(ch.PeerInfo().HostPort)
	err = json.CallPeer(ctx, peer, svc, "echo", arg, &resp)
	require.NoError(t, err, "Call failed")
	assert.Equal(t, arg, resp, "Response mismatch")
}
