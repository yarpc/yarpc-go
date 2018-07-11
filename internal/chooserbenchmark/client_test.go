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
	"github.com/stretchr/testify/require"
	"go.uber.org/atomic"
	"go.uber.org/yarpc/api/peer"
	"go.uber.org/yarpc/peer/roundrobin"
)

func TestClient(t *testing.T) {
	// initiate parameters for NewClient
	clientGroup := &ClientGroup{
		Name:        "roundrobin",
		Count:       1,
		RPS:         10000,
		Constructor: func(t peer.Transport) peer.ChooserList { return roundrobin.New(t) },
	}
	listeners := NewListeners(1)
	clientStart, clientStop, serverStop := make(chan struct{}), make(chan struct{}), make(chan struct{})
	var wg = sync.WaitGroup{}

	// create a new client and start the peer list chooser
	client := NewClient(0, clientGroup, listeners, clientStart, clientStop, &wg)

	// issue failure when chooser not started
	err := client.issue()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "peer list is not running")

	client.chooser.Start()
	lis, err := listeners.Listener(0)
	assert.NoError(t, err)

	reqCounter := atomic.Int64{}
	// start client go routine
	go client.Start()
	// start server go routine
	go func(lis Listener) {
		for {
			select {
			case req := <-lis:
				close(req.channel)
				reqCounter.Inc()
			case <-serverStop:
				wg.Done()
				return
			}
		}
	}(lis)
	assert.Equal(t, int64(0), reqCounter.Load(), "shouldn't receive request before client start")

	// start client
	close(clientStart)
	time.Sleep(time.Millisecond * 10)

	// stop client
	wg.Add(1)
	close(clientStop)
	wg.Wait()

	// stop server
	wg.Add(1)
	close(serverStop)
	wg.Wait()

	clientResCounter1 := client.resCounter.Load()
	serverReqCounter1 := reqCounter.Load()
	assert.True(t, clientResCounter1 <= serverReqCounter1,
		"received responses in client should be less than received requests in server, resCounter: %v, reqCounter: %v", clientResCounter1, serverReqCounter1)
	time.Sleep(10 * time.Millisecond)
	clientResCounter2 := client.resCounter.Load()
	serverReqCounter2 := reqCounter.Load()
	assert.Equal(t, clientResCounter1, clientResCounter2)
	assert.Equal(t, serverReqCounter1, serverReqCounter2)
	// close listener
	close(lis)
}
