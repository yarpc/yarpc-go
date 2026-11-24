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

//go:build !go1.24

package http

import (
	"net"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc/internal/http2test"
	"go.uber.org/yarpc/internal/testtime"
	"golang.org/x/net/http2"
)

// TestInboundH2CGracefulShutdown stops an inbound with a call in flight on an
// HTTP/2 cleartext (h2c) connection, then drops every server-side connection
// the moment Stop returns, as the process exiting would.
//
// BUG: Stop does not wait for h2c connections. h2c.NewHandler hijacks them,
// and http.Server.Shutdown neither waits for hijacked connections nor for the
// HTTP/2 shutdown hook that sends GOAWAY, which it runs in a goroutine. Stop
// therefore returns with the call still in flight, the GOAWAY races the
// process exit and often loses, and the client sees its connection drop under
// a request the server never answered.
func TestInboundH2CGracefulShutdown(t *testing.T) {
	conns := &serverConns{}
	inbound, h := startBlockingInbound(t, conns.record())
	addr := inbound.Addr().String()

	c := http2test.DialRawH2C(t, addr)
	c.OpenStream(t, 1, "/", h2cRequestHeaders("block"), []byte("payload"))
	h.awaitEntered(t)

	stopErr := make(chan error, 1)
	go func() { stopErr <- inbound.Stop() }()
	select {
	case err := <-stopErr:
		require.NoError(t, err)
	case <-time.After(5 * testtime.Second):
		require.FailNow(t, "expected Stop to return without waiting for the in-flight h2c call")
	}
	conns.closeAll()

	frames := c.ReadUntilClosed(t, 5*testtime.Second)
	assert.False(t, http2test.ResponseOn(frames, 1).Complete,
		"expected in-flight stream 1 to be lost when the process exits after Stop")
	goAways := 0
	for _, f := range frames {
		if http2test.IsGoAway(f) {
			goAways++
		}
	}
	// Not asserted: whether the GOAWAY won the race against the exit is
	// nondeterministic, which is itself part of the bug.
	t.Logf("GOAWAY frames delivered before the connection dropped: %d", goAways)

	_, err := net.Dial("tcp", addr)
	assert.Error(t, err, "new connections must be refused once Stop returns")
}

// TestInboundH2CStopWithWedgedHandler stops an inbound whose h2c connection
// has a call that never completes.
//
// BUG: Stop returns immediately instead of waiting up to ShutdownTimeout, and
// leaves the connection open and served behind it. The GOAWAY does arrive
// eventually, sent by the shutdown hook that Stop does not wait for: the
// HTTP/2 shutdown wiring exists, its synchronization with Stop does not.
func TestInboundH2CStopWithWedgedHandler(t *testing.T) {
	timeout := time.Minute
	inbound, h := startBlockingInbound(t, ShutdownTimeout(timeout))

	c := http2test.DialRawH2C(t, inbound.Addr().String())
	c.OpenStream(t, 1, "/", h2cRequestHeaders("block"), []byte("payload"))
	h.awaitEntered(t)

	start := time.Now()
	require.NoError(t, inbound.Stop())
	assert.Less(t, time.Since(start), timeout,
		"expected Stop to return without waiting for ShutdownTimeout")

	frames := c.ReadUntil(t, 5*testtime.Second, http2test.IsGoAway)
	goAway := frames[len(frames)-1]
	assert.Equal(t, http2.ErrCodeNo, goAway.ErrCode, "graceful shutdown must use NO_ERROR")
	assert.Equal(t, uint32(1), goAway.LastStreamID, "GOAWAY must cover in-flight stream 1")

	c.Ping(t)
	frames = append(frames, c.ReadUntil(t, 5*testtime.Second, http2test.IsPingAck)...)
	assert.False(t, http2test.ResponseOn(frames, 1).Complete,
		"stream 1 completed although its handler never returned")
}

// serverConns records the connections an inbound accepts, so that a test can
// drop them all at once, the way exiting the process would.
type serverConns struct {
	mu    sync.Mutex
	conns []net.Conn
}

func (s *serverConns) record() InboundOption {
	return func(i *Inbound) {
		i.server.ConnState = func(c net.Conn, state http.ConnState) {
			if state == http.StateNew {
				s.mu.Lock()
				s.conns = append(s.conns, c)
				s.mu.Unlock()
			}
		}
	}
}

func (s *serverConns) closeAll() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, c := range s.conns {
		_ = c.Close()
	}
}
