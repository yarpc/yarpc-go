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

package yarpc

import (
	"context"
)

// Chooser is a collection of Peers. Outbounds request peers from the
// peer.Chooser to determine where to send requests.
type Chooser interface {
	// The name of the chooser.
	Name() string
	// Choose a Peer for the next call, block until a peer is available (or timeout)
	Choose(context.Context, *Request) (peer Peer, onFinish func(error), err error)
}

// ChooserProvider is a registry of pre-configured Choosers.
type ChooserProvider interface {
	Chooser(name string) (Chooser, bool)
}

// List listens to adds and removes of Peers from a peer list updater.
// A Chooser will implement the List interface in order to receive
// updates to the list of Peers it is keeping track of.
type List interface {
	// The name of the list.
	Name() string
	// Update performs the additions and removals to the Peer List.
	Update(updates ListUpdates) error
}

// ListProvider is a registry of pre-configured Lists.
type ListProvider interface {
	List(name string) (List, bool)
}

// ListUpdates specifies the updates to be made to a List
type ListUpdates struct {
	// Additions are the identifiers that should be added to the list
	Additions []Identifier

	// Removals are the identifiers that should be removed to the list
	Removals []Identifier
}

// ChooserList is both a Chooser and a List, useful for expressing both
// capabilities of a single instance.
type ChooserList interface {
	Name() string
	Choose(context.Context, *Request) (peer Peer, onFinish func(error), err error)
	Update(updates ListUpdates) error
}
