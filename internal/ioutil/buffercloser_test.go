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

package ioutil

import (
	"bytes"
	"io"
	"math/rand"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuffers(t *testing.T) {
	var wg sync.WaitGroup
	for g := 0; g < 10; g++ {
		wg.Add(1)
		go func() {
			for i := 0; i < 100; i++ {
				buf := NewBufferCloser()
				b := make([]byte, 1)
				n, readErr := buf.Read(b)
				assert.Equal(t, 0, n, "expected empty buffer")
				assert.Equal(t, io.EOF, readErr, "expected empty buffer")

				bytesOfNoise := make([]byte, rand.Intn(5000))
				_, err := rand.Read(bytesOfNoise)
				assert.NoError(t, err, "Unexpected error from rand.Read")
				_, err = buf.ReadFrom(bytes.NewReader(bytesOfNoise))
				assert.NoError(t, err, "Unexpected error from buf.ReadFrom")

				if i%2 == 0 {
					actualBytes := make([]byte, len(bytesOfNoise))
					num, err := buf.Read(actualBytes)
					assert.NoError(t, err, "unexpected error reading from buffer")
					assert.Equal(t, len(bytesOfNoise), num, "wrong number of bytes read")
					assert.Equal(t, string(bytesOfNoise), string(actualBytes), "bytes read did not match")
				} else {
					actualBuf := &bytes.Buffer{}
					num, err := buf.WriteTo(actualBuf)
					assert.NoError(t, err, "unexpected error writing to other buffer")
					assert.Equal(t, len(bytesOfNoise), int(num), "wrong number of bytes written")
					assert.Equal(t, string(bytesOfNoise), string(actualBuf.Bytes()), "bytes written did not match")
				}

				buf.Close()
			}
			wg.Done()
		}()
	}
	wg.Wait()
}
