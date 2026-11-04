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

package http2

// Benchmarks of the connections the HTTP/2 server writes to, under load.
//
// BenchmarkConnWrite and BenchmarkConnWriteParallel isolate the connection
// from the network, by writing to one that discards everything.
// BenchmarkH2CServer measures it end to end, with an h2c server serving
// parallel requests over loopback.
//
// The connections and the listener come from newBenchConn and
// newBenchListener only, so that the benchmarks stay the same as those
// change: compare their results before and after with benchstat
// (golang.org/x/perf/cmd/benchstat), and vary the load with -cpu, e.g.:
//
//	go test -run '^$' -bench 'BenchmarkConnWrite|BenchmarkH2CServer' -benchmem \
//	    -cpu 1,4,8,16 -count 10 ./internal/net/http2 > new.txt
//	benchstat old.txt new.txt
//
// Add -cpuprofile and -mutexprofile to see where the time goes.

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
	xhttp2 "golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

// newBenchConn returns the connection the benchmarks write to, over conn,
// which reads the preface of an HTTP/1 or HTTP/2 client first.
// It is a [TrackedConn] that read it: tracked for an HTTP/2 client only.
func newBenchConn(conn *discardConn) net.Conn {
	tc := newTrackedConn(conn, func(net.Conn) func() { return func() {} })
	if _, err := tc.Read(make([]byte, len(conn.preface))); err != nil {
		panic(err)
	}
	return tc
}

// newBenchListener returns the listener the h2c server accepts connections
// from, over ln. It tracks them, with [WrapListener].
func newBenchListener(b *testing.B, ln net.Listener) net.Listener {
	helper := &recordingHelper{}
	b.Cleanup(func() {
		require.NotEmpty(b, helper.trackedConns(), "connections must be tracked")
	})
	return WrapListener(ln, helper)
}

// discardConn is a [net.Conn] that reads its preface, then nothing, and
// discards what is written to it.
type discardConn struct {
	stubConn
	preface []byte
}

// newDiscardConn returns a [discardConn] that reads the preface of a client
// of proto: "h1" or "h2".
func newDiscardConn(proto string) *discardConn {
	if proto == "h1" {
		return &discardConn{preface: []byte("POST / HTTP/1.1\r\n\r\n")}
	}
	return &discardConn{preface: []byte(xhttp2.ClientPreface)}
}

func (c *discardConn) Read(b []byte) (int, error) {
	if len(c.preface) == 0 {
		return 0, io.EOF
	}
	n := copy(b, c.preface)
	c.preface = c.preface[n:]
	return n, nil
}

func (c *discardConn) Write(b []byte) (int, error) { return len(b), nil }

// benchProtos are the protocols of the clients whose connections the
// benchmarks write to.
var benchProtos = []string{"h1", "h2"}

// benchWrite is a Write the HTTP/2 server makes: whole frames, so that the
// connection stays at frame boundaries as it is written again and again.
type benchWrite struct {
	name string
	b    []byte
}

// benchWrites returns Writes of the shapes the HTTP/2 server makes: a small
// response flushed at once, its write buffer flushed full of DATA frames,
// and a DATA frame of the default maximum frame size.
func benchWrites(tb testing.TB) []benchWrite {
	frames := func(write func(*xhttp2.Framer) error) []byte {
		var buf bytes.Buffer
		require.NoError(tb, write(xhttp2.NewFramer(&buf, nil)))
		return buf.Bytes()
	}
	headerBlock := bytes.Repeat([]byte{0x88}, 32)
	small := frames(func(f *xhttp2.Framer) error {
		if err := f.WriteHeaders(xhttp2.HeadersFrameParam{StreamID: 1, BlockFragment: headerBlock, EndHeaders: true}); err != nil {
			return err
		}
		if err := f.WriteData(1, false, make([]byte, 128)); err != nil {
			return err
		}
		return f.WriteHeaders(xhttp2.HeadersFrameParam{StreamID: 1, BlockFragment: headerBlock, EndHeaders: true, EndStream: true})
	})
	buffer := frames(func(f *xhttp2.Framer) error {
		for i := 0; i < 4; i++ {
			if err := f.WriteData(1, false, make([]byte, 1024-9)); err != nil { // 9: frame header
				return err
			}
		}
		return nil
	})
	maxFrame := frames(func(f *xhttp2.Framer) error {
		return f.WriteData(1, false, make([]byte, 16<<10))
	})
	return []benchWrite{
		{"small-response", small},
		{"4KiB-buffer", buffer},
		{"16KiB-frame", maxFrame},
	}
}

func TestBenchConn(t *testing.T) {
	for _, proto := range benchProtos {
		conn := newBenchConn(newDiscardConn(proto))
		require.Equal(t, proto == "h2", conn.(*TrackedConn).tracked.Load(), proto)
		for _, w := range benchWrites(t) {
			n, err := conn.Write(w.b)
			require.NoError(t, err, proto)
			require.Equal(t, len(w.b), n, proto)
		}
	}
}

// BenchmarkConnWrite measures the cost of a Write, one at a time.
func BenchmarkConnWrite(b *testing.B) {
	for _, proto := range benchProtos {
		for _, w := range benchWrites(b) {
			b.Run("proto="+proto+"/write="+w.name, func(b *testing.B) {
				conn := newBenchConn(newDiscardConn(proto))
				b.SetBytes(int64(len(w.b)))
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					if _, err := conn.Write(w.b); err != nil {
						b.Fatal(err)
					}
				}
			})
		}
	}
}

// BenchmarkConnWriteParallel measures the cost of a Write under load:
// GOMAXPROCS goroutines write at once, each to a connection of its own, as
// the HTTP/2 server has a single writer per connection.
func BenchmarkConnWriteParallel(b *testing.B) {
	for _, proto := range benchProtos {
		for _, w := range benchWrites(b) {
			b.Run("proto="+proto+"/write="+w.name, func(b *testing.B) {
				b.SetBytes(int64(len(w.b)))
				b.ReportAllocs()
				b.RunParallel(func(pb *testing.PB) {
					conn := newBenchConn(newDiscardConn(proto))
					for pb.Next() {
						if _, err := conn.Write(w.b); err != nil {
							b.Error(err)
							return
						}
					}
				})
			})
		}
	}
}

// BenchmarkH2CServer measures the cost of the connections end to end, under
// load: an h2c server echoes the body of requests sent over conns
// connections, GOMAXPROCS at once on each.
func BenchmarkH2CServer(b *testing.B) {
	for _, conns := range []int{1, 16} {
		for _, size := range []int{128, 16 << 10} {
			b.Run(fmt.Sprintf("conns=%d/size=%d", conns, size), func(b *testing.B) {
				benchmarkH2CServer(b, conns, size)
			})
		}
	}
}

func benchmarkH2CServer(b *testing.B, conns, size int) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(b, err)
	ln = newBenchListener(b, ln)
	echo := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(w, r.Body)
	})
	srv := &http.Server{Handler: h2c.NewHandler(echo, &xhttp2.Server{})}
	go func() { _ = srv.Serve(ln) }()
	defer func() { _ = srv.Close() }()

	clients := make([]*http.Client, conns)
	for i := range clients {
		tr := &xhttp2.Transport{
			AllowHTTP: true,
			DialTLSContext: func(ctx context.Context, network, addr string, _ *tls.Config) (net.Conn, error) {
				var d net.Dialer
				return d.DialContext(ctx, network, addr)
			},
		}
		defer tr.CloseIdleConnections()
		clients[i] = &http.Client{Transport: tr}
	}
	url := "http://" + ln.Addr().String()
	body := make([]byte, size)

	var next atomic.Int64
	b.SetBytes(int64(2 * size))
	b.SetParallelism(conns)
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		client := clients[int(next.Add(1))%conns]
		for pb.Next() {
			resp, err := client.Post(url, "application/octet-stream", bytes.NewReader(body))
			if err != nil {
				b.Error(err)
				return
			}
			n, err := io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
			if err != nil || n != int64(size) {
				b.Errorf("read %d bytes of %d: %v", n, size, err)
				return
			}
		}
	})
}
