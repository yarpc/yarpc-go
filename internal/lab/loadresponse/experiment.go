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

package main

import (
	"sync"
	"time"

	"go.uber.org/yarpc/api/peer"
)

// Experiment captures the parameters for a load response experiment.
//
// Example,
//
//  5 clients
//  10 client workers per client
//  pace 1 request per client worker per 1 millisecond
//  2 millisecond timeout per request
//  50 requests per millisecond
//  5 servers
//  10 workers per server
//  1 millisecond latency per request
//  50 concurrent requests handled
type Experiment struct {
	ClientCount       int
	ClientConcurrency int
	ServerCount       int
	ServerConcurrency int
	QueueLength       int
	Pace              time.Duration
	Timeout           time.Duration
	Duration          time.Duration
	Latency           func(serverNumber int, serverCount int) time.Duration
	NewList           func(peer.Transport) peer.ChooserList
}

// Results captures the aggregated results of an experiment.
type Results struct {
	Requested int
	Responded int
	Dropped   int
	TimedOut  int
	Duration  time.Duration
	// TODO ThroughputRPS int
	// TODO latency histogram
}

// Run executes an experiment.
func (e Experiment) Run() Results {
	servers := NewServers(e.ServerCount, e.ServerConcurrency, e.QueueLength, e.Latency)
	clients := NewClients(e.ClientCount, e.ClientConcurrency, e.Pace, e.Timeout, servers, e.NewList)

	done := make(chan struct{})
	var wg sync.WaitGroup

	start := time.Now()
	servers.Start(done, &wg)
	clients.Start(done, &wg)

	time.Sleep(e.Duration)
	close(done)
	wg.Wait()
	stop := time.Now()

	results := Results{
		Duration: stop.Sub(start),
	}
	for _, client := range clients {
		results.Requested += client.Requested
		results.TimedOut += client.TimedOut
		results.Dropped += client.Dropped
		results.Responded += client.Responded
	}
	return results
}
