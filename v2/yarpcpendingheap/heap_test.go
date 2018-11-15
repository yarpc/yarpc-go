// Copyright (c) 2018 Uber Technologies, Inc.
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

package yarpcpendingheap

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPeerHeapEmpty(t *testing.T) {
	var ph pendingHeap
	assert.Zero(t, ph.Len(), "New peer heap should be empty")
	popAndVerifyHeap(t, &ph)
}

func TestRoundRobinHeapOrdering(t *testing.T) {
	p1 := &peerScore{score: 1}
	p2 := &peerScore{score: 2}
	p3 := &peerScore{score: 3}

	// same score as p3, but always pushed after p3, so it will be returned last.
	p4 := &peerScore{score: 3}

	want := []*peerScore{p1, p2, p3, p4}
	tests := [][]*peerScore{
		{p1, p2, p3, p4},
		{p3, p4, p2, p1},
		{p3, p1, p2, p4},
	}

	for _, tt := range tests {
		h := pendingHeap{
			nextRand: nextRand(0), /* irrelevant since we're not doing random insertions*/
		}
		for _, ps := range tt {
			h.pushPeer(ps)
		}

		popped := popAndVerifyHeap(t, &h)
		assert.Equal(t, want, popped, "Unexpected ordering of peers")
	}
}

func TestPeerHeapInsertionOrdering(t *testing.T) {
	p1 := &peerScore{score: 1}
	p2 := &peerScore{score: 2}
	p3 := &peerScore{score: 3}
	p4 := &peerScore{score: 3} // same score as p3

	tests := []struct {
		name          string
		give          []*peerScore
		nextRandIndex int
		insert        *peerScore
		want          []*peerScore
	}{
		{
			name:   "p3.last < p4.last",
			give:   []*peerScore{p1, p2, p3},
			insert: p4,
			// no swap since nextRandIndex+1 == len(list)
			nextRandIndex: 3,
			want:          []*peerScore{p1, p2, p3, p4},
			// p1.last = 1
			// p2.last = 2
			// p3.last = 3
			// p4.last = 4
		},
		{
			name:          "p4.last < p3.last",
			give:          []*peerScore{p1, p2, p3},
			insert:        p4,
			nextRandIndex: 0, // swap p1.last
			want:          []*peerScore{p1, p2, p4, p3},
			// p1.last = 4
			// p2.last = 2
			// p4.last = 1
			// p3.last = 3
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := pendingHeap{nextRand: nextRandFromSlice([]int{tt.nextRandIndex})}
			// prepare list
			for _, ps := range tt.give {
				h.pushPeer(ps)
			}

			// new peer added
			h.pushPeerRandom(tt.insert)

			popped := popAndVerifyHeap(t, &h)
			assert.Equal(t, tt.want, popped, "Unexpected ordering of peers")
		})
	}
}

func TestPeerHeapUpdate(t *testing.T) {
	h := pendingHeap{nextRand: nextRand(0)}
	p1 := &peerScore{score: 1}
	p2 := &peerScore{score: 2}
	p3 := &peerScore{score: 3}

	h.pushPeer(p3)
	h.pushPeer(p1)
	h.pushPeer(p2)

	ps, ok := h.popPeer()
	require.True(t, ok, "pop with non-empty heap should succeed")
	assert.Equal(t, p1, ps, "Wrong peer")

	// Now update p2's score to be higher than p3.
	p2.score = 10
	h.update(p2.index)

	popped := popAndVerifyHeap(t, &h)
	assert.Equal(t, []*peerScore{p3, p2}, popped, "Unexpected order after p2 update")
}

func TestPeerHeapDelete(t *testing.T) {
	const numPeers = 10

	h := pendingHeap{nextRand: nextRand(0)}
	peers := make([]*peerScore, numPeers)
	for i := range peers {
		peers[i] = &peerScore{score: int64(i)}
		h.pushPeer(peers[i])
	}

	// The first peer is the lowest, remove it so it swaps with the last peer.
	h.delete(0)

	// Now when we pop peers, we expect peers 1 to N.
	want := peers[1:]
	popped := popAndVerifyHeap(t, &h)
	assert.Equal(t, want, popped, "Unexpected peers after delete peer 0")
}

func (ph *pendingHeap) validate(ps *peerScore) error {
	if ps.index < 0 || ps.index >= ph.Len() || ph.peers[ps.index] != ps {
		return fmt.Errorf("pendingHeap bug: %+v has bad index %v (len %v)", ps, ps.index, ph.Len())
	}
	return nil
}

func TestPeerHeapValidate(t *testing.T) {
	h := pendingHeap{nextRand: nextRand(0)}
	ps := &peerScore{score: 1}
	h.pushPeer(ps)
	assert.Nil(t, h.validate(ps), "peer %v should validate", ps)

	for _, i := range []int{0, -1, 5} {
		ps := &peerScore{index: i}
		assert.Error(t, h.validate(ps), "peer %v should not validate", ps)
	}
}

func popAndVerifyHeap(t *testing.T, h *pendingHeap) []*peerScore {
	var popped []*peerScore

	lastScore := int64(-1)
	for h.Len() > 0 {
		verifyIndexes(t, h)

		ps, ok := h.popPeer()
		require.True(t, ok, "pop with non-empty heap should succeed")
		popped = append(popped, ps)

		if lastScore == -1 {
			lastScore = ps.score
			continue
		}

		if ps.score < lastScore {
			t.Fatalf("heap returned peer %v with lower score than %v", ps, lastScore)
		}
		lastScore = ps.score
	}

	_, ok := h.popPeer()
	require.False(t, ok, "Expected no peers to be returned with empty list")
	return popped
}

func verifyIndexes(t *testing.T, h *pendingHeap) {
	for i := range h.peers {
		assert.Equal(t, i, h.peers[i].index, "wrong index for peer %v", h.peers[i])
	}
}

func TestPeerHeapInvalidAdd(t *testing.T) {
	var ph pendingHeap
	assert.Nil(t, (&ph).Add(nil, nil), "heap does not panic when adding nil")
}

func TestPeerHeapInvalidRemoval(t *testing.T) {
	var ph pendingHeap
	(&ph).Remove(nil, nil, nil)
}
