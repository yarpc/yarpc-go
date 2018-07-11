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

package chooserbenchmark

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/atomic"
	"go.uber.org/yarpc/api/peer"
	"go.uber.org/yarpc/peer/pendingheap"
)

func serverForTest(id int, name string, listeners Listeners, latency time.Duration) (*Server, error) {
	start, stop := make(chan struct{}), make(chan struct{})
	lis, err := listeners.Listener(id)
	if err != nil {
		return nil, err
	}
	return NewServer(id, name, latency, DefaultLogNormalSigma, lis, start, stop, &sync.WaitGroup{})
}

func clientForTest(id int, group *ClientGroup, listeners Listeners) *Client {
	start, stop := make(chan struct{}), make(chan struct{})
	return NewClient(id, group, listeners, start, stop, &sync.WaitGroup{})
}

func TestNewServerGroupMeta(t *testing.T) {
	counters := []int64{12, 13, 1, 3, 5}
	serverCount := len(counters)
	buckets := []int64{4, 15}
	servers := make([]*Server, serverCount)
	listners := NewListeners(serverCount)
	sum := int64(0)
	latency := time.Duration(1000)
	for i := 0; i < len(counters); i++ {
		server, err := serverForTest(i, "foo", listners, latency)
		server.counter = counters[i]
		sum += counters[i]
		assert.NoError(t, err)
		servers[i] = server
	}
	meta := newServerGroupMeta("bar", servers, buckets)
	assert.Equal(t, "bar", meta.name)
	assert.Equal(t, serverCount, len(meta.servers))
	assert.Equal(t, sum, meta.requestCount)
	assert.Equal(t, latency, meta.meanLatency)
}

func TestNewClientGroupMetaThenPopulate(t *testing.T) {
	// the logic is similar to server, here we test when there is only one group
	resCounters := []int64{1, 2, 4, 5}
	reqCounters := []int64{2, 3, 4, 5}
	histogramIndice := []int64{3, 2, 3, 1, 8}
	clientCount := len(resCounters)
	rps := 100
	group := &ClientGroup{
		Name:        "foo",
		Count:       clientCount,
		RPS:         rps,
		Constructor: func(t peer.Transport) peer.ChooserList { return pendingheap.New(t) },
	}
	listeners := NewListeners(clientCount)
	clients := make([]*Client, clientCount)
	reqCount, resCount := int64(0), int64(0)
	for i := 0; i < clientCount; i++ {
		clients[i] = clientForTest(i, group, listeners)
		clients[i].resCounter = *atomic.NewInt64(resCounters[i])
		clients[i].reqCounter = *atomic.NewInt64(reqCounters[i])
		reqCount += reqCounters[i]
		resCount += resCounters[i]
		clients[i].histogram.counters[histogramIndice[i]].Inc()
	}
	meta, err := newClientGroupMeta("foo", clients)
	assert.NoError(t, err)
	assert.Equal(t, "foo", meta.name)
	assert.Equal(t, clientCount, meta.count)
	assert.Equal(t, reqCount, meta.reqCount)
	assert.Equal(t, resCount, meta.resCount)
	assert.Equal(t, rps, meta.rps)

	clientGroupNames, clientData, maxLatencyFrequency, err := populateClientData(clients)
	assert.NoError(t, err)
	assert.True(t, len(clientGroupNames) == 1 && clientGroupNames[0] == "foo")
	assert.Equal(t, 1, len(clientData))
	assert.Equal(t, int64(2), maxLatencyFrequency)
}

func TestAggregateThenPopulateServer(t *testing.T) {
	groupNames := []string{"a", "b"}
	latency := time.Duration(1)
	countersGroupA := []int64{99, 10}
	countersGroupB := []int64{1, 2, 3, 4, 5}
	aCount := len(countersGroupA)
	bCount := len(countersGroupB)
	servers := make([]*Server, aCount+bCount)
	listeners := NewListeners(aCount + bCount)
	for i := 0; i < aCount; i++ {
		server, err := serverForTest(i, "a", listeners, latency)
		assert.NoError(t, err)
		server.counter = countersGroupA[i]
		servers[i] = server
	}
	for i := 0; i < bCount; i++ {
		server, err := serverForTest(i+aCount, "b", listeners, latency)
		assert.NoError(t, err)
		server.counter = countersGroupB[i]
		servers[i+aCount] = server
	}
	serverGroupNames, serversByGroup, maxRequestCount, minRequestCount := aggregateServersByGroupName(servers)
	assert.True(t, serverGroupNames[0] == groupNames[0] && serverGroupNames[1] == groupNames[1])
	assert.Equal(t, 2, len(serversByGroup))
	assert.Equal(t, int64(99), maxRequestCount)
	assert.Equal(t, int64(1), minRequestCount)

	buckets := []int64{5, 20, 100}
	serverData, maxServerCount, maxServerHistogramValue := populateServerData(serverGroupNames, serversByGroup, buckets)
	assert.Equal(t, 2, len(serverData))
	assert.Equal(t, 5, maxServerCount)
	assert.Equal(t, int64(5), maxServerHistogramValue)
}

func TestGaugeCalculation(t *testing.T) {
	g := gaugeCalculation(1, int64(12345), int64(1234))
	assert.Equal(t, 4, g.idLen)
	assert.Equal(t, 4, g.freqLen)
	assert.Equal(t, 8, g.counterLen)
	// we didn't assert star unit here, because they will change when you change display setting
}

func TestGetNumLength(t *testing.T) {
	assert.Equal(t, 3, getNumLength(123))
	assert.Equal(t, 1, getNumLength(1))
	assert.Equal(t, 9, getNumLength(123456789))
}

func TestNormalizeLength(t *testing.T) {
	assert.Equal(t, 0, normalizeLength(-1))
	assert.Equal(t, 4, normalizeLength(1))
	assert.Equal(t, 4, normalizeLength(4))
	assert.Equal(t, 8, normalizeLength(5))
	assert.Equal(t, 8, normalizeLength(7))
	assert.Equal(t, 342836, normalizeLength(342834))
}
