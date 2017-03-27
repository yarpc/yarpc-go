// Copyright (c) 2017 Uber Technologies, Inc.
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

package grpc

import (
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/internal/sync"
)

var _ transport.Inbound = (*Inbound)(nil)

// Inbound is a grpc transport.Inbound.
type Inbound struct {
	once    sync.LifecycleOnce
	address string
	router  transport.Router
}

// NewInbound returns a new Inbound for the given address.
func NewInbound(address string) (*Inbound, error) {
	return &Inbound{
		sync.Once(),
		address,
		nil,
	}, nil
}

// Start implements transport.Lifecycle#Start.
func (i *Inbound) Start() error {
	return i.once.Start(i.start)
}

// Stop implements transport.Lifecycle#Stop.
func (i *Inbound) Stop() error {
	return i.once.Stop(i.stop)
}

// IsRunning implements transport.Lifecycle#IsRunning.
func (i *Inbound) IsRunning() bool {
	return i.IsRunning()
}

// SetRouter implements transport.Inbound#SetRouter.
func (i *Inbound) SetRouter(router transport.Router) {}

// Transports implements transport.Inbound#Transports.
func (i *Inbound) Transports() []transport.Transport {
	return []transport.Transport{}
}

func (i *Inbound) start() error {
	return nil
}

func (i *Inbound) stop() error {
	return nil
}
