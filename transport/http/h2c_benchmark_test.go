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
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

// BenchmarkH2CConnectionReads compares the old h2c handler and persistent
// connection buffering on one warm connection. Reads/writes count net.Conn
// operations, not syscalls. ns/op and allocations include both loopback peers.
func BenchmarkH2CConnectionReads(b *testing.B) {
	for _, size := range []int{1024, 32768} {
		for _, parallel := range []bool{false, true} {
			for _, buffered := range []bool{false, true} {
				b.Run(fmt.Sprintf("bytes=%d/parallel=%t/buffered=%t", size, parallel, buffered), func(b *testing.B) {
					benchmarkH2CConnectionReads(b, size, parallel, buffered)
				})
			}
		}
	}
}

func benchmarkH2CConnectionReads(b *testing.B, size int, parallel, buffered bool) {
	payload := bytes.Repeat([]byte("x"), size)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := io.Copy(io.Discard, r.Body); err != nil {
			b.Error(err)
			return
		}
		if _, err := w.Write(payload); err != nil {
			b.Error(err)
		}
	})
	var serverHandler http.Handler = h2c.NewHandler(handler, &http2.Server{})
	if buffered {
		serverHandler = newBufferedH2CHandler(handler, &http2.Server{})
	}
	server := httptest.NewUnstartedServer(serverHandler)
	listener := &h2cCountingListener{Listener: server.Listener}
	server.Listener = listener
	server.Start()
	b.Cleanup(func() {
		server.Close()
		listener.closeConnections()
	})
	client := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http2.Transport{
			AllowHTTP: true,
			DialTLSContext: func(ctx context.Context, network, addr string, _ *tls.Config) (net.Conn, error) {
				var dialer net.Dialer
				return dialer.DialContext(ctx, network, addr)
			},
		},
	}
	b.Cleanup(client.CloseIdleConnections)
	call := func() error {
		response, err := client.Post(server.URL, "application/octet-stream", bytes.NewReader(payload))
		if err != nil {
			return err
		}
		n, readErr := io.Copy(io.Discard, response.Body)
		closeErr := response.Body.Close()
		if readErr != nil {
			return readErr
		}
		if closeErr != nil {
			return closeErr
		}
		if n != int64(size) || response.StatusCode != http.StatusOK || response.ProtoMajor != 2 {
			return fmt.Errorf("invalid response: bytes=%d status=%d proto=%s", n, response.StatusCode, response.Proto)
		}
		return nil
	}
	require.NoError(b, call()) // Establish the connection before timing.
	reads, writes := listener.reads.Load(), listener.writes.Load()
	b.ReportAllocs()
	b.SetBytes(int64(size))
	b.ResetTimer()
	if parallel {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				if err := call(); err != nil {
					b.Error(err)
					return
				}
			}
		})
	} else {
		for i := 0; i < b.N; i++ {
			require.NoError(b, call())
		}
	}
	b.StopTimer()
	b.ReportMetric(float64(listener.reads.Load()-reads)/float64(b.N), "server-reads/op")
	b.ReportMetric(float64(listener.writes.Load()-writes)/float64(b.N), "server-writes/op")
	b.ReportMetric(float64(listener.connections.Load()), "conns")
}

type h2cCountingListener struct {
	net.Listener
	reads       atomic.Int64
	writes      atomic.Int64
	connections atomic.Int64
	mu          sync.Mutex
	conns       []net.Conn
}

func (l *h2cCountingListener) Accept() (net.Conn, error) {
	conn, err := l.Listener.Accept()
	if err != nil {
		return nil, err
	}
	l.connections.Add(1)
	l.mu.Lock()
	l.conns = append(l.conns, conn)
	l.mu.Unlock()
	return &h2cCountingConn{Conn: conn, listener: l}, nil
}

func (l *h2cCountingListener) closeConnections() {
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, conn := range l.conns {
		_ = conn.Close()
	}
}

type h2cCountingConn struct {
	net.Conn
	listener *h2cCountingListener
}

func (c *h2cCountingConn) Read(p []byte) (int, error) {
	c.listener.reads.Add(1)
	return c.Conn.Read(p)
}

func (c *h2cCountingConn) Write(p []byte) (int, error) {
	c.listener.writes.Add(1)
	return c.Conn.Write(p)
}
