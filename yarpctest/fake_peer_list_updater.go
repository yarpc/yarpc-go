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
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/pkg/lifecycletest"
)

// FakePeerListUpdaterOption is an option for NewFakePeerListUpdater.
type FakePeerListUpdaterOption func(*FakePeerListUpdater)

// Watch is a fake option for NewFakePeerListUpdater that enables "watch". It's fake.
func Watch(u *FakePeerListUpdater) {
	u.watch = true
}

// UpdaterNop is a fake option for NewFakePeerListUpdater that sets a nop var. It's fake.
func UpdaterNop(nop string) func(*FakePeerListUpdater) {
	return func(u *FakePeerListUpdater) {
		u.nop = nop
	}
}

// FakePeerListUpdater is a fake peer list updater.  It doesn't actually update
// a peer list.
type FakePeerListUpdater struct {
	transport.Lifecycle
	watch bool
	nop   string
}

// NewFakePeerListUpdater returns a new FakePeerListUpdater, applying any
// passed options.
func NewFakePeerListUpdater(opts ...FakePeerListUpdaterOption) *FakePeerListUpdater {
	u := &FakePeerListUpdater{
		Lifecycle: lifecycletest.NewNop(),
	}
	for _, opt := range opts {
		opt(u)
	}
	return u
}

// Watch returns whether the peer list updater was configured to "watch". It is
// fake.
func (u *FakePeerListUpdater) Watch() bool {
	return u.watch
}

// Nop returns the nop variable.
func (u *FakePeerListUpdater) Nop() string {
	return u.nop
}
