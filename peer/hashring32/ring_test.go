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
	"fmt"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/yarpc/api/peer"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/peer/hostport"
	"go.uber.org/yarpc/yarpctest"
)

var (
	t         = yarpctest.NewFakeTransport()
	ringPeer1 = hostport.NewPeer("127.0.0.1:10000", t)
	ringPeer2 = hostport.NewPeer("127.0.0.1:10001", t)
	ringPeer3 = hostport.NewPeer("127.0.0.1:10002", t)
	ringPeer4 = hostport.NewPeer("127.0.0.1:10004", t)
	ringPeer5 = hostport.NewPeer("127.0.0.1:10005", t)
)

func newTestRing() *ring {
	return newPeerRing("fnv-hashring", testHash, 100)
}

func TestAddRemove(t *testing.T) {
	r := newTestRing()

	s := r.Add(ringPeer1)
	assert.NotNil(t, s, "Add returned a subscriber")
	s2 := r.Add(ringPeer1)
	assert.Nil(t, s2, "Add returned nil on redundant peer")

	p := r.Choose(context.Background(), &transport.Request{ShardKey: key1})
	assert.NotNil(t, p, "Choose failed to select peer")
	assert.Equal(t, ringPeer1.Identifier(), p.Identifier(), "chose the single peer")
	assert.Equal(t, 1, r.Len(), "Size of members should be 1")

	r.Remove(ringPeer1, s)
	assert.Equal(t, 0, r.Len(), "Size of members should be 0")
}

// TestMultipleChoose tests whether peer chooser
// always selects the same peer when ring is at the same topology.
func TestMultipleChoose(t *testing.T) {
	r := newTestRing()
	r.Add(ringPeer1)
	r.Add(ringPeer2)
	r.Add(ringPeer3)

	assert.Equal(t, 3, r.Len(), "Size of members should be 3")
	p := r.choose(key1)
	assert.NotEqual(t, "", p, "Choose failed to select a peer")
	p2 := r.choose(key1)
	assert.NotEqual(t, "", p2, "Choose failed to select a peer")
	assert.Equal(t, p, p2, "Choose selected a different peer")
	p3 := r.choose(key1)
	assert.NotEqual(t, "", p3, "Choose failed to select a peer")
	assert.Equal(t, p, p3, "Choose selected a different peer")

	s4 := r.Add(ringPeer4)
	s5 := r.Add(ringPeer5)
	r.Remove(ringPeer4, s4)
	r.Remove(ringPeer5, s5)

	p4 := r.choose(key1)
	assert.NotEqual(t, "", p4, "Choose failed to select a peer")
	p5 := r.choose(key1)
	assert.Equal(t, p, p4, "Choose selected a different peer")
	assert.NotEqual(t, "", p5, "Choose failed to select a peer")
	assert.Equal(t, p, p5, "Choose selected a different peer")
}

func TestNoShardKey(t *testing.T) {
	r := newTestRing()
	r.Add(ringPeer1)
	p := r.choose("")
	assert.NotEqual(t, "", p, "Choose failed to select a peer")
	assert.Equal(t, ringPeer1, p, "Should returns a random individual.")
}

func TestNoPeer(t *testing.T) {
	r := newTestRing()
	p := r.choose("")
	assert.Nil(t, p, "Should return nil when no peer can be selected.")

	p2 := r.choose(key1)
	assert.Nil(t, p2, "Should return nil when no peer can be selected.")
}

func TestRingIncludeExclude(t *testing.T) {
	r := newTestRing()
	servers := []peer.StatusPeer{
		ringPeer1,
		ringPeer2,
		ringPeer3,
		ringPeer4,
		ringPeer5,
	}
	r.Include(servers)
	assert.Equal(t, 5, r.Len(), "Load balancer pool size should be 5.")
	r.Include(servers)
	assert.Equal(t, 5, r.Len(), "Load balancer pool size should be 5.")

	toBeRemoved := []peer.StatusPeer{
		ringPeer2,
		ringPeer3,
		ringPeer4,
	}
	r.Exclude(toBeRemoved)
	assert.Equal(t, 2, r.Len(), "Load balancer pool size should be 2")
	r.Exclude(toBeRemoved)
	assert.Equal(t, 2, r.Len(), "Load balancer pool size should be 2")
}

// TODO TestHashCollision

func TestIndexOf(t *testing.T) {
	var empty = []uint32{}
	assert.Equal(t, -1, indexOf(empty, uint32(0)), "Should return -1 for empty array")

	var slice = []uint32{0, 1, 2, 3, 4}
	assert.Equal(t, 0, indexOf(slice, uint32(0)), "Should return index 0")
	assert.Equal(t, 3, indexOf(slice, uint32(3)), "Should return index 3")
	assert.Equal(t, 4, indexOf(slice, uint32(4)), "Should return index 4")
	assert.Equal(t, 0, indexOf(slice, uint32(5)), "Should return index 0")
}

// Benchmarks

func BenchmarkRingAdd(b *testing.B) {
	servers := generatePeers(b.N)

	r := newTestRing()
	b.ResetTimer()
	for _, server := range servers {
		r.Add(server)
	}
	b.ReportAllocs()
}

func BenchmarkRingRemove(b *testing.B) {
	count := b.N
	servers := generatePeers(count)

	r := newTestRing()
	r.Include(servers)

	b.ResetTimer()
	for _, server := range servers {
		r.Remove(server, nopSubscriber)
	}
	b.ReportAllocs()
}

func BenchmarkRingInclude(b *testing.B) {
	count := b.N
	servers := generatePeers(count)

	r := newTestRing()

	b.ResetTimer()
	r.Include(servers)
	b.ReportAllocs()
}

func BenchmarkRingExclude(b *testing.B) {
	count := b.N
	servers := generatePeers(count)

	r := newTestRing()
	r.Include(servers)
	b.ResetTimer()

	r.Exclude(servers)
	b.ReportAllocs()
}

func BenchmarkRingSet(b *testing.B) {
	count := b.N
	oldServers := generatePeers(count)
	newServers := newPopulation(count/3, oldServers)

	r := newTestRing()
	r.Include(oldServers)
	b.ResetTimer()
	r.Set(newServers)
	b.ReportAllocs()
}

// newPopulation updates numUpdates individuals in the oldPopulation,
// and keeps the size unchanged.
func newPopulation(numUpdates int, oldPopulation []peer.StatusPeer) []peer.StatusPeer {
	count := len(oldPopulation)

	sameHostPortsCount := count - numUpdates
	newPopulation := make([]peer.StatusPeer, 0, count)
	hostPorts := make(map[string]struct{})

	for _, p := range oldPopulation {
		if len(newPopulation) == sameHostPortsCount {
			break
		}
		hostPorts[p.Identifier()] = struct{}{}
		newPopulation = append(newPopulation, p)
	}

	// Generate new hostPorts so that size doesn't change
	for {
		if len(newPopulation) == count {
			break
		}
		hostPort := generateRandomHostPort()
		if _, ok := hostPorts[hostPort]; ok {
			continue
		}
		newPopulation = append(newPopulation, hostport.NewPeer(hostport.PeerIdentifier(hostPort), t))
		hostPorts[hostPort] = struct{}{}
	}
	return newPopulation
}

// Helpers

const ipUpperBound = 256

func generateRandomHostPort() string {
	// Make port number exact 5 digits so no replica collisions
	return fmt.Sprintf("%d.%d.%d.%d:%d",
		rand.Intn(ipUpperBound),
		rand.Intn(ipUpperBound),
		rand.Intn(ipUpperBound),
		rand.Intn(ipUpperBound),
		rand.Intn(55536)+10000, // interval [10000, 65536)
	)
}

func generatePeers(count int) []peer.StatusPeer {
	peersSet := make(map[string]struct{}, count)
	peers := make([]peer.StatusPeer, 0, count)
	for i := 0; i < count; i++ {
		for {
			hostPort := generateRandomHostPort()
			if _, ok := peersSet[hostPort]; ok {
				continue
			}
			peersSet[hostPort] = struct{}{}
			peers = append(peers, hostport.NewPeer(hostport.PeerIdentifier(hostPort), t))
			break
		}
	}
	return peers
}
