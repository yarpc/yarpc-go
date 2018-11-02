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

package internaliopool

import (
	"bytes"
	"math/rand"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuffers(t *testing.T) {
	var wg sync.WaitGroup
	const parallel = 10
	const serial = 100
	wg.Add(parallel)
	for g := 0; g < parallel; g++ {
		go func() {
			for i := 0; i < serial; i++ {
				inputBytes := make([]byte, rand.Intn(5000)+20)
				_, err := rand.Read(inputBytes)
				if !assert.NoError(t, err, "Unexpected error from rand.Read") {
					reader := bytes.NewReader(inputBytes)

					outputBytes := make([]byte, 0, len(inputBytes))
					writer := bytes.NewBuffer(outputBytes)

					copyLength, err := Copy(writer, reader)
					assert.NoError(t, err)
					assert.Equal(t, copyLength, len(inputBytes))
				}
			}
			wg.Done()
		}()
	}
	wg.Wait()
}
