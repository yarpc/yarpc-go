// Copyright (c) 2022 Uber Technologies, Inc.
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

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDigester(t *testing.T) {
	const (
		goroutines = 10
		iterations = 100
	)

	expected := []byte{'f', 'o', 'o', 0, 'b', 'a', 'r'}

	var wg sync.WaitGroup
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				d := New()
				defer d.Free()

				assert.Equal(t, 0, len(d.Digest()), "Expected fresh digester to have no internal state.")
				assert.True(t, cap(d.Digest()) > 0, "Expected fresh digester to have available capacity.")

				d.Add("foo")
				d.Add("bar")
				assert.Equal(
					t,
					string(expected),
					string(d.Digest()),
					"Expected digest to be null-separated concatenation of inputs.",
				)
			}
		}()
	}

	wg.Wait()
}
