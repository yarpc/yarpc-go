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

package http

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
	_ "unsafe" // for go:linkname

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/multierr"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/encoding/raw"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

// http2DataChunkPools is the set of pools x/net/http2 takes response body
// chunks from. It is unexported, and no public API can make the read loop
// panic while it buffers a DATA frame, so the test reaches it by name. This
// only links while x/net/http2 has its own client: newer x/net versions hand
// the client to net/http on Go 1.27 and later, where net/http owns the pools.
//
//go:linkname http2DataChunkPools golang.org/x/net/http2.dataChunkPools
var http2DataChunkPools [5]sync.Pool

// TestHTTP2OutboundSurvivesReadLoopPanic reproduces an HTTP/2 client
// connection that stays stuck after its read loop panics.
//
// x/net/http2 holds the connection mutex while it buffers a DATA frame and
// releases it with a plain Unlock. A panic in between leaves the mutex locked,
// and the read loop's deferred cleanup then blocks on that same mutex, so the
// panic never finishes unwinding. The process does not crash, the connection
// is never closed, and everything that needs the mutex blocks: requests in
// flight, new requests (which also block while handling their own deadline),
// ClientConn.State and Close.
//
// The test makes the chunk pools return one value of the wrong type, which is
// the panic seen in production. It does not reproduce what put that value in
// the pool. It then checks what the outbound should still guarantee.
func TestHTTP2OutboundSurvivesReadLoopPanic(t *testing.T) {
	const (
		childEnv    = "YARPC_TEST_H2_READLOOP_PANIC_CHILD"
		callTimeout = time.Second
		callGrace   = time.Second
	)

	if os.Getenv(childEnv) == "" {
		// The stuck connection leaves goroutines blocked for the life of
		// the process, which fails the goroutine leak checks of every later
		// test. Run the scenario in a process of its own.
		cmd := exec.Command(os.Args[0], "-test.run=^"+t.Name()+"$")
		cmd.Env = append(os.Environ(), childEnv+"=1")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%v\n%s", err, out)
		}
		return
	}

	// Shorten the connection health checks so the test does not wait for
	// the defaults.
	origPoll, origReadIdle, origPing := defaultH2ConnHealthPollInterval, defaultHTTP2ReadIdleTimeout, defaultHTTP2PingTimeout
	defaultH2ConnHealthPollInterval = 50 * time.Millisecond
	defaultHTTP2ReadIdleTimeout = 500 * time.Millisecond
	defaultHTTP2PingTimeout = 500 * time.Millisecond
	t.Cleanup(func() {
		defaultH2ConnHealthPollInterval, defaultHTTP2ReadIdleTimeout, defaultHTTP2PingTimeout = origPoll, origReadIdle, origPing
	})
	// Long enough for a read-idle ping to be sent and to time out. The
	// peer's connection health poll runs many times in that window.
	recoveryWindow := defaultHTTP2ReadIdleTimeout + defaultHTTP2PingTimeout + time.Second

	payload := make([]byte, 20<<10)
	server := httptest.NewServer(h2c.NewHandler(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write(payload)
		}),
		&http2.Server{},
	))
	t.Cleanup(server.Close)

	// Two outbounds to one address share a peer and its connection.
	httpTransport := NewTransport()
	outA := httpTransport.NewSingleOutbound(server.URL, UseHTTP2())
	outB := httpTransport.NewSingleOutbound(server.URL, UseHTTP2())
	require.NoError(t, httpTransport.Start())
	require.NoError(t, outA.Start())
	require.NoError(t, outB.Start())

	call := func(out *Outbound) <-chan error {
		done := make(chan error, 1)
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), callTimeout)
			defer cancel()
			res, err := out.Call(ctx, &transport.Request{
				Caller:    "caller",
				Service:   "service",
				Encoding:  raw.Encoding,
				Procedure: "hello",
				// No request body: the server runs in this process and
				// would take request chunks from the same pools.
				Body: strings.NewReader(""),
			})
			if err != nil {
				done <- err
				return
			}
			_, err = io.Copy(io.Discard, res.Body)
			_ = res.Body.Close()
			done <- err
		}()
		return done
	}
	// returned reports whether the call finished shortly after its deadline.
	returned := func(done <-chan error) (bool, error) {
		select {
		case err := <-done:
			return true, err
		case <-time.After(callTimeout + callGrace):
			return false, nil
		}
	}

	for _, out := range []*Outbound{outA, outB} {
		ok, err := returned(call(out))
		require.True(t, ok, "call on a healthy connection did not return")
		require.NoError(t, err, "call on a healthy connection failed")
	}

	// From here on, the first chunk any pool allocates has the wrong type.
	// Replacing the pools also drops the chunks cached by the calls above.
	var injected atomic.Bool
	for i := range http2DataChunkPools {
		newChunk := http2DataChunkPools[i].New
		http2DataChunkPools[i] = sync.Pool{New: func() interface{} {
			if injected.CompareAndSwap(false, true) {
				return "not a chunk"
			}
			return newChunk()
		}}
	}
	injectedAt := time.Now()

	ok, _ := returned(call(outA))
	require.True(t, injected.Load(), "the read loop did not take a chunk from the pools")
	assert.True(t, ok, "call whose response hit the panic is still blocked %v after its deadline", callGrace)

	ok, _ = returned(call(outB))
	assert.True(t, ok, "call on another outbound sharing the peer is still blocked %v after its deadline", callGrace)

	time.Sleep(time.Until(injectedAt.Add(recoveryWindow)))
	ok, err := returned(call(outB))
	if assert.True(t, ok, "call made %v after the panic is still blocked %v after its deadline: the connection was not replaced", recoveryWindow, callGrace) {
		assert.NoError(t, err, "call made %v after the panic failed: the connection was not replaced", recoveryWindow)
	}

	stopped := make(chan error, 1)
	go func() {
		stopped <- multierr.Combine(outA.Stop(), outB.Stop(), httpTransport.Stop())
	}()
	select {
	case err := <-stopped:
		assert.NoError(t, err, "outbounds and transport failed to stop")
	case <-time.After(5 * time.Second):
		t.Error("stopping the outbounds and the transport is still blocked after 5s")
	}
}
