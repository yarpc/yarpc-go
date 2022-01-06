// Copyright (c) 2022 Uber Technologies, Inc.
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

// hashring32 with farmhash fingerprint32 produces a consistent hash ring that
// is compatible with Ringpop.
// This test verifies compatibility.

package hashring32

import (
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/dgryski/go-farm"
	"github.com/stretchr/testify/assert"
	"github.com/uber/ringpop-go/hashring"
	"github.com/uber/ringpop-go/membership"
)

// Shadow tests create a real RingPop ring, and make the scheduler shadowing
// all the ring topology updates. Ring size and look up results are compared
// and verified to be identical after every operation.

func TestRingpopShadowAddRemove(t *testing.T) {
	ring := newRing(100, t)
	servers := generateServers(100)

	for server := range servers {
		ring.add(server)
		ring.sameMembers()
	}
	for server := range servers {
		ring.remove(server)
		ring.sameMembers()
	}
}

func TestRingpopShadowIncludeExclude(t *testing.T) {
	ring := newRing(100, t)
	servers := generateServers(100)

	ring.include(servers)
	ring.sameMembers()
	ring.exclude(servers)
	ring.sameMembers()
}

func TestRingpopShadowSet(t *testing.T) {
	ring := newRing(100, t)
	servers := generateServers(100)
	ring.include(servers)
	newServers := newPopulation(10, servers)
	ring.set(newServers)

}

func TestRingpopShadowChoose(t *testing.T) {
	ring := newRing(100, t)
	servers := generateServers(1000)
	ring.set(servers)

	for i := 0; i < 100; i++ {
		ring.lookup(generateRandomString())
	}
}

func TestRingpopShadowCombo(t *testing.T) {
	ring := newRing(100, t)

	s1 := generateRandomHostPort()
	s2 := generateRandomHostPort()
	s3 := generateRandomHostPort()
	ring.add(s1)
	ring.add(s2)
	ring.add(s3)
	for i := 0; i < 20; i++ {
		ring.lookup(generateRandomString())
	}
	ring.remove(s1)
	ring.sameMembers()

	for i := 0; i < 20; i++ {
		ring.lookup(generateRandomString())
	}

	servers := generateServers(100)
	ring.include(servers)
	for i := 0; i < 100; i++ {
		ring.lookup(generateRandomString())
	}

	newServers := newPopulation(10, servers)
	ring.set(newServers)
	for i := 0; i < 100; i++ {
		ring.lookup(generateRandomString())
		ring.lookupN(generateRandomString(), 5)
		ring.lookupNth(generateRandomString(), 5)
	}
}

type ring struct {
	hashring *hashring.HashRing
	pool     map[string]*member

	shadowScheduler *Hashring32

	t *testing.T
	m sync.RWMutex
}

func newRing(replica int, t *testing.T) *ring {
	return &ring{
		hashring:        hashring.New(farm.Fingerprint32, replica),
		pool:            make(map[string]*member),
		shadowScheduler: makeHashring32(),
		t:               t,
	}
}

type member string

func newRingMember(id string) *member {
	member := member(id)
	return &member
}

// Label is a dummy method for implementing ringpop.Member interface.
func (f *member) Label(key string) (string, bool) {
	return "", false
}

// ID->Address mapping is managed by outside libraries.
func (f *member) GetAddress() string {
	// When GetAddress() and Identity() provides different value,
	// Ringpop uses new format fmt.Sprintf("%s#%v", identity, replica)
	// Use this to force ringpop library use the new format.
	return (string)(*f)
}

func (f *member) Identity() string {
	return (string)(*f)
}

func (r *ring) sameMembers() {
	assert.Equal(r.t, len(r.hashring.Servers()), r.shadowScheduler.Len())
	for s := range r.shadowScheduler.membersSet {
		assert.True(r.t, r.hashring.HasServer(s), fmt.Sprintf("Expected not to see %s.", s))
	}
}

func (r *ring) lookup(key string) {
	res, ok := r.hashring.Lookup(key)
	r.m.RLock()
	defer r.m.RUnlock()

	resSArr, errS := r.shadowScheduler.Choose(Shard{
		Key: key,
	})
	resS := resSArr[0]

	assert.Equal(r.t, res, resS, "Load balancer should returns the same result.")
	if ok {
		assert.NoError(r.t, errS, "Load balancer should returns no error.")
	} else {
		assert.Error(r.t, errS, "Load Balancer should returns error.")
	}
}

func (r *ring) lookupN(key string, n int) {
	res := r.hashring.LookupN(key, n)
	r.m.RLock()
	defer r.m.RUnlock()

	resSArr, errS := r.shadowScheduler.Choose(Shard{
		Key: key,
		N:   n - 1,
	})

	assert.NoError(r.t, errS)
	assert.Equal(r.t, res, resSArr, "Load balancer should returns the same result.")
}

func (r *ring) lookupNth(key string, n int) {
	res := r.hashring.LookupN(key, n)
	r.m.RLock()
	defer r.m.RUnlock()

	resSArr, errS := r.shadowScheduler.ChooseNth(Shard{
		Key: key,
		N:   n - 1,
	})

	assert.NoError(r.t, errS)
	assert.Equal(r.t, res[n-1], resSArr, "Load balancer should returns the same result.")
}

func (r *ring) add(individual string) {
	r.m.Lock()
	defer r.m.Unlock()

	newMember := r.shadowScheduler.Add(individual)

	if r.pool[individual] != nil {
		assert.False(r.t, newMember, "Load balancer doesn't find member.")
		return
	}

	r.pool[individual] = newRingMember(individual)
	r.hashring.ProcessMembershipChanges([]membership.MemberChange{
		{After: r.pool[individual]},
	})

	assert.True(r.t, newMember, "Load balancer has a member that should not exist.")
}

func (r *ring) remove(individual string) {
	r.m.Lock()
	defer r.m.Unlock()

	found := r.shadowScheduler.Remove(individual)

	if r.pool[individual] == nil {
		assert.False(r.t, found, "Ringpop loadbalancer found a member that shoould not exist.")
		return
	}
	r.hashring.ProcessMembershipChanges([]membership.MemberChange{
		{Before: r.pool[individual]},
	})
	delete(r.pool, individual)

	assert.True(r.t, found, "Load balancer has a member that should not exist.")
}

func (r *ring) include(population map[string]struct{}) {
	r.m.Lock()
	membershipChanges := make([]membership.MemberChange, 0, len(population))
	for id := range population {
		if r.pool[id] == nil {
			r.pool[id] = newRingMember(id)
			membershipChanges = append(membershipChanges, membership.MemberChange{
				After: r.pool[id],
			})
		}
	}
	r.hashring.ProcessMembershipChanges(membershipChanges)
	r.m.Unlock()

	r.shadowScheduler.Include(population)
	assert.Equal(r.t, r.size(), r.shadowScheduler.Len(), "Sizes doesn't match the real ring.")
}

func (r *ring) exclude(population map[string]struct{}) {
	r.m.Lock()
	membershipChanges := make([]membership.MemberChange, 0, len(population))
	for id := range population {
		if r.pool[id] != nil {
			membershipChanges = append(membershipChanges, membership.MemberChange{
				Before: r.pool[id],
			})
			delete(r.pool, id)
		}
	}
	r.hashring.ProcessMembershipChanges(membershipChanges)
	r.m.Unlock()

	r.shadowScheduler.Exclude(population)
	assert.Equal(r.t, r.size(), r.shadowScheduler.Len(), "Sizes doesn't match the real ring.")
}

func (r *ring) set(population map[string]struct{}) {
	r.m.Lock()

	membershipChanges := make([]membership.MemberChange, 0, len(population))
	newPool := make(map[string]*member)
	for id := range population {
		newPool[id] = newRingMember(id)
		if r.pool[id] == nil {
			membershipChanges = append(membershipChanges, membership.MemberChange{
				After: newPool[id],
			})
		}
	}
	for id := range r.pool {
		if newPool[id] == nil {
			membershipChanges = append(membershipChanges, membership.MemberChange{
				Before: r.pool[id],
			})
		}
	}
	r.pool = newPool
	r.hashring.ProcessMembershipChanges(membershipChanges)
	r.m.Unlock()

	r.shadowScheduler.Set(population)
	assert.Equal(r.t, r.size(), r.shadowScheduler.Len(), "Sizes doesn't match the real ring.")
}

func (r *ring) size() int {
	r.m.RLock()
	defer r.m.RUnlock()
	return len(r.pool)
}

func generateRandomString() string {
	rand.Seed(time.Now().UnixNano())
	var letterRunes = []rune("1234567890abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

	b := make([]rune, 20)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

// helpers
const ipUpperBound = 256

func generateRandomHostPort() string {
	// Make port number exact 5 digits so no replica collisions
	return fmt.Sprintf("%d.%d.%d.%d:%d%d%d%d%d",
		rand.Intn(ipUpperBound),
		rand.Intn(ipUpperBound),
		rand.Intn(ipUpperBound),
		rand.Intn(ipUpperBound),
		rand.Intn(6)+1, // max port is 65535
		rand.Intn(6),
		rand.Intn(6),
		rand.Intn(4),
		rand.Intn(7))
}

func generateServers(count int) map[string]struct{} {
	servers := make(map[string]struct{}, count)
	for i := 0; i < count; i++ {
		for {
			hostport := generateRandomHostPort()
			if _, ok := servers[hostport]; ok {
				continue
			}
			servers[hostport] = struct{}{}
			break
		}
	}
	return servers
}

// newPopulation updates numUpdates individuals in the oldPopulation,
// and keeps the size unchanged.
func newPopulation(numUpdates int, oldPopulation map[string]struct{}) map[string]struct{} {
	count := len(oldPopulation)

	sameServersCount := count - numUpdates
	newPopulation := make(map[string]struct{})

	for server := range oldPopulation {
		if len(newPopulation) == sameServersCount {
			break
		}
		newPopulation[server] = struct{}{}
	}

	// Generate new servers so that size doesn't change
	for {
		if len(newPopulation) == count {
			break
		}
		server := generateRandomHostPort()
		// Avoid adding old servers
		if _, ok := oldPopulation[server]; ok {
			continue
		}
		newPopulation[server] = struct{}{}
	}
	return newPopulation
}
