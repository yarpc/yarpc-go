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

package yarpcdialer

import (
	"fmt"

	"go.uber.org/yarpc/v2"
)

var _ yarpc.DialerProvider = (*Provider)(nil)

// Provider implements yarpc.DialerProvider.
type Provider struct {
	dialers map[string]yarpc.Dialer
}

// NewProvider returns a new DialerProvider.
func NewProvider() *Provider {
	return &Provider{
		dialers: make(map[string]yarpc.Dialer),
	}
}

// Dialer returns a named yarpc.Dialer.
func (p *Provider) Dialer(name string) (yarpc.Dialer, bool) {
	c, ok := p.dialers[name]
	return c, ok
}

// Register registers a yarpc.Dialer to the given name.
func (p *Provider) Register(name string, dialer yarpc.Dialer) error {
	if _, ok := p.dialers[name]; ok {
		return fmt.Errorf("dialer %q is already registered", name)
	}
	p.dialers[name] = dialer
	return nil
}
