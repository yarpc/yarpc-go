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

//go:build go1.24

package http

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc/internal/http2test"
	"go.uber.org/yarpc/internal/testtime"
	"golang.org/x/net/http2"
)

// TestInboundH2CGracefulShutdown stops an inbound with a call in flight on an
// HTTP/2 cleartext (h2c) connection and expects Stop to drain it.
func TestInboundH2CGracefulShutdown(t *testing.T) {
	inbound, h := startBlockingInbound(t)
	addr := inbound.Addr().String()

	c := http2test.DialRawH2C(t, addr)
	c.OpenStream(t, 1, "/", h2cRequestHeaders("block"), []byte("payload"))
	h.awaitEntered(t)

	stopErr := make(chan error, 1)
	go func() { stopErr <- inbound.Stop() }()

	frames := c.ReadUntil(t, 5*testtime.Second, http2test.IsGoAway)
	goAway := frames[len(frames)-1]
	assert.Equal(t, http2.ErrCodeNo, goAway.ErrCode, "graceful shutdown must use NO_ERROR")
	assert.Equal(t, uint32(1), goAway.LastStreamID, "GOAWAY must cover in-flight stream 1")

	select {
	case err := <-stopErr:
		require.FailNow(t, "Stop returned while the h2c call was still in flight", "err: %v", err)
	default:
	}

	h.releaseAll()
	frames = append(frames, c.ReadUntil(t, 5*testtime.Second, http2test.StreamDone(1))...)
	res := http2test.ResponseOn(frames, 1)
	assert.True(t, res.Complete, "in-flight stream 1 must complete")
	assert.False(t, res.Reset, "in-flight stream 1 must not be reset")
	assert.Equal(t, "payload", string(res.Body))

	select {
	case err := <-stopErr:
		assert.NoError(t, err)
	case <-time.After(5 * testtime.Second):
		require.FailNow(t, "Stop did not return after the h2c call drained")
	}

	_, err := net.Dial("tcp", addr)
	assert.Error(t, err, "new connections must be refused once Stop returns")
}

// TestInboundH2CStopWithWedgedHandler stops an inbound whose h2c connection
// has a call that never completes, and expects Stop to give up at ShutdownTimeout.
func TestInboundH2CStopWithWedgedHandler(t *testing.T) {
	timeout := 200 * testtime.Millisecond
	inbound, h := startBlockingInbound(t, ShutdownTimeout(timeout))

	c := http2test.DialRawH2C(t, inbound.Addr().String())
	c.OpenStream(t, 1, "/", h2cRequestHeaders("block"), []byte("payload"))
	h.awaitEntered(t)

	start := time.Now()
	stopErr := make(chan error, 1)
	go func() { stopErr <- inbound.Stop() }()

	frames := c.ReadUntil(t, 5*testtime.Second, http2test.IsGoAway)
	goAway := frames[len(frames)-1]
	assert.Equal(t, http2.ErrCodeNo, goAway.ErrCode, "graceful shutdown must use NO_ERROR")
	assert.Equal(t, uint32(1), goAway.LastStreamID, "GOAWAY must cover in-flight stream 1")

	select {
	case err := <-stopErr:
		assert.ErrorIs(t, err, context.DeadlineExceeded)
		assert.GreaterOrEqual(t, time.Since(start), timeout, "Stop must wait for ShutdownTimeout")
	case <-time.After(5 * testtime.Second):
		require.FailNow(t, "Stop did not return after ShutdownTimeout")
	}
	assert.False(t, http2test.ResponseOn(frames, 1).Complete,
		"stream 1 completed although its handler never returned")
}
