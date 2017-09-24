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

package throttle

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/internal/testtime"
	"go.uber.org/yarpc/yarpctest"
)

func TestThrottleMiddleware(t *testing.T) {
	nop := func(ctx context.Context, req *transport.Request) (*transport.Response, error) {
		return nil, nil
	}
	trans := yarpctest.NewFakeTransport()
	out := trans.NewOutbound(yarpctest.NewFakePeerList(), yarpctest.OutboundCallOverride(nop))
	out.Start()
	defer out.Stop()

	mw := NewUnaryMiddleware(WithRate(1), WithBurstLimit(1))
	defer mw.Stop()

	call := func() (time.Duration, error) {
		begin := time.Now()
		ctx := context.Background()
		ctx, cancel := context.WithTimeout(ctx, 10*time.Millisecond)
		defer cancel()
		_, err := mw.Call(ctx, nil, out)
		end := time.Now()
		return end.Sub(begin), err
	}

	// One request passes in the second it takes to run this test, due to the
	// burst limit of 1.
	latency, err := call()
	assert.True(t, latency < 40*testtime.Millisecond)
	assert.Nil(t, err)

	assert.Equal(t, int64(1), mw.metrics.passes.Load())

	// Having exceeded our burst limit in no appreciable time, the second call
	// drops because it can't conceivably run before the deadline.
	latency, err = call()
	assert.True(t, latency < 40*testtime.Millisecond)
	assert.NotNil(t, err)

	assert.Equal(t, int64(1), mw.metrics.drops.Load())
}
