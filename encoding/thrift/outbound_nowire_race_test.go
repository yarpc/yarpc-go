// Copyright (c) 2026 Uber Technologies, Inc.
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

package thrift

import (
	"context"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"go.uber.org/thriftrw/wire"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/api/transport/transporttest"
	"go.uber.org/yarpc/internal/clientconfig"
	"go.uber.org/yarpc/internal/testtime"
)

// TestNoWireClient_ConcurrentBufferPoolReuseRace is a regression/detector
// test for the "YARPC bufferpool" use-after-Put race reported in production
// (Slack "Race 2"): an outbound Thrift NoWire request body backed by a
// pooled buffer can be returned to the pool (via Close) and reused by an
// unrelated, concurrent request's Write while the transport is still
// asynchronously draining the original body's raw bytes (e.g. a slow or
// backpressured HTTP/2 upload). The bug only manifested under concurrent
// load in production, so this test drives many overlapping calls at once
// rather than relying on a single coincidental pool hit.
//
// If buildTransportRequest ever sources its request buffer from bufferpool
// again without otherwise guaranteeing the buffer outlives every reader of
// it, this test must fail under `go test -race`.
func TestNoWireClient_ConcurrentBufferPoolReuseRace(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	trans := transporttest.NewMockUnaryOutbound(mockCtrl)

	var readers sync.WaitGroup
	trans.EXPECT().Call(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, treq *transport.Request) (*transport.Response, error) {
			// Simulate the HTTP/2 client's writeRequestBody: an independent
			// goroutine slowly drains the raw request body bytes over the
			// wire, well after Call() has returned control to its caller.
			readers.Add(1)
			go func(body io.Reader) {
				defer readers.Done()
				one := make([]byte, 1)
				for {
					_, err := body.Read(one)
					if err != nil {
						return
					}
					time.Sleep(10 * time.Microsecond)
				}
			}(treq.Body)

			// Simulate the transport considering the body "done with" and
			// closing it right away (e.g. on a GOAWAY replay or early
			// cancellation) without waiting for the above goroutine to
			// finish physically draining it. If Close() returns the
			// underlying buffer to the pool, this races with the read above
			// and with whatever else the pool hands the buffer to next.
			if closer, ok := treq.Body.(io.Closer); ok {
				closer.Close()
			}

			return &transport.Response{
				Body: io.NopCloser(strings.NewReader(encodeThriftString(t, "response"))),
			}, nil
		},
	).AnyTimes()

	nwc := NewNoWire(Config{
		Service: "MyService",
		ClientConfig: clientconfig.MultiOutbound("caller", "service",
			transport.Outbounds{Unary: trans}),
	})

	ctx, cancel := context.WithTimeout(context.Background(), testtime.Second)
	defer cancel()

	// Fire many concurrent, logically unrelated calls (mirroring "high
	// load" from the incident reports) so the shared bufferpool is under
	// real contention: some goroutine's Get() is likely to reuse a buffer
	// that another goroutine's slow reader (above) is still draining.
	const (
		numGoroutines     = 50
		callsPerGoroutine = 20
	)

	var wg sync.WaitGroup
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < callsPerGoroutine; j++ {
				var br fakeBodyReader
				assert.NoError(t, nwc.Call(ctx, fakeEnveloper(wire.Call), &br))
			}
		}()
	}
	wg.Wait()

	readers.Wait()
}
