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
	"fmt"
)

// Response is what servers give back to clients
type Response struct {
	// current not used, but is necessary for fine-grained metrics
	serverID int
}

// Request is what clients send to servers, request is a channel of channel,
// since we need a bi-direction connection between servers and clients.
// server will wait on Request channel, client will wait on Response channel
type Request struct {
	channel chan Response
	// same as serverId, is for fine-grained metrics
	clientID int
}

// Listener is like an end point in real world, listening requests from clients
type Listener chan Request

// Listeners keeps a list of go channels as end points that receive requests
// from clients, it's a shared object among all go routines
type Listeners []Listener

// NewListeners makes n go channels and returns it as Listeners object
func NewListeners(n int) Listeners {
	listeners := make([]Listener, n)
	for i := 0; i < n; i++ {
		listeners[i] = make(Listener)
	}
	return Listeners(listeners)
}

// Listener return the Listener object with corresponding peer id
func (sg Listeners) Listener(pid int) (Listener, error) {
	if pid < 0 || pid >= len(sg) {
		return nil, fmt.Errorf("pid index out of range, pid: %d size: %d", pid, len(sg))
	}
	return sg[pid], nil
}
