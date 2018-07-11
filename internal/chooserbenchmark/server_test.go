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

package chooserbenchmark

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/atomic"
)

func TestServer(t *testing.T) {
	lis := make(chan Request)
	start, stop := make(chan struct{}), make(chan struct{})
	issue, received := make(chan struct{}), make(chan struct{})
	wg := sync.WaitGroup{}
	server, err := NewServer(0, "fast", time.Millisecond*15, DefaultLogNormalSigma, lis, start, stop, &wg)
	assert.NoError(t, err)
	resCounter := atomic.Int64{}
	go server.Serve()
	go func(lis Listener) {
		for {
			select {
			case <-stop:
				wg.Done()
				return
			case <-issue:
				go func(lis Listener) {
					req := Request{channel: make(chan Response)}
					lis <- req
					res := <-req.channel
					resCounter.Inc()
					assert.Equal(t, 0, res.serverID)
					received <- struct{}{}
				}(lis)
			}
		}
	}(lis)
	wg.Add(1)
	close(start)
	wg.Wait()
	issue <- struct{}{}
	time.Sleep(time.Millisecond * 3)
	wg.Add(2)
	close(stop)
	wg.Wait()
	<-received
	assert.Equal(t, int64(1), resCounter.Load())
	close(issue)
}
