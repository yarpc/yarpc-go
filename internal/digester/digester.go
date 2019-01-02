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

package digester

import "sync"

var _digesterPool = sync.Pool{New: func() interface{} {
	return &Digester{make([]byte, 0, 128)}
}}

// Digester creates a null-delimited byte slice from a series of strings. It's
// an efficient way to create map keys.
//
// This helps because (1) appending to a string allocates and (2) converting a
// byte slice to a string allocates, but (3) the Go compiler optimizes away
// byte-to-string conversions in map lookups. Using this type to build up a key
// and doing map lookups with myMap[string(d.digest())] is fast and
// zero-allocation.
type Digester struct {
	bs []byte
}

// New creates a new Digester.
// For optimal performance, be sure to call "Free" on each digester.
func New() *Digester {
	d := _digesterPool.Get().(*Digester)
	d.bs = d.bs[:0]
	return d
}

// Add adds a string to the digester slice.
func (d *Digester) Add(s string) {
	if len(d.bs) > 0 {
		// separate labels with a null byte
		d.bs = append(d.bs, '\x00')
	}
	d.bs = append(d.bs, s...)
}

// Digest returns a map key for the digester.
func (d *Digester) Digest() []byte {
	return d.bs
}

// Free is called to indicate that the digester can be returned to the pool to
// be used in another context.
func (d *Digester) Free() {
	_digesterPool.Put(d)
}
