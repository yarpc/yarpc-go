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

package circus

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc/api/peer"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/internal/testtime"
	"go.uber.org/yarpc/peer/hostport"
	"go.uber.org/yarpc/yarpctest"
)

func TestIsRunning(t *testing.T) {
	trans := yarpctest.NewFakeTransport()
	assert.True(t, New(trans, Seed(0)).IsRunning())
}

func TestWaitingForPeer(t *testing.T) {
	trans := yarpctest.NewFakeTransport(
		yarpctest.InitialConnectionStatus(peer.Unavailable),
	)
	list := New(trans)

	t.Run("NoDeadlineError", func(t *testing.T) {
		_, _, err := list.Choose(context.Background(), &transport.Request{})
		assert.Error(t, err)
	})

	t.Run("TimeoutError", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), testtime.Millisecond)
		defer cancel()

		_, _, err := list.Choose(ctx, &transport.Request{})
		assert.Error(t, err)
	})

	t.Run("WaitForPeer", func(t *testing.T) {
		go func() {
			time.Sleep(500 * testtime.Millisecond)
			list.Update(peer.ListUpdates{
				Additions: []peer.Identifier{
					hostport.Identify("1"),
				},
			})
			trans.SimulateConnect(hostport.Identify("1"))
		}()

		ctx, cancel := context.WithTimeout(context.Background(), testtime.Second)
		defer cancel()

		_, _, err := list.Choose(ctx, &transport.Request{})
		assert.NoError(t, err)
	})
}

func TestPeerRotation(t *testing.T) {
	trans := yarpctest.NewFakeTransport()
	list := New(trans, NoShuffle())

	const max = size - 4

	var ids []peer.Identifier
	for i := 0; i < max*2; i++ {
		ids = append(ids, hostport.Identify(strconv.Itoa(i)))
	}

	err := list.Update(peer.ListUpdates{
		Additions: ids,
	})
	require.NoError(t, err)

	req := &transport.Request{}
	ctx, cancel := context.WithTimeout(context.Background(), testtime.Second)
	defer cancel()

	for i := 0; i < max*2; i++ {
		peer, onFinish, err := list.Choose(ctx, req)
		require.NoError(t, err)
		require.Equal(t, strconv.Itoa(i%max), peer.Identifier())
		_, _ = peer, onFinish
	}
}
