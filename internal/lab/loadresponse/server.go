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
)

// Servers captures a cluster of servers.
type Servers []Server

// NewServers returns a cluster of servers.
func NewServers(count int, concurrency int, queueLength int, latency func(int, int) time.Duration) (servers Servers) {
	for i := 0; i < count; i++ {
		server := Server{
			ID:          i,
			ServerCount: count,
			Concurrency: concurrency,
			QueueLength: queueLength,
			Latency:     latency(i, count),
		}
		server.Init()
		servers = append(servers, server)
	}
	return
}

// Start kicks off a cluster of servers.
//
// The server ups the wait group on start, runs until the done channel closes,
// and downs the wait group symmetrically.
func (s Servers) Start(done chan struct{}, wg *sync.WaitGroup) {
	for i := 0; i < len(s); i++ {
		s[i].Start(done, wg)
	}
}

// Server represents a simulated server.
type Server struct {
	ID          int
	ServerCount int
	Inbox       chan Request
	Concurrency int
	QueueLength int
	Latency     time.Duration
}

// Init prepares a server so it can start.
func (s *Server) Init() {
	s.Inbox = make(chan Request, s.QueueLength)
}

// Start kicks off a cluster of servers.
//
// The server ups the wait group, runs until the done channel closes, and
// symmetrically downs the wait group.
func (s *Server) Start(done chan struct{}, wg *sync.WaitGroup) {
	wg.Add(s.Concurrency)
	for i := 0; i < s.Concurrency; i++ {
		go s.run(done, wg)
	}
}

// run is a worker goroutine for a server.
func (s *Server) run(done chan struct{}, wg *sync.WaitGroup) {
	defer wg.Done()

Loop:
	for {
		select {
		case <-done:
			break Loop
		case req := <-s.Inbox:
			// TODO timer reset
			time.Sleep(s.Latency)

			select {
			case <-done:
				break Loop
			case req.Response <- Response{
				Server: s.ID,
			}:
			default:
			}
		}

	}
}
