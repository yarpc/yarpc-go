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

package redis

import (
	"testing"

	"go.uber.org/yarpc/api/transport/transporttest"
	"go.uber.org/yarpc/internal/testtime"
	"go.uber.org/yarpc/transport/x/redis/redistest"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestOperationOrder(t *testing.T) {
	queueKey, processingKey := "queueKey", "processingKey"
	timeout := testtime.Second

	mockCtrl := gomock.NewController(t)
	client := redistest.NewMockClient(mockCtrl)

	gomock.InOrder(
		client.EXPECT().BRPopLPush(queueKey, processingKey, timeout),
		client.EXPECT().LRem(queueKey, gomock.Any()),
	)

	inbound := NewInbound(client, queueKey, processingKey, timeout)
	inbound.SetRouter(&transporttest.MockRouter{})

	assert.Equal(t, queueKey, inbound.queueKey)
	assert.Equal(t, processingKey, inbound.processingKey)

	// We're specifically testing the ingestion loop; with the current client
	// code, it's extremely difficult to make substantive assertions about the
	// number of messages handled.
	inbound.handle()
}
