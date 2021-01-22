// Copyright (c) 2021 Uber Technologies, Inc.
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

package bufferpool

import (
	"bytes"
	"io"
	"io/ioutil"
	"math/rand"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBufferWrite(t *testing.T) {
	runTestWithBuffer(t, func(t *testing.T, buf *Buffer) {
		buf.Write([]byte("hello world"))
		assert.Equal(t, "hello world", string(buf.Bytes()), "Unexpected written bytes")
	})
}

func TestBufferWriteTo(t *testing.T) {
	runTestWithBuffer(t, func(t *testing.T, buf *Buffer) {
		buf.Write([]byte("hello world"))

		sink := &bytes.Buffer{}
		buf.WriteTo(sink)
		assert.Equal(t, "hello world", sink.String(), "Unexpected written bytes")
	})
}

func TestBufferRead(t *testing.T) {
	runTestWithBuffer(t, func(t *testing.T, buf *Buffer) {
		io.WriteString(buf, "hello world")

		got, err := ioutil.ReadAll(buf)
		require.NoError(t, err, "Read failed")
		assert.Equal(t, "hello world", string(got), "Unexpected read bytes")
	})
}

func TestBufferReadFrom(t *testing.T) {
	runTestWithBuffer(t, func(t *testing.T, buf *Buffer) {
		_, err := buf.ReadFrom(strings.NewReader("hello world"))
		require.NoError(t, err, "ReadFrom failed")

		assert.Equal(t, "hello world", string(buf.Bytes()), "Unexpected read bytes")
	})
}

func TestBufferPrePostOp(t *testing.T) {
	runTest(t, func(t *testing.T, pool *Pool) {
		buf := pool.Get()
		defer buf.Release()

		v := buf.preOp()
		assert.NotPanics(t, func() {
			buf.postOp(v)
		})

		// Doing the postOp twice will panic
		assert.Panics(t, func() {
			buf.postOp(v)
		})
	})
}

func TestBufferReuse(t *testing.T) {
	runTest(t, func(t *testing.T, pool *Pool) {
		runConcurrently(t, func() {
			buf := pool.Get()
			assert.Equal(t, 0, len(buf.Bytes()), "Expected zero buffer size")

			io.WriteString(buf, "test")
			buf.Release()
		})
	})
}

func TestBufferUseAfterRelease(t *testing.T) {
	runTest(t, func(t *testing.T, pool *Pool) {
		buf := pool.Get()
		buf.Release()

		assert.Panics(t, func() {
			io.WriteString(buf, "test")
		})
	})
}

func TestBufferReleaseTwice(t *testing.T) {
	runTest(t, func(t *testing.T, pool *Pool) {
		buf := pool.Get()

		buf.Release()
		assert.Panics(t, func() {
			buf.Release()
		})
	})
}

func TestBuffers(t *testing.T) {
	runConcurrently(t, func() {
		buf := Get()
		assert.Zero(t, buf.Len(), "Expected truncated buffer")

		bs := randBytes(rand.Intn(5000))
		_, err := rand.Read(bs)
		assert.NoError(t, err, "Unexpected error from rand.Read")
		_, err = buf.Write(bs)
		assert.NoError(t, err, "Unexpected error from buffer.Write")

		assert.Equal(t, buf.Len(), len(bs), "Expected same buffer size")

		Put(buf)
	})
}

func runTestWithBuffer(t *testing.T, f func(t *testing.T, buf *Buffer)) {
	runTest(t, func(t *testing.T, pool *Pool) {
		buf := pool.Get()
		defer buf.Release()

		f(t, buf)
	})
}

// runTest runs a given test function with pools created using
// different options.
// It also runs the test multiple times to trigger buffer reuse.
func runTest(t *testing.T, f func(t *testing.T, pool *Pool)) {
	const numIterations = 10

	t.Run("no use-after-free detection", func(t *testing.T) {
		for i := 0; i < numIterations; i++ {
			f(t, NewPool())
		}
	})

	t.Run("with use-after-free detection", func(t *testing.T) {
		for i := 0; i < numIterations; i++ {
			f(t, NewPool(DetectUseAfterFreeForTests()))
		}
	})
}

func runConcurrently(t *testing.T, f func()) {
	const numGoroutines = 5

	var wg sync.WaitGroup
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for j := 0; j < 10; j++ {
				f()
			}
		}()
	}

	wg.Wait()
}

func randBytes(n int) []byte {
	buf := make([]byte, n)
	rand.Read(buf)
	return buf
}
