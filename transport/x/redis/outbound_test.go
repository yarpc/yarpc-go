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

package redis

import (
	"bytes"
	"context"
	"sync"
	"testing"

	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/encoding/raw"
	"go.uber.org/yarpc/transport/x/redis/redistest"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCall(t *testing.T) {
	queueKey := "queueKey"
	mockCtrl := gomock.NewController(t)
	client := redistest.NewMockClient(mockCtrl)

	client.EXPECT().Start()
	client.EXPECT().LPush(queueKey, gomock.Any())
	client.EXPECT().Stop()

	out := NewOnewayOutbound(client, queueKey)
	assert.Equal(t, queueKey, out.queueKey)

	err := out.Start()
	assert.NoError(t, err, "could not start redis outbound")

	ack, err := out.CallOneway(context.Background(), &transport.Request{
		Caller:    "caller",
		Service:   "service",
		Encoding:  raw.Encoding,
		Procedure: "hello",
		Body:      bytes.NewReader([]byte("hello!")),
	})
	assert.NotNil(t, ack)
	assert.NoError(t, err)

	assert.NoError(t, out.Stop(), "error stoping redis outbound")
}

func TestStartMultiple(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	client := redistest.NewMockClient(mockCtrl)
	client.EXPECT().Start().Times(1).Return(nil)

	out := NewOnewayOutbound(client, "queueKey")

	var wg sync.WaitGroup
	signal := make(chan struct{})

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-signal

			err := out.Start()
			assert.NoError(t, err)
		}()
	}
	close(signal)
	wg.Wait()
}

func TestStopMultiple(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	client := redistest.NewMockClient(mockCtrl)
	client.EXPECT().Start().Times(1).Return(nil)
	client.EXPECT().Stop().Times(1).Return(nil)

	out := NewOnewayOutbound(client, "queueKey")

	err := out.Start()
	require.NoError(t, err)

	var wg sync.WaitGroup
	signal := make(chan struct{})

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-signal

			err := out.Stop()
			assert.NoError(t, err)
		}()
	}
	close(signal)
	wg.Wait()
}

func TestCallWithoutStarting(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	client := redistest.NewMockClient(mockCtrl)
	client.EXPECT().Start().Times(1).Return(nil)

	out := NewOnewayOutbound(client, "queueKey")

	ack, err := out.CallOneway(
		context.Background(),
		&transport.Request{
			Caller:    "caller",
			Service:   "service",
			Encoding:  raw.Encoding,
			Procedure: "foo",
			Body:      bytes.NewReader([]byte("sup")),
		})

	assert.Nil(t, ack, "ack not nil")
	assert.Error(t, err, "made call")
}
