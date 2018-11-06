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

package yarpcchooser

import (
	"fmt"

	"go.uber.org/yarpc/v2"
)

var _ yarpc.ChooserProvider = (*Provider)(nil)

// Provider implements yarpc.ChooserProvider.
type Provider struct {
	choosers map[string]yarpc.Chooser
}

// NewProvider returns a new ChooserProvider.
func NewProvider() *Provider {
	return &Provider{
		choosers: make(map[string]yarpc.Chooser),
	}
}

// Chooser returns a named yarpc.Chooser.
func (p *Provider) Chooser(name string) (yarpc.Chooser, bool) {
	c, ok := p.choosers[name]
	return c, ok
}

// Register registers a yarpc.Chooser to the given name.
func (p *Provider) Register(name string, chooser yarpc.Chooser) error {
	if _, ok := p.choosers[name]; ok {
		return fmt.Errorf("chooser %q is already registered", name)
	}
	p.choosers[name] = chooser
	return nil
}
