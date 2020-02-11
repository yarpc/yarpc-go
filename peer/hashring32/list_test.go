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

package hashring32

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc/api/peer"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/peer/hashring32/internal/farmhashring"
	"go.uber.org/yarpc/yarpctest"
	"go.uber.org/zap/zaptest"
)

func TestAddRemoveAndChoose(t *testing.T) {
	trans := yarpctest.NewFakeTransport(yarpctest.InitialConnectionStatus(peer.Available))
	pl := New(
		trans,
		farmhashring.Fingerprint32,
		OffsetHeader("test"),
		PeerOverrideHeader("poTest"),
		ReplicaDelimiter("#"),
		Logger(zaptest.NewLogger(t)),
	)

	pl.Start()

	pl.Update(
		peer.ListUpdates{
			Additions: []peer.Identifier{
				&FakeShardIdentifier{id: "id1", shard: "shard-1"},
				&FakeShardIdentifier{id: "id2", shard: "shard-2"},
			},
		},
	)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	r, _, err := pl.Choose(ctx, &transport.Request{ShardKey: "foo1"})
	require.NoError(t, err)
	assert.Equal(t, "id1", r.Identifier())

	r, _, err = pl.Choose(ctx, &transport.Request{ShardKey: "foo2"})
	require.NoError(t, err)
	assert.Equal(t, "id2", r.Identifier())

	pl.Update(
		peer.ListUpdates{
			Removals: []peer.Identifier{
				&FakeShardIdentifier{id: "id2", shard: "shard2"},
			},
		},
	)

	r, _, _ = pl.Choose(ctx, &transport.Request{ShardKey: "foo1"})
	assert.Equal(t, "id1", r.Identifier())

	r, _, _ = pl.Choose(ctx, &transport.Request{ShardKey: "foo2"})
	assert.Equal(t, "id1", r.Identifier())

}

func TestOverrideChooseAndRemoveOverrideChoose(t *testing.T) {
	var headers transport.Headers
	trans := yarpctest.NewFakeTransport(yarpctest.InitialConnectionStatus(peer.Available))
	pl := New(
		trans,
		farmhashring.Fingerprint32,
		OffsetHeader("test"),
		PeerOverrideHeader("poTest"),
		ReplicaDelimiter("#"),
	)

	pl.Start()
	t.Log("started")

	pl.Update(
		peer.ListUpdates{
			Additions: []peer.Identifier{
				&FakeShardIdentifier{id: "id1", shard: "shard-1"},
				&FakeShardIdentifier{id: "id2", shard: "shard-2"},
			},
		},
	)
	t.Log("updated")

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// Test overriden by header.
	headers = headers.With("poTest", "shard-2")
	r, _, _ := pl.Choose(ctx, &transport.Request{ShardKey: "foo1", Headers: headers})
	assert.Equal(t, "id2", r.Identifier())
	t.Log("chose once")

	// Test invalid override header.
	headers = headers.With("poTest", "shard-3")
	r, _, _ = pl.Choose(ctx, &transport.Request{ShardKey: "foo1", Headers: headers})
	assert.Equal(t, "id1", r.Identifier())
	t.Log("chose twice")

	pl.Update(
		peer.ListUpdates{
			Removals: []peer.Identifier{
				&FakeShardIdentifier{id: "id2", shard: "shard2"},
			},
		},
	)
	t.Log("updated again")

	// Test removed key in override header.
	headers = headers.With("poTest", "shard-2")
	r, _, _ = pl.Choose(ctx, &transport.Request{ShardKey: "foo2", Headers: headers})
	assert.Equal(t, "id1", r.Identifier())
	t.Log("chose a third time")

}
