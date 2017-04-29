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

package cherami

import (
	"testing"

	"go.uber.org/yarpc/api/transport/transporttest"
	"go.uber.org/yarpc/transport/x/cherami/mocks"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestInbound(t *testing.T) {
	mockConsumer := &mocks.Consumer{}
	mockConsumer.On(`Close`)
	mockFactory := &mocks.ClientFactory{}
	mockFactory.On(`GetConsumer`, mock.Anything, mock.Anything).Return(mockConsumer, nil, nil)
	transport := NewTransport(nil)
	inbound := transport.NewInbound(InboundOptions{
		Destination:   `dest`,
		ConsumerGroup: `cg`,
	})
	inbound.setClientFactory(mockFactory)
	inbound.SetRouter(&transporttest.MockRouter{})
	err := inbound.Start()
	assert.Nil(t, err)

	err = inbound.Stop()
	assert.Nil(t, err)
}
