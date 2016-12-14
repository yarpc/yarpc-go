// Copyright (c) 2016 Uber Technologies, Inc.
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

package redispubsub

import (
	"sync"
	"testing"

	"go.uber.org/yarpc/api/transport/transporttest"
	"go.uber.org/yarpc/transport/x/redispubsub/redistest"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOperations(t *testing.T) {
	channel := "redis-channel"

	mockCtrl := gomock.NewController(t)
	client := redistest.NewMockClient(mockCtrl)

	startCall := client.EXPECT().Start()
	subCall := client.EXPECT().
		Subscribe(channel, gomock.Any()).
		After(startCall)
	client.EXPECT().
		Stop().
		After(subCall)

	inbound := NewInbound(client, channel)
	inbound.SetRegistry(&transporttest.MockRegistry{})

	assert.Equal(t, channel, inbound.channel)

	err := inbound.Start()
	assert.NoError(t, err, "error starting redis inbound")

	err = inbound.Stop()
	assert.NoError(t, err, "error stopping redis inbound")
}

func TestStartInboundMultiple(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	client := redistest.NewMockClient(mockCtrl)
	channel := "redis-channel"

	client.EXPECT().Start().Times(1).Return(nil)
	client.EXPECT().Subscribe(channel, gomock.Any()).Times(1)
	client.EXPECT().Stop().Times(1).Return(nil)

	in := NewInbound(client, channel)
	in.SetRegistry(&transporttest.MockRegistry{})

	var wg sync.WaitGroup
	signal := make(chan struct{})

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-signal

			err := in.Start()
			assert.NoError(t, err)
		}()
	}
	close(signal)
	wg.Wait()
}

func TestStopInboundMultiple(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	client := redistest.NewMockClient(mockCtrl)
	channel := "redis-channel"

	client.EXPECT().Start().Times(1).Return(nil)
	client.EXPECT().Subscribe(channel, gomock.Any()).Times(1)
	client.EXPECT().Stop().Times(1).Return(nil)

	in := NewInbound(client, channel)
	in.SetRegistry(&transporttest.MockRegistry{})

	err := in.Start()
	require.NoError(t, err)

	var wg sync.WaitGroup
	signal := make(chan struct{})

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-signal

			err := in.Stop()
			assert.NoError(t, err)
		}()
	}
	close(signal)
	wg.Wait()
}
