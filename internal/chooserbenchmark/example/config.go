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

package example

import (
	"time"

	"go.uber.org/yarpc/api/peer"
	bench "go.uber.org/yarpc/internal/chooserbenchmark"
	"go.uber.org/yarpc/peer/pendingheap"
	"go.uber.org/yarpc/peer/roundrobin"
)

// PendingHeap returns a list implemented fewest pending strategy
func PendingHeap(t peer.Transport) peer.ChooserList {
	return pendingheap.New(t)
}

// RoundRobin returns a list implemented round robin strategy
func RoundRobin(t peer.Transport) peer.ChooserList {
	return roundrobin.New(t)
}

// RoundRobinWorks is a test configuration example when round robin is enough
var RoundRobinWorks = &bench.Config{
	ClientGroups: []bench.ClientGroup{
		{
			Name:        "roundrobin",
			Count:       500,
			RPS:         20,
			Constructor: RoundRobin,
		},
		{
			Name:        "pendingheap",
			Count:       500,
			RPS:         20,
			Constructor: PendingHeap,
		},
	},
	ServerGroups: []bench.ServerGroup{
		{
			Name:    "normal",
			Count:   50,
			Latency: time.Millisecond * 100,
		},
	},
	Duration: 10 * time.Second,
}

// FewestPendingSuperior is a test configuration example when fewest pending
// requests strategy is better
var FewestPendingSuperior = &bench.Config{
	ClientGroups: []bench.ClientGroup{
		{
			Name:        "roundrobin",
			Count:       1000,
			RPS:         20,
			Constructor: RoundRobin,
		},
		{
			Name:        "pendingheap",
			Count:       1000,
			RPS:         20,
			Constructor: PendingHeap,
		},
	},
	ServerGroups: []bench.ServerGroup{
		{
			Name:    "normal",
			Count:   5,
			Latency: time.Millisecond * 100,
		},
		{
			Name:    "slow",
			Count:   5,
			Latency: time.Second,
		},
	},
	Duration: 10 * time.Second,
}

// FewestPendingDegradation is an example when fewest pending request not work
var FewestPendingDegradation = &bench.Config{
	ClientGroups: []bench.ClientGroup{
		{
			Name:        "roundrobin",
			Count:       1000,
			RPS:         20,
			Constructor: RoundRobin,
		},
		{
			Name:        "pendingheap",
			Count:       1000,
			RPS:         20,
			Constructor: PendingHeap,
		},
	},
	ServerGroups: []bench.ServerGroup{
		{
			Name:    "normal",
			Count:   50,
			Latency: time.Millisecond * 100,
		},
		{
			Name:    "slow",
			Count:   50,
			Latency: time.Second,
		},
	},
	Duration: 10 * time.Second,
}
