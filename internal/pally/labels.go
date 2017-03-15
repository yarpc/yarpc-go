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

package pally

import (
	"errors"
	"sync"
)

var (
	// Match the Prometheus error message.
	errInconsistentCardinality = errors.New("inconsistent label cardinality")

	_digesterPool = sync.Pool{New: func() interface{} {
		return &digester{make([]byte, 0, 128)}
	}}
)

// A digester creates a null-delimited byte slice from a series of variable
// label values. It's an efficient way to create map keys from metric names and
// labels.
type digester struct {
	bs []byte
}

// For optimal performance, be sure to free each digester.
func newDigester() *digester {
	d := _digesterPool.Get().(*digester)
	d.bs = d.bs[:0]
	return d
}

func (d *digester) add(s string) {
	if len(d.bs) > 0 {
		// separate labels with a null byte
		d.bs = append(d.bs, '\x00')
	}
	d.bs = append(d.bs, s...)
}

func (d *digester) digest() []byte {
	return d.bs
}

func (d *digester) free() {
	_digesterPool.Put(d)
}

// Labels describe the dimensions of a metric.
type Labels map[string]string
