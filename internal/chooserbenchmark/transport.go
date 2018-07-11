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
	"strconv"

	"go.uber.org/yarpc/api/peer"
)

var _ peer.Transport = (*Transport)(nil)

// Transport is a fake transport that only retain peers and release peers
// requests and responses are sent over go channels, not real network traffic
type Transport struct{}

// NewTransport returns a bench transport
func NewTransport() *Transport {
	return &Transport{}
}

// RetainPeer returns a bench peer
func (t *Transport) RetainPeer(id peer.Identifier, ps peer.Subscriber) (peer.Peer, error) {
	i, _ := strconv.Atoi(id.Identifier())
	return NewPeer(i, ps), nil
}

// ReleasePeer does nothing
func (t *Transport) ReleasePeer(id peer.Identifier, ps peer.Subscriber) error {
	return nil
}
