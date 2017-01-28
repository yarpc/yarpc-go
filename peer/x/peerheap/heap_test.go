package peerheap

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPeerHeapEmpty(t *testing.T) {
	var ph peerHeap
	assert.Zero(t, ph.Len(), "New peer heap should be empty")
	popAndVerifyHeap(t, &ph)
}

func TestPeerHeapOrdering(t *testing.T) {
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
		var h peerHeap
		for _, ps := range tt {
			h.pushPeer(ps)
		}

		popped := popAndVerifyHeap(t, &h)
		assert.Equal(t, want, popped, "Unexpected ordering of peers")
	}
}

func TestPeerHeapUpdate(t *testing.T) {
	var h peerHeap
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
	h.update(p2.idx)

	popped := popAndVerifyHeap(t, &h)
	assert.Equal(t, []*peerScore{p3, p2}, popped, "Unexpected order after p2 update")
}

func TestPeerHeapDelete(t *testing.T) {
	const numPeers = 10

	var h peerHeap
	peers := make([]*peerScore, numPeers)
	for i := range peers {
		peers[i] = &peerScore{score: i}
		h.pushPeer(peers[i])
	}

	// The first peer is the lowest, remove it so it swaps with the last peer.
	h.delete(0)

	// Now when we pop a peer, we expect peers[1].
	got, ok := h.popPeer()
	require.True(t, ok, "Failed to get peer from non-empty peer heap")
	assert.Equal(t, peers[1], got, "Got incorrect peer after peer 0 was removed")
}

func TestPeerHeapValidate(t *testing.T) {
	var h peerHeap
	h.pushPeer(&peerScore{score: 1})

	for _, i := range []int{0, -1, 5} {
		ps := &peerScore{idx: i}
		assert.Error(t, h.validate(ps), "peer %v should not validate", ps)
	}
}

func popAndVerifyHeap(t *testing.T, h *peerHeap) []*peerScore {
	var popped []*peerScore

	lastScore := -1
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

func verifyIndexes(t *testing.T, h *peerHeap) {
	for i := range h.peers {
		assert.Equal(t, i, h.peers[i].idx, "wrong index for peer %v", h.peers[i])
	}
}
