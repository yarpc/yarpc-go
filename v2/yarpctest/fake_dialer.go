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

package yarpctest

import (
	"fmt"

	yarpc "go.uber.org/yarpc/v2"
)

// FakeDialer is a fake dialer.
type FakeDialer struct {
	name string
}

var _ yarpc.Dialer = (*FakeDialer)(nil)

// NewFakeDialer returns a fake dialer.
func NewFakeDialer(name string) *FakeDialer {
	return &FakeDialer{
		name: name,
	}
}

// Name returns the fake List's name.
func (d *FakeDialer) Name() string { return d.name }

// RetainPeer pretends to retain a peer, but actually always returns an error. It's fake.
func (d *FakeDialer) RetainPeer(_ yarpc.Identifier, _ yarpc.Subscriber) (yarpc.Peer, error) {
	return nil, fmt.Errorf(`fake dialer can't actually retain peers`)
}

// ReleasePeer pretends to release a peer.
func (d *FakeDialer) ReleasePeer(_ yarpc.Identifier, _ yarpc.Subscriber) error {
	return nil
}
