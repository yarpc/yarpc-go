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
	"time"
)

// Server receives request from clients, sleep for a random latency and give
// response back to clients
type Server struct {
	// identifiers
	groupName string
	id        int

	// metrics
	counter int64

	// random latency generator
	latency *LogNormalLatency

	// end point of this server
	listener Listener

	start chan struct{}
	stop  chan struct{}
	wg    *sync.WaitGroup
}

// NewServer creates a new server
func NewServer(
	id int,
	groupName string,
	latency time.Duration,
	sigma float64,
	lis Listener,
	start, stop chan struct{},
	wg *sync.WaitGroup,
) (*Server, error) {
	return &Server{
		groupName: groupName,
		id:        id,
		listener:  lis,
		latency:   NewLogNormalLatency(latency, sigma),
		start:     start,
		stop:      stop,
		wg:        wg,
	}, nil
}

func (s *Server) handle(req Request) {
	time.Sleep(s.latency.Random())
	req.channel <- Response{serverID: s.id}
	close(req.channel)
}

// Serve is the long-run go routine receives requests
func (s *Server) Serve() {
	<-s.start
	s.wg.Done()
	for {
		select {
		case req := <-s.listener:
			s.counter++
			go s.handle(req)
		case <-s.stop:
			close(s.listener)
			s.wg.Done()
			return
		}
	}
}
