// Copyright (c) 2017 Uber Technologies, Inc.
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
	"hash/fnv"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/yarpc/api/peer"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/internal/testtime"
	"go.uber.org/yarpc/peer/hostport"
	"go.uber.org/yarpc/yarpctest"
)

const (
	ringID1 = hostport.PeerIdentifier("127.0.0.1:10000")
	ringID2 = hostport.PeerIdentifier("127.0.0.1:10001")
	ringID3 = hostport.PeerIdentifier("127.0.0.1:10002")
	ringID4 = hostport.PeerIdentifier("127.0.0.1:10004")
	ringID5 = hostport.PeerIdentifier("127.0.0.1:10005")

	key1 = "123"
)

func testHash(key []byte) uint32 {
	digest := fnv.New32()
	digest.Write(key)
	return digest.Sum32()
}

func newTestList() *List {
	t := yarpctest.NewFakeTransport()
	return New(t, "fnvring", testHash)
}

func TestRingAddRemove(t *testing.T) {
	r := newTestList()
	r.Start()
	defer r.Stop()

	b := r.Update(peer.ListUpdates{Additions: []peer.Identifier{ringID1}})
	b2 := r.Update(peer.ListUpdates{Additions: []peer.Identifier{ringID1}})

	p, _, err := r.Choose(context.Background(), &transport.Request{ShardKey: key1})
	assert.NoError(t, err, "Choose failed to select peer")
	assert.Nil(t, b, "Choose returned false but element did not exist.")
	assert.Error(t, b2, "Choose returned true but element already exists.")
	assert.Equal(t, ringID1.Identifier(), p.Identifier(), "aaa")
	assert.Equal(t, 1, r.Len(), "Size of members should be 1")

	r.Update(peer.ListUpdates{Removals: []peer.Identifier{ringID1}})
	r.Update(peer.ListUpdates{Removals: []peer.Identifier{ringID1, ringID1}})
	r.Update(peer.ListUpdates{Removals: []peer.Identifier{ringID1}})
	ctx, cancel := context.WithTimeout(context.Background(), 1*testtime.Millisecond)
	defer cancel()
	p, _, err = r.Choose(ctx, &transport.Request{ShardKey: key1})
	assert.Error(t, err, "Expected not to find any peer")
	assert.Equal(t, nil, p, "Expected to find nil peer")
	assert.Equal(t, 0, r.Len(), "Size of members should be 1")
}
