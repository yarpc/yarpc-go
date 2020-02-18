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
	"context"
	"math/rand"
	"sync"
	"time"

	"go.uber.org/yarpc/api/peer"
	"go.uber.org/yarpc/api/transport"
)

// Clients is a cluster of clients.
type Clients []Client

// NewClients creates a client cluster.
func NewClients(
	count int,
	concurrency int,
	pace time.Duration,
	timeout time.Duration,
	servers []Server,
	newList func(peer.Transport) peer.ChooserList,
) (clients Clients) {
	for i := 0; i < count; i++ {
		clients = append(clients, Client{
			ID:          i,
			Concurrency: concurrency,
			Pace:        pace,
			Timeout:     timeout,
			Servers:     servers,
			NewList:     newList,
		})
	}
	return
}

// Start kicks off a client cluster.
//
// The clients shut down when the done channel closes and report that they are
// shut down by downing the wait group by as much as starting upped.
func (c Clients) Start(done chan struct{}, wg *sync.WaitGroup) {
	for i := 0; i < len(c); i++ {
		c[i].Start(done, wg)
	}
}

// Client is a set of load generators that share a peer list.
type Client struct {
	ID          int
	Concurrency int
	Pace        time.Duration
	Timeout     time.Duration
	Servers     []Server
	NewList     func(peer.Transport) peer.ChooserList

	Requested int
	Responded int
	Dropped   int
	TimedOut  int
}

// Start kicks off a batch of client workers.
//
// The client adds to the wait group, runs until the done channel closes, and
// downs the wait group when all workers have stopped.
func (c *Client) Start(done chan struct{}, pwg *sync.WaitGroup) {
	pwg.Add(1)

	trans := NewTransport(c.Servers)
	list := c.NewList(trans)
	_ = list.Update(peer.ListUpdates{
		Additions: jot(len(c.Servers)),
	})

	_ = list.Start()

	var wg sync.WaitGroup
	wg.Add(c.Concurrency)
	workers := make([]ClientWorker, c.Concurrency)

	for i := 0; i < c.Concurrency; i++ {
		w := &workers[i]
		w.ClientID = c.ID
		w.WorkerID = i
		w.Servers = c.Servers
		w.List = list
		w.Pace = c.Pace
		w.Timeout = c.Timeout

		go workers[i].run(done, &wg)
	}

	go func() {
		defer pwg.Done()
		wg.Wait()
		_ = list.Stop()

		for i := 0; i < c.Concurrency; i++ {
			w := &workers[i]
			c.Requested += w.Requested
			c.Responded += w.Responded
			c.TimedOut += w.TimedOut
			c.Dropped += w.Dropped
		}
	}()
}

// jot generates a numbered list of identifiers.
func jot(n int) (ids []peer.Identifier) {
	for i := 0; i < n; i++ {
		ids = append(ids, Identifier(i))
	}
	return
}

// ClientWorker is an individual load generator for a client.
type ClientWorker struct {
	ClientID int
	WorkerID int
	Servers  []Server
	List     peer.ChooserList
	Pace     time.Duration
	Timeout  time.Duration

	Requested int
	Responded int
	Dropped   int
	TimedOut  int
}

// run is a goroutine worker that will generate load until the done channel
// closes and then will down the wait group to signal completion.
func (c *ClientWorker) run(done chan struct{}, wg *sync.WaitGroup) {
	defer wg.Done()

	timer := time.NewTimer(c.Pace)
	defer timer.Stop()

Loop:
	for {
		select {
		case <-done:
			break Loop
		case <-timer.C:
		}
		// Mean time between requests is pace, with variance from 0 to 2 times.
		timer.Reset(c.Pace * 2 *
			time.Duration(rand.Intn(1000)) /
			time.Duration(1000))

		c.call(done, c.List)
	}
}

// call generates a single request of load and blocks until the worker is asked
// to shut down, the request times out, or a server responds.
func (c *ClientWorker) call(done chan struct{}, list peer.Chooser) {
	req := &transport.Request{}
	ctx, cancel := context.WithTimeout(context.Background(), c.Timeout)
	defer cancel()

	p, onFinish, err := list.Choose(ctx, req)
	if err != nil {
		return
	}
	defer onFinish(nil)

	c.Requested++

	server := p.(*Peer).Server
	res := make(chan Response)

	select {
	case <-done:
		return
	case <-ctx.Done():
		c.TimedOut++
		return
	case server.Inbox <- Request{
		Response: res,
	}:
	default:
		c.Dropped++
	}

	select {
	case <-done:
		return
	case <-ctx.Done():
		c.TimedOut++
	case <-res:
		c.Responded++
	}
}
