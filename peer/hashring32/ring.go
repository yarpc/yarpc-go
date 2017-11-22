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
	"math"
	"math/rand"
	"sort"
	"strconv"
	"sync"

	"go.uber.org/yarpc/api/peer"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/yarpcerrors"
)

// HashFunc is the hash function used in ringpop hash ring.
type HashFunc func([]byte) uint32

// ring maintains the same ring topology as downstream,
// and apply a best effort selection to relay to the correct peer.
// When shard key not provided, a random peer would be returned.
type ring struct {
	serversByHash  map[uint32][]peer.StatusPeer
	servers        map[string]peer.StatusPeer
	hashRing       []uint32
	hashFunc       HashFunc
	replica        int
	m              sync.RWMutex
	errUnavailable error
}

type ringSubscriber struct{}

func (r ringSubscriber) NotifyStatusChanged(peer.Identifier) {}

var nopSubscriber = ringSubscriber{}

const (
	// Estimated number of hosts in a single downstream
	// This value is used to pre-allocate memory
	estimatedMaxNumHosts = 1500

	// Radix sort constants
	radix     = 16
	radixSize = 65536
	radixMask = 0xFFFF
	// for arrays shorter than min length, use regular sort
	radixMinLength = 128

	maxUINT32 = uint32(math.MaxUint32)
)

// newPeerRing a new consistent-hashing peer ring.
func newPeerRing(name string, hashFunc HashFunc, replica int) *ring {
	capacity := estimatedMaxNumHosts * replica
	return &ring{
		serversByHash:  make(map[uint32][]peer.StatusPeer, capacity),
		hashRing:       make([]uint32, 0, capacity),
		servers:        make(map[string]peer.StatusPeer, estimatedMaxNumHosts),
		hashFunc:       hashFunc,
		replica:        replica,
		errUnavailable: yarpcerrors.Newf(yarpcerrors.CodeUnavailable, "%s peer list fails to find a peer", name),
	}
}

// Choose a peer based on the hash of the shard key. Choose first peer in the
// ring that owns a replica to the right of the hash.
//
// A Choose on an empty population returns nil.
func (r *ring) Choose(ctx context.Context, req *transport.Request) peer.StatusPeer {
	return r.choose(req.ShardKey)
}

func (r *ring) choose(shardKey string) peer.StatusPeer {
	r.m.RLock()
	defer r.m.RUnlock()

	if len(r.servers) == 0 {
		return nil
	}

	var ix int
	if shardKey == "" {
		// Random index to get hash value
		ix = rand.Intn(len(r.hashRing))
	} else {
		// Binary search to find hash value
		key := r.hashFunc([]byte(shardKey))
		ix = indexOf(r.hashRing, key)
	}

	hash := r.hashRing[ix]
	servers := r.serversByHash[hash]

	return servers[0]
}

// Add an individual to the population.
// Returns a subscriber if the individual was absent, nil otherwise.
func (r *ring) Add(p peer.StatusPeer) peer.Subscriber {
	r.m.Lock()
	defer r.m.Unlock()

	individual := p.Identifier()

	if _, ok := r.servers[individual]; ok {
		return nil
	}
	r.addToServersMapAndHashRing(p)
	r.radixSortHashRing()
	return nopSubscriber
}

// Remove removes an individual from the population.
// Returns early if the individual is absent.
func (r *ring) Remove(p peer.StatusPeer, sub peer.Subscriber) {
	r.m.Lock()
	defer r.m.Unlock()

	individual := p.Identifier()

	if _, ok := r.servers[individual]; !ok {
		return
	}
	toBeRemovedIndexSet := make(map[int]struct{}, r.replica)
	r.removeFromServersMap(p, toBeRemovedIndexSet)
	r.removeFromHashRing(toBeRemovedIndexSet)
}

// Include extends the ring's population with new members. Think
// set union.
// The implementations can mutate the input.
func (r *ring) Include(pids []peer.StatusPeer) {
	r.m.Lock()
	defer r.m.Unlock()

	r.includeNoLock(pids)
}

func (r *ring) includeNoLock(pids []peer.StatusPeer) {
	for _, p := range pids {
		individual := p.Identifier()
		if _, ok := r.servers[individual]; ok {
			continue
		}
		r.addToServersMapAndHashRing(p)
	}
	r.radixSortHashRing()
}

// Exclude shrinks the ring's population by removing members. Think
// set difference.
// The implementations can mutate the input.
func (r *ring) Exclude(pids []peer.StatusPeer) {
	r.m.Lock()
	defer r.m.Unlock()

	r.excludeNoLock(pids)
}

func (r *ring) excludeNoLock(pids []peer.StatusPeer) {
	toBeRemovedIndexSet := make(map[int]struct{})

	for _, p := range pids {
		individual := p.Identifier()
		if _, ok := r.servers[individual]; !ok {
			continue
		}
		r.removeFromServersMap(p, toBeRemovedIndexSet)
	}
	r.removeFromHashRing(toBeRemovedIndexSet)
}

// Set replaces the ring's population with the new one.
// The implementations can mutate the input.
func (r *ring) Set(pids []peer.StatusPeer) {
	r.m.Lock()
	defer r.m.Unlock()

	// Create string set for population, to ensure it is addressable.
	population := make(map[string]peer.StatusPeer)
	for _, p := range pids {
		population[p.Identifier()] = p
	}

	// peers in old set but not in new set
	toBeRemoved := make([]peer.StatusPeer, 0, len(pids))
	for k, p := range r.servers {
		if _, ok := population[k]; !ok {
			toBeRemoved = append(toBeRemoved, p)
		}
	}
	r.excludeNoLock(toBeRemoved)

	// peers in new set but not in old set
	toBeAdded := make([]peer.StatusPeer, 0, len(pids))
	for k, p := range population {
		if _, ok := r.servers[k]; !ok {
			toBeAdded = append(toBeRemoved, p)
		}
	}
	r.includeNoLock(toBeAdded)
}

// Len returns the size of the ring population.
func (r *ring) Len() int {
	r.m.RLock()
	defer r.m.RUnlock()

	return len(r.servers)
}

// addToHashRing adds individual into both servers map and hash ring array,
// the array requires sort later.
func (r *ring) addToServersMapAndHashRing(p peer.StatusPeer) {
	individual := p.Identifier()

	r.servers[individual] = p
	hashes := hashes(individual, r.replica, r.hashFunc)
	r.hashRing = append(r.hashRing, hashes...)

	for _, hash := range hashes {
		r.serversByHash[hash] = append(r.serversByHash[hash], p)
	}
}

// removeFromServersMap removes server from map and
// adds it's hash ring array index to the to-be-removed set.
func (r *ring) removeFromServersMap(p peer.StatusPeer, toBeRemoved map[int]struct{}) {
	individual := p.Identifier()

	delete(r.servers, individual)

	hashes := hashes(individual, r.replica, r.hashFunc)

	for _, hash := range hashes {
		servers := r.serversByHash[hash]
		// Remove servers that have this ID
		var newServers []peer.StatusPeer
		for _, server := range servers {
			if server.Identifier() == individual {
				continue
			}
			newServers = append(newServers, server)
		}
		// When a hash value has no servers, mark as 'to be removed'
		// Cannot remove now because it is needed for binary search
		if len(newServers) == 0 {
			delete(r.serversByHash, hash)
			index := indexOf(r.hashRing, hash)
			// Remove hash from ring slice and also its duplicates (collisions)
			for i := index; i < len(r.hashRing); i++ {
				if r.hashRing[i] != hash {
					break
				}
				toBeRemoved[i] = struct{}{}
			}
		} else {
			r.serversByHash[hash] = newServers
		}
	}
}

// removeFromHashRing removes all peers in the given set from the ring.
func (r *ring) removeFromHashRing(toBeRemoved map[int]struct{}) {
	// Remove by setting values to max, sorting, and returning sub-array
	// Remove the tails instead of heads to avoid unnecessary memory allocations
	for index := range toBeRemoved {
		r.hashRing[index] = maxUINT32
	}
	r.radixSortHashRing()
	r.hashRing = r.hashRing[:len(r.hashRing)-len(toBeRemoved)]
}

// sortHashRing sorts the hash ring array.
func (r *ring) radixSortHashRing() {
	if len(r.hashRing) <= radixMinLength {
		sort.Slice(r.hashRing, func(i, j int) bool {
			return r.hashRing[i] < r.hashRing[j]
		})
		return
	}

	origin := r.hashRing
	swap := make([]uint32, len(r.hashRing))
	var key uint16
	var offset [radixSize]int

	for i := uint(0); i < 2; i++ {
		keyOffset := i * radix
		keyMask := uint32(radixMask << keyOffset)
		var counts [radixSize]int

		for _, h := range origin {
			key = uint16((h & keyMask) >> keyOffset)
			counts[key]++
		}

		offset[0] = 0
		for j := 1; j < radixSize; j++ {
			offset[j] = offset[j-1] + counts[j-1]
		}

		for _, h := range origin {
			key = uint16((h & keyMask) >> keyOffset)
			swap[offset[key]] = h
			offset[key]++
		}
		swap, origin = origin, swap
	}
}

// hashes creates replicas and their hash values.
func hashes(s string, replicas int, hashFunc HashFunc) []uint32 {
	r := make([]uint32, replicas)
	for i := 0; i < replicas; i++ {
		// This older replicaStr format will cause replica point collisions when there are
		// multiple instances running on the same host (e.g. on port 2100 and 21001).
		r[i] = hashFunc([]byte(s + strconv.Itoa(i)))
	}
	return r
}

// indexOf applies binary search to get the value in an array.
func indexOf(slice []uint32, v uint32) int {
	if len(slice) == 0 {
		return -1
	}
	// binary search
	index := sort.Search(len(slice),
		func(i int) bool { return slice[i] >= v })
	// greater than all elements, returns the first
	if index >= len(slice) {
		return 0
	}
	return index
}

func (r *ring) Start() error {
	return nil
}

func (r *ring) Stop() error {
	return nil
}

func (r *ring) IsRunning() bool {
	return true
}
