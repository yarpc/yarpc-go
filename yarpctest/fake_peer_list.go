// Copyright (c) 2019 Uber Technologies, Inc.
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

package yarpctest

import (
	"context"
	"fmt"

	"go.uber.org/yarpc/api/peer"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/pkg/lifecycletest"
)

// FakePeerListOption is an option for NewFakePeerList.
type FakePeerListOption func(*FakePeerList)

// ListNop is a fake option for NewFakePeerList that sets a nop var. It's fake.
func ListNop(nop string) func(*FakePeerList) {
	return func(u *FakePeerList) {
		u.nop = nop
	}
}

// FakePeerList is a fake peer list.
type FakePeerList struct {
	transport.Lifecycle

	nop string
}

// NewFakePeerList returns a fake peer list.
func NewFakePeerList(opts ...FakePeerListOption) *FakePeerList {
	pl := &FakePeerList{
		Lifecycle: lifecycletest.NewNop(),
	}
	for _, opt := range opts {
		opt(pl)
	}
	return pl
}

// Choose pretends to choose a peer, but actually always returns an error. It's fake.
func (c *FakePeerList) Choose(ctx context.Context, req *transport.Request) (peer.Peer, func(error), error) {
	return nil, nil, fmt.Errorf(`fake peer list can't actually choose peers`)
}

// Update pretends to add or remove peers.
func (c *FakePeerList) Update(up peer.ListUpdates) error {
	return nil
}

// Nop returns the Peer List's nop variable.
func (c *FakePeerList) Nop() string {
	return c.nop
}
