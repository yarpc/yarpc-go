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
	"context"

	"go.uber.org/yarpc/api/peer"
	"go.uber.org/yarpc/api/transport"
)

var _ transport.UnaryOutbound = (*Outbound)(nil)

// Outbound is a transport.UnaryOutbound.
type Outbound struct {
	chooser peer.Chooser
}

// NewOutbound returns a new Outbound for the given chooser.
func NewOutbound(chooser peer.Chooser) (*Outbound, error) {
	return nil, nil
}

// Start implements transport.Lifecycle#Start.
func (o *Outbound) Start() error {
	return nil
}

// Stop implements transport.Lifecycle#Stop.
func (o *Outbound) Stop() error {
	return nil
}

// IsRunning implements transport.Lifecycle#IsRunning.
func (o *Outbound) IsRunning() bool {
	return false
}

// Transports implements transport.Inbound#Transports.
func (o *Outbound) Transports() []transport.Transport {
	return nil
}

// Call implements transport.UnaryOutbound#Call.
func (o *Outbound) Call(ctx context.Context, request *transport.Request) (*transport.Response, error) {
	return nil, nil
}
