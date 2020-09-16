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
	"errors"
	"math"
	"math/rand"
	"sort"
	"strconv"
	"sync"

	"go.uber.org/yarpc/peer/hashring32/internal/radixsort32"
)

const (
	defaultNumReplicas      = 100
	defaultNumPeersEstimate = 1500
)

var (
	// ErrEmptyPool is thrown when no availabel peer in pool.
	ErrEmptyPool = errors.New("empty pool")
	// ErrNegativeN is thrown when Shard.N is negative.
	ErrNegativeN = errors.New("negative n")
	// ErrUnexpected is thrown when a invalid code branch is reached and should never happen.
	ErrUnexpected = errors.New("unexpected error")
)

// Hashring32 will implement Choose, Add, Remove, Include, Exclude, Set, and Len.
type Hashring32 struct {
	hash             HashFunc32
	numReplicas      int
	numPeersEstimate int
	formatReplica    ReplicaFormatterFunc
	sorter           *radixsort32.RadixSorter32

	m sync.RWMutex

	// Hashes in hashesArray and membersMapByHash should always be synced.
	hashesArray      []uint32
	membersMapByHash map[uint32](map[string]struct{})
	membersSet       map[string]struct{}
}

// New creates a new Hashring32.
func New(hash HashFunc32, options ...Option) *Hashring32 {
	ring := &Hashring32{
		hash:             hash,
		numReplicas:      defaultNumReplicas,
		numPeersEstimate: defaultNumPeersEstimate,
		formatReplica:    formatSimpleReplica,
	}

	// set or override constructor options
	for _, opt := range options {
		opt.apply(ring)
	}

	// initialize inner fields
	capacity := ring.numReplicas * ring.numPeersEstimate
	ring.hashesArray = make([]uint32, 0, capacity)
	ring.membersMapByHash = make(map[uint32](map[string]struct{}), capacity)
	ring.membersSet = make(map[string]struct{}, ring.numPeersEstimate)
	// TODO: Adjust parameters base on benchmarks
	ring.sorter = radixsort32.New(
		radixsort32.Radix(16),
		radixsort32.MaxLen(capacity),
		radixsort32.MinLen(1000))

	return ring
}

// Option is an option for the hash ring constructor.
type Option interface {
	apply(*Hashring32)
}

// NumReplicas specifies the number of replicas to use for each peer.
//
// More replicas produces a more even distribution of entities and slower
// membership updates.
//
// Changing the replica number changes the topology of the ring.
// Do not change the replica number on a populated ring.
// Drain any stateful service before changing the number of replicas.
func NumReplicas(n int) Option {
	return numReplicasOption{numReplicas: n}
}

type numReplicasOption struct {
	numReplicas int
}

func (o numReplicasOption) apply(r *Hashring32) {
	r.numReplicas = o.numReplicas
}

// ReplicaFormatter specifies the function the hash ring will use to construct
// replica names from a peer identifier and a replica number.
//
// Replica names are hashed to find their positions within the hash ring.
//
// The default replica formatter simply concatenates the peer identifier and
// the replica number as a decimal string.
func ReplicaFormatter(formatReplica ReplicaFormatterFunc) Option {
	return replicaFormatterOption{
		formatReplica: formatReplica,
	}
}

type replicaFormatterOption struct {
	formatReplica ReplicaFormatterFunc
}

func (o replicaFormatterOption) apply(r *Hashring32) {
	r.formatReplica = o.formatReplica
}

func formatSimpleReplica(identifier string, replicaNum int) string {
	return identifier + strconv.Itoa(replicaNum)
}

// DelimitedReplicaFormatter joins a peer identifier and replica number with a given delimiter.
func DelimitedReplicaFormatter(delimiter string) ReplicaFormatterFunc {
	return func(identifier string, replicaNum int) string {
		replica := strconv.Itoa(replicaNum)
		identifierLen := len(identifier)
		buffer := make([]byte, 0, identifierLen+len(replica)+len(delimiter))
		buffer = append(buffer, []byte(identifier)...)
		buffer = append(buffer, []byte(delimiter)...)
		buffer = append(buffer, []byte(replica)...)
		return string(buffer)
	}
}

// NumPeersEstimate specifies an estimate for the number of identified peers
// the hashring will contain.
//
// This figure and the number of replicas determines the initial capacity of the ring slice.
func NumPeersEstimate(numPeersEstimate int) Option {
	return numPeersEstimateOption{
		numPeersEstimate: numPeersEstimate,
	}
}

type numPeersEstimateOption struct {
	numPeersEstimate int
}

func (o numPeersEstimateOption) apply(r *Hashring32) {
	r.numPeersEstimate = o.numPeersEstimate
}

// Choose returns first (shard.N + 1) peers within the matched range of hashed
// shard.Key in the hash ring.
func (r *Hashring32) Choose(shard Shard) ([]string, error) {
	r.m.RLock()
	defer r.m.RUnlock()

	if shard.N < 0 {
		return nil, ErrNegativeN
	}

	if len(r.membersSet) == 0 {
		return nil, ErrEmptyPool
	}

	var ix int
	if shard.Key == "" {
		// Random index to get hash value
		ix = rand.Intn(len(r.hashesArray))
	} else {
		// Binary search to find hash value
		key := r.hash(shard.Key)
		ix = indexOf(r.hashesArray, key)
	}

	res := make([]string, shard.N+1)
	// used to remove duplicates
	set := make(map[string]struct{}, shard.N+1)
	for len(set) < shard.N+1 && len(set) < len(r.membersSet) {
		hash := r.hashesArray[ix]
		ix++

		// reach end of ring , start from the beginning
		if ix == len(r.hashesArray) {
			ix = 0
		}

		// same hash can contains different members (collisions)
		for member := range r.membersMapByHash[hash] {
			// different hashes can point to same server (replicas)
			if _, ok := set[member]; !ok {
				res[len(set)] = member
				set[member] = struct{}{}
				if len(set) == shard.N+1 {
					break
				}
			}
		}
	}

	return res[:len(set)], nil
}

// ChooseNth returns (shard.N + 1)th peer within the matched range of hashed
// shard.Key in the hash ring.
func (r *Hashring32) ChooseNth(shard Shard) (string, error) {
	r.m.RLock()
	defer r.m.RUnlock()

	if shard.N < 0 {
		return "", ErrNegativeN
	}

	if len(r.membersSet) == 0 {
		return "", ErrEmptyPool
	}

	var ix int
	if shard.Key == "" {
		// Random index to get hash value
		ix = rand.Intn(len(r.hashesArray))
	} else {
		// Binary search to find hash value
		key := r.hash(shard.Key)
		ix = indexOf(r.hashesArray, key)
	}

	// use by ChooseNth
	var last *string
	// used to remove duplicates
	var set map[string]struct{}
	if shard.N > 0 {
		set = make(map[string]struct{}, shard.N+1)
	}
	for set == nil || (len(set) < shard.N+1 && len(set) < len(r.membersSet)) {
		hash := r.hashesArray[ix]
		ix++

		// reach end of ring , start from the beginning
		if ix == len(r.hashesArray) {
			ix = 0
		}

		// same hash can contains different members (collisions)
		for member := range r.membersMapByHash[hash] {
			if set == nil {
				return member, nil
			}
			// different hashes can point to same server (replicas)
			if _, ok := set[member]; !ok {
				last = &member
				set[member] = struct{}{}
				if len(set) == shard.N+1 {
					break
				}
			}
		}
	}

	return *last, nil
}

// Add adds a member into the hash ring and returns whether it is a new member.
func (r *Hashring32) Add(member string) (new bool) {
	r.m.Lock()
	defer r.m.Unlock()

	if _, ok := r.membersSet[member]; ok {
		return false
	}

	r.addHelper(member)

	r.sorter.Sort(r.hashesArray)
	return true
}

// Remove removes a member from the hash ring and returns whether it was an existing member.
func (r *Hashring32) Remove(member string) (found bool) {
	r.m.Lock()
	defer r.m.Unlock()

	if _, ok := r.membersSet[member]; !ok {
		return false
	}

	delete(r.membersSet, member)

	toBeRemoved := make(map[int]struct{}, r.numReplicas)
	r.removeHelper(member, toBeRemoved)

	for i := range toBeRemoved {
		r.hashesArray[i] = uint32(math.MaxUint32)
	}
	r.sorter.Sort(r.hashesArray)
	r.hashesArray = r.hashesArray[:len(r.hashesArray)-len(toBeRemoved)]
	return true
}

// Include includes a group of new members into the hash ring
func (r *Hashring32) Include(group map[string]struct{}) {
	r.m.Lock()
	defer r.m.Unlock()

	for member := range group {
		if _, ok := r.membersSet[member]; ok {
			continue
		}

		r.addHelper(member)
	}
	r.sorter.Sort(r.hashesArray)
}

// Exclude excludes a group of new members from the hash ring
func (r *Hashring32) Exclude(group map[string]struct{}) {
	r.m.Lock()
	defer r.m.Unlock()

	// cannot use pre-allocated slices from buffer pool here because size is dynamic.
	toBeRemoved := make(map[int]struct{}, len(group)*r.numReplicas)

	for member := range group {
		if _, ok := r.membersSet[member]; !ok {
			continue
		}
		r.removeHelper(member, toBeRemoved)
	}

	for index := range toBeRemoved {
		r.hashesArray[index] = uint32(math.MaxUint32)
	}
	r.sorter.Sort(r.hashesArray)
	r.hashesArray = r.hashesArray[:len(r.hashesArray)-len(toBeRemoved)]
}

// Set clear the whole ring and replace with a group of new members.
func (r *Hashring32) Set(group map[string]struct{}) {
	r.m.Lock()
	defer r.m.Unlock()

	// cannot use pre-allocated slices from buffer pool here because size is dynamic.
	toBeRemoved := make(map[int]struct{}, len(group)*r.numReplicas)

	// remove old members
	for member := range r.membersSet {
		// skip intersection
		if _, ok := group[member]; ok {
			continue
		}
		r.removeHelper(member, toBeRemoved)
	}
	// add new members
	for member := range group {
		// skip intersection
		if _, ok := r.membersSet[member]; ok {
			continue
		}
		r.addHelper(member)
	}
	for i := range toBeRemoved {
		r.hashesArray[i] = uint32(math.MaxUint32)
	}
	r.sorter.Sort(r.hashesArray)
	r.hashesArray = r.hashesArray[:len(r.hashesArray)-len(toBeRemoved)]
}

// Len returns number of members of the hash ring.
func (r *Hashring32) Len() int {
	r.m.RLock()
	defer r.m.RUnlock()
	return len(r.membersSet)
}

// addHelper adds member into hashes array (without sorting) and map.
func (r *Hashring32) addHelper(member string) {
	r.membersSet[member] = struct{}{}
	for i := 0; i < r.numReplicas; i++ {
		hash := r.hash(r.formatReplica(member, i))

		if _, ok := r.membersMapByHash[hash]; !ok {
			r.membersMapByHash[hash] = make(map[string]struct{})
			r.hashesArray = append(r.hashesArray, hash)
		}
		r.membersMapByHash[hash][member] = struct{}{}
	}
}

// removeHelper adds a member into toBeRemoved set and removes it from map.
func (r *Hashring32) removeHelper(member string, toBeRemoved map[int]struct{}) {
	delete(r.membersSet, member)

	for i := 0; i < r.numReplicas; i++ {
		hash := r.hash(r.formatReplica(member, i))
		delete(r.membersMapByHash[hash], member)
		// no more member for this hash
		if len(r.membersMapByHash[hash]) == 0 {
			delete(r.membersMapByHash, hash)
			// mark removed ones.
			// do not remove immediately because they are needed for binary search.
			toBeRemoved[indexOf(r.hashesArray, hash)] = struct{}{}
		}
	}
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
