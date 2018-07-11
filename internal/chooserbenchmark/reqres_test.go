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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListeners(t *testing.T) {
	listeners := NewListeners(10)
	assert.Equal(t, 10, cap(listeners))
	assert.Equal(t, 10, len(listeners))
	_, err := listeners.Listener(-1)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "pid index out of range, pid")
	_, err = listeners.Listener(10)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "pid index out of range, pid")
	_, err = listeners.Listener(0)
	assert.NoError(t, err)
	_, err = listeners.Listener(9)
	assert.NoError(t, err)
}

func TestSendRecvMessage(t *testing.T) {
	listeners := NewListeners(10)
	lis, err := listeners.Listener(0)
	assert.NoError(t, err)
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func(wg *sync.WaitGroup) {
		req := Request{channel: make(chan Response), clientID: 1}
		lis <- req
		res := <-req.channel
		assert.Equal(t, 0, res.serverID)
		wg.Done()
	}(&wg)
	req := <-lis
	assert.Equal(t, 1, req.clientID)
	req.channel <- Response{serverID: 0}
	wg.Wait()
}
