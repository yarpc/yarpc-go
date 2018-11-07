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
	"context"
	"fmt"

	yarpc "go.uber.org/yarpc/v2"
)

// FakePeerChooserOption is an option for NewFakePeerChooser.
type FakePeerChooserOption func(*FakePeerChooser)

// ChooserNop is a fake option for NewFakePeerChooser that sets a nop var. It's fake.
func ChooserNop(nop string) func(*FakePeerChooser) {
	return func(u *FakePeerChooser) {
		u.nop = nop
	}
}

// FakePeerChooser is a fake peer chooser.
type FakePeerChooser struct {
	name string
	nop  string
}

// NewFakePeerChooser returns a fake peer list.
func NewFakePeerChooser(name string, opts ...FakePeerChooserOption) *FakePeerChooser {
	pl := &FakePeerChooser{name: name}
	for _, opt := range opts {
		opt(pl)
	}
	return pl
}

// Name returns the fake Chooser's name.
func (c *FakePeerChooser) Name() string { return c.name }

// Choose pretends to choose a peer, but actually always returns an error. It's fake.
func (c *FakePeerChooser) Choose(ctx context.Context, req *yarpc.Request) (yarpc.Peer, func(error), error) {
	return nil, nil, fmt.Errorf(`fake peer chooser can't actually choose peers`)
}

// Nop returns the Peer Chooser's nop variable.
func (c *FakePeerChooser) Nop() string {
	return c.nop
}
