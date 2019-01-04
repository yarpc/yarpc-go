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

package lifecycletest

import "go.uber.org/yarpc/pkg/lifecycle"

// NewNop returns a new one-time no-op lifecycle.
func NewNop() *Nop {
	return &Nop{once: lifecycle.NewOnce()}
}

// Nop is a no-op implementation of a lifecycle Once. It advances state but
// performs no actions.
type Nop struct {
	once *lifecycle.Once
}

// Start advances the Nop to Running without side-effects.
func (n *Nop) Start() error {
	return n.once.Start(nil)
}

// Stop advances the Nop to Stopped without side-effects.
func (n *Nop) Stop() error {
	return n.once.Stop(nil)
}

// IsRunning returns the Nop lifecycle.Status.
func (n *Nop) IsRunning() bool {
	return n.once.IsRunning()
}
