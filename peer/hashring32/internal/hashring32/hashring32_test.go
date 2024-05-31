// Copyright (c) 2024 Uber Technologies, Inc.
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
	"sort"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/yarpc/peer/hashring32/internal/farmhashring"
)

// Unit Tests

const (
	ringpopID1 = "127.0.0.1:10000"
	ringpopID2 = "127.0.0.1:10001"
	ringpopID3 = "127.0.0.1:10002"
	ringpopID4 = "127.0.0.1:10004"
	ringpopID5 = "127.0.0.1:10005"

	key1 = "123"
	key2 = "456"
)

func makeHashring32() *Hashring32 {
	return New(
		farmhashring.Fingerprint32,
		NumReplicas(100),
		NumPeersEstimate(1500),
	)
}

func TestAddRemove(t *testing.T) {
	rp := makeHashring32()

	b := rp.Add(ringpopID1)
	b2 := rp.Add(ringpopID1)
	assertRingState(t, rp)

	ids, err := rp.Choose(Shard{
		Key: key1,
	})
	id := ids[0]
	assert.NoError(t, err, "Choose failed to select peer")
	assert.True(t, b, "Choose returned false but element did not exist.")
	assert.False(t, b2, "Choose returned true but element already exists.")
	assert.Equal(t, ringpopID1, id, "aaa")
	assert.Equal(t, 1, rp.Len(), "Size of members should be 1")

	b = rp.Remove(ringpopID1)
	b2 = rp.Remove(ringpopID1)
	assertRingState(t, rp)

	ids, err = rp.Choose(Shard{
		Key: key1,
	})
	assert.True(t, b, "Remove returned false but element were actually removed")
	assert.False(t, b2, "Remove returned true but element already exists.")
	assert.Error(t, err, "Expected not to find any peer")
	assert.Nil(t, ids, "Expected to find empty string as ID")
	assert.Equal(t, 0, rp.Len(), "Size of members should be 1")
}

// TestMultipleChoose tests whether peer chooser
// always selects the same peer when ring is at the same topology.
func TestMultipleChoose(t *testing.T) {
	rp := makeHashring32()
	rp.Add(ringpopID1)
	rp.Add(ringpopID2)
	rp.Add(ringpopID3)
	assertRingState(t, rp)

	assert.Equal(t, 3, rp.Len(), "Size of members should be 1")
	id, err := rp.Choose(Shard{
		Key: key1,
	})
	assert.NoError(t, err, "Choose failed to select peer")
	id2, err := rp.Choose(Shard{
		Key: key1,
	})
	assert.NoError(t, err, "Choose failed to select peer")
	assert.Equal(t, id, id2, "Choose selected a different peer")
	id3, err := rp.Choose(Shard{
		Key: key1,
	})
	assert.NoError(t, err, "Choose failed to select peer")
	assert.Equal(t, id, id3, "Choose selected a different peer")

	rp.Add(ringpopID4)
	rp.Add(ringpopID5)
	rp.Remove(ringpopID4)
	rp.Remove(ringpopID5)
	assertRingState(t, rp)

	id2, err = rp.Choose(Shard{
		Key: key1,
	})
	assert.NoError(t, err, "Choose failed to select peer")
	assert.Equal(t, id, id2, "Choose selected a different peer")
	id3, err = rp.Choose(Shard{
		Key: key1,
	})
	assert.NoError(t, err, "Choose failed to select peer")
	assert.Equal(t, id, id3, "Choose selected a different peer")
}

func TestNoKey(t *testing.T) {
	rp := makeHashring32()
	rp.Add(ringpopID1)
	assertRingState(t, rp)

	id, err := rp.Choose(Shard{Key: ""})
	assert.NoError(t, err, "Should returns a random individual when no shard key.")
	assert.Equal(t, ringpopID1, id[0], "Should returns a random individual.")
}

func TestRingpopIncludeExclude(t *testing.T) {
	rp := makeHashring32()
	peers := map[string]struct{}{
		ringpopID1: {},
		ringpopID2: {},
		ringpopID3: {},
		ringpopID4: {},
		ringpopID5: {},
	}
	rp.Include(peers)
	assert.Equal(t, 5, rp.Len(), "Load balancer pool size should be 5.")
	rp.Include(peers)
	assert.Equal(t, 5, rp.Len(), "Load balancer pool size should be 5.")

	toBeRemoved := map[string]struct{}{
		ringpopID2: {},
		ringpopID3: {},
		ringpopID4: {},
	}
	rp.Exclude(toBeRemoved)
	assert.Equal(t, 2, rp.Len(), "Load balancer pool size should be 2")
	rp.Exclude(toBeRemoved)
	assert.Equal(t, 2, rp.Len(), "Load balancer pool size should be 2")
}

func TestHashCollision(t *testing.T) {
	rp := makeHashring32()
	// Add collisions
	rp.Add("98.25.121.4:24016")
	rp.Add("251.73.56.7:21031")
	assertRingState(t, rp)

	b := rp.Remove("98.25.121.4:24016")
	assert.Equal(t, 1, rp.Len(), "Expects length to be 1")
	b2 := rp.Remove("251.73.56.7:21031")
	assertRingState(t, rp)
	assert.Equal(t, 0, rp.Len(), "Expects length to be 1")
	assert.True(t, b, "Element should exist")
	assert.True(t, b2, "Element should exist")
}

func TestIndexOf(t *testing.T) {
	var empty = []uint32{}
	assert.Equal(t, -1, indexOf(empty, uint32(0)), "Should return -1 for empty array")

	var slice = []uint32{0, 1, 2, 3, 4}
	assert.Equal(t, 0, indexOf(slice, uint32(0)), "Should return index 0")
	assert.Equal(t, 3, indexOf(slice, uint32(3)), "Should return index 3")
	assert.Equal(t, 4, indexOf(slice, uint32(4)), "Should return index 4")
	assert.Equal(t, 0, indexOf(slice, uint32(5)), "Should return index 0")
}

func TestNotEnoughPeers(t *testing.T) {
	rp := makeHashring32()
	rp.Add("127.0.0.1:0")
	assertRingState(t, rp)

	peers, err := rp.Choose(Shard{Key: key1, N: 1})
	assert.Equal(t, 1, len(peers))
	assert.Equal(t, "127.0.0.1:0", peers[0])
	assert.NoError(t, err)

	peer, err := rp.ChooseNth(Shard{Key: key1, N: 1})
	assert.Equal(t, "127.0.0.1:0", peer)
	assert.NoError(t, err)
}

func TestSimpleReplicaFormatter(t *testing.T) {
	assert.Equal(t, "member10", formatSimpleReplica("member", 10))
}

func TestDelimitedReplicaFormatter(t *testing.T) {
	PoundReplicaFormatter := DelimitedReplicaFormatter("#")
	assert.Equal(t, "member#11", PoundReplicaFormatter("member", 11))

	EmptyReplicaFormatter := DelimitedReplicaFormatter("")
	assert.Equal(t, "member12", EmptyReplicaFormatter("member", 12))

	TildaReplicaFormatter := DelimitedReplicaFormatter("~")
	assert.Equal(t, "member~13", TildaReplicaFormatter("member", 13))
}

// Benchmarks
func BenchmarkHashring32Adds100(b *testing.B) {
	benchmarkAdds(b, makeHashring32(), 100)
}

func BenchmarkHashring32Adds1000(b *testing.B) {
	benchmarkAdds(b, makeHashring32(), 1000)
}

func BenchmarkHashring32Removes100(b *testing.B) {
	benchmarkRemoves(b, makeHashring32(), 100)
}

func BenchmarkHashring32Removes1000(b *testing.B) {
	benchmarkRemoves(b, makeHashring32(), 1000)
}

func BenchmarkHashring32Include100(b *testing.B) {
	benchmarkInclude(b, makeHashring32(), 100)
}

func BenchmarkHashring32Include1000(b *testing.B) {
	benchmarkInclude(b, makeHashring32(), 1000)
}

func BenchmarkHashring32Exclude100(b *testing.B) {
	benchmarkExclude(b, makeHashring32(), 100)
}

func BenchmarkHashring32Exclude1000(b *testing.B) {
	benchmarkExclude(b, makeHashring32(), 1000)
}

func BenchmarkHashring32Choose100NoHint(b *testing.B) {
	benchmarkChoose(b, makeHashring32(), 100)
}

func BenchmarkHashring32Choose1000NoHint(b *testing.B) {
	benchmarkChoose(b, makeHashring32(), 1000)
}

func BenchmarkHashring32Set100Updates10(b *testing.B) {
	benchmarkSet(b, makeHashring32(), 100, 10)
}

func BenchmarkHashring32Set1000Updates100(b *testing.B) {
	benchmarkSet(b, makeHashring32(), 1000, 100)
}

func BenchmarkHashring32Set1500Updates1500(b *testing.B) {
	benchmarkSet(b, makeHashring32(), 1000, 1000)
}

// benchmark helpers

func benchmarkAdds(b *testing.B, hashring32 *Hashring32, capacity int) {
	b.StopTimer()
	updates := generateStringsGroup(capacity)
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		hashring32.Set(generateEmptySet())
		b.StartTimer()
		for member := range updates {
			hashring32.Add(member)
		}
	}
}

func benchmarkRemoves(b *testing.B, hashring32 *Hashring32, capacity int) {
	b.StopTimer()
	updates := generateStringsGroup(capacity)
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		hashring32.Set(deepCopy(updates))
		b.StartTimer()
		for member := range updates {
			hashring32.Remove(member)
		}
	}
}

func benchmarkInclude(b *testing.B, hashring32 *Hashring32, capacity int) {
	b.StopTimer()
	updates := generateStringsGroup(capacity)
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		hashring32.Set(generateEmptySet())
		b.StartTimer()
		hashring32.Include(updates)
	}
}

func benchmarkExclude(b *testing.B, hashring32 *Hashring32, capacity int) {
	b.StopTimer()
	updates := generateStringsGroup(capacity)
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		hashring32.Set(deepCopy(updates))
		b.StartTimer()
		hashring32.Exclude(updates)
	}
}

// If hints is empty use NoHint otherwise pick a hint in the array in round robin.
func benchmarkChoose(b *testing.B, hashring32 *Hashring32, capacity int, shards ...Shard) {
	b.StopTimer()
	updates := generateStringsGroup(capacity)
	hashring32.Set(updates)
	numShards := len(shards)
	k := 0
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		for j := 0; j < capacity; j++ {
			if numShards == 0 {
				hashring32.Choose(Shard{})
			} else {
				hashring32.Choose(shards[k])
				k++
				if k == numShards {
					k = 0
				}
			}
		}
	}
}

// capacity = size of old group = size of new group
// numUpdates is the size of members in new group but not old group
// numUpdates has to be smaller than or equal to capacity
func benchmarkSet(b *testing.B, hashring32 *Hashring32, capacity, numUpdates int) {
	b.StopTimer()
	tmp := generateStringsGroup(capacity + numUpdates)
	old := make(map[string]struct{})
	new := make(map[string]struct{})
	j := 0
	for member := range tmp {
		if j < capacity {
			old[member] = struct{}{}
		}
		if j >= numUpdates {
			new[member] = struct{}{}
		}
		j++
	}
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		hashring32.Set(old)
		b.StartTimer()
		hashring32.Set(new)
	}
}

// helpers

func deepCopy(set map[string]struct{}) map[string]struct{} {
	target := make(map[string]struct{})
	for k := range set {
		target[k] = struct{}{}
	}
	return target
}

func generateEmptySet() map[string]struct{} {
	return make(map[string]struct{})
}

// better to use capacity 0 ~ 65535
func generateStringsGroup(capacity int) map[string]struct{} {
	set := make(map[string]struct{})
	for i := 0; i < capacity; i++ {
		str := "127.0.0.1:" + strconv.Itoa(i)
		set[str] = struct{}{}
	}
	return set
}

// assertRingState verifies whether the ring is in a valid state
func assertRingState(t *testing.T, hr *Hashring32) {
	// whether hash array is sorted
	isSorted(t, hr)
	// whether hash array is synced with hash map
	arrayMapSynced(t, hr)
	// whether choose with various N returns valid results
	chooseCombo(t, hr, key1)
	chooseCombo(t, hr, key2)
}

// chooseCombo provides sanity test with various N compared to the ring size.
func chooseCombo(t *testing.T, hr *Hashring32, key string) {
	// Choose exactly 1 peer
	chooseN(t, hr, key, 0)
	// Choose more than peer list size
	chooseN(t, hr, key, hr.Len())
	// Choose exactly peer list size
	chooseN(t, hr, key, hr.Len()-1)
	// Choose less peers than peer list size
	chooseN(t, hr, key, hr.Len()-2)

	// Choose the 1st (index 0) peer
	chooseNth(t, hr, key, 0)
	// Choose (size + 1)th peer
	chooseNth(t, hr, key, hr.Len())
	// Choose (size)th peer
	chooseNth(t, hr, key, hr.Len()-1)
	// Choose (size - 1)th peer
	chooseNth(t, hr, key, hr.Len()-2)

	if key != "randomKey" {
		chooseCombo(t, hr, "randomKey")
	}
}

func chooseN(t *testing.T, hr *Hashring32, key string, n int) {
	size := hr.Len()

	peers, err := hr.Choose(Shard{
		N:   n,
		Key: key,
	})

	if n < 0 {
		assert.Error(t, err, "n cannot be negative integer")
		assert.Nil(t, peers, "peers should be nil when n is negative")
		return
	}

	if size == 0 {
		assert.Error(t, err, "should return error for empty peer list")
		assert.Nil(t, peers, "peers should be nil when n is negative")
		return
	}

	// all provided peers are in members set
	for _, peer := range peers {
		_, ok := hr.membersSet[peer]
		assert.True(t, ok, "returned peer not in members set")

	}

	if n+1 >= size {
		assert.NoError(t, err, "should not return error even when size smaller than n")
		assert.Len(t, peers, size, "should return all available peers when requested more than that")
		return
	}

	// n + 1 < size
	assert.Len(t, peers, n+1, "number of peers should equal to n + 1")
}

func chooseNth(t *testing.T, hr *Hashring32, key string, n int) {
	size := hr.Len()

	peer, err := hr.ChooseNth(Shard{
		N:   n,
		Key: key,
	})

	if n < 0 {
		assert.Error(t, err, "n cannot be negative integer")
		assert.Equal(t, peer, "", "peers should be nil when n is negative")
		return
	}

	if size == 0 {
		assert.Error(t, err, "should return error for empty peer list")
		assert.Equal(t, peer, "", "peers should be empty string when no peer")
		return
	}

	// n + 1 < size
	assert.NotEqual(t, peer, "")
	if _, ok := hr.membersSet[peer]; ok {
		assert.True(t, ok, "returned peer not in members set")
	}
}

func isSorted(t *testing.T, ring *Hashring32) {
	assert.True(t, sort.SliceIsSorted(ring.hashesArray, func(i, j int) bool {
		return ring.hashesArray[i] < ring.hashesArray[j]
	}), "hashes array should be sorted")
}

func arrayMapSynced(t *testing.T, ring *Hashring32) {
	assert.Equal(t,
		len(ring.membersMapByHash), len(ring.hashesArray), "hashes array and map should be always synced")
	for _, hash := range ring.hashesArray {
		v := ring.membersMapByHash[hash]
		{
			assert.NotNil(t, v, "hashes array and map should be always synced")
		}
	}
}
