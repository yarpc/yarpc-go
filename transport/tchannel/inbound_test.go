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

	"github.com/yarpc/yarpc-go/transport"
	"github.com/yarpc/yarpc-go/transport/transporttest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uber/tchannel-go"
)

func TestInboundStartNew(t *testing.T) {
	tests := []struct {
		withInbound func(*tchannel.Channel, func(transport.Inbound))
	}{
		{
			func(ch *tchannel.Channel, f func(transport.Inbound)) {
				i := NewInbound(ch)
				require.NoError(t, i.Start(new(transporttest.MockHandler)))
				defer i.Stop()

				f(i)
			},
		},
		{
			func(ch *tchannel.Channel, f func(transport.Inbound)) {
				i := NewInbound(ch, ListenAddr(":0"))
				require.NoError(t, i.Start(new(transporttest.MockHandler)))
				defer i.Stop()

				f(i)
			},
		},
	}

	for _, tt := range tests {
		ch, err := tchannel.NewChannel("foo", nil)
		require.NoError(t, err)
		tt.withInbound(ch, func(i transport.Inbound) {
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

	i := NewInbound(ch)

	require.NoError(t, i.Start(new(transporttest.MockHandler)))
	defer i.Stop()

	assert.NoError(t, i.Stop())
	assert.Equal(t, tchannel.ChannelClosed, ch.State())
}

func TestInboundStopWithoutStarting(t *testing.T) {
	ch, err := tchannel.NewChannel("foo", nil)
	require.NoError(t, err)

	i := NewInbound(ch)
	assert.NoError(t, i.Stop())
}

func TestInboundInvalidAddress(t *testing.T) {
	ch, err := tchannel.NewChannel("foo", nil)
	require.NoError(t, err)
	i := NewInbound(ch, ListenAddr("not valid"))
	assert.Error(t, i.Start(new(transporttest.MockHandler)))
}
