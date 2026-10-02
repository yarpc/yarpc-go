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
	"io"
	"net"
	"net/http"
	"runtime/metrics"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// BenchmarkH2ConnectionContention measures how the number of independent
// HTTP/2 connections affects lock contention. The same parallel load is sent
// through 1, 2, 4 and 8 connections to one server, each connection owned by
// its own peer. One connection is what every peer shared before peers had
// their own pools, so conns=1 is the baseline.
//
// Besides ns/op it reports:
//   - mutex-wait-ns/op: time goroutines spent blocked on sync mutexes, per
//     request, from the runtime's cumulative /sync/mutex/wait/total:seconds.
//     It is process-wide, so it includes the server side of the loopback
//     test; compare the conns=N rows against each other rather than reading
//     the absolute value.
//   - conns: connections the server accepted, confirming the peers really
//     used independent connections.
//
// Run with, for example:
//
//	go test ./transport/http -run '^$' -bench BenchmarkH2ConnectionContention -benchtime=3s
func BenchmarkH2ConnectionContention(b *testing.B) {
	for _, conns := range []int{1, 2, 4, 8} {
		b.Run("conns="+strconv.Itoa(conns), func(b *testing.B) {
			benchmarkH2Connections(b, conns)
		})
	}
}

func benchmarkH2Connections(b *testing.B, conns int) {
	var accepted atomic.Int32
	server := newH2TestServer(func(_ net.Conn, state http.ConnState) {
		if state == http.StateNew {
			accepted.Add(1)
		}
	})
	defer server.Close()

	tr := NewTransport()
	require.NoError(b, tr.Start())
	defer func() { assert.NoError(b, tr.Stop()) }()

	addr := strings.TrimPrefix(server.URL, "http://")
	senders := make([]*h2PeerSender, conns)
	for i := range senders {
		p := mustNewPeer(b, addr, tr)
		defer func() {
			if pool := p.loadH2Pool(); pool != nil {
				pool.Stop()
				pool.Wait()
			}
		}()
		senders[i] = &h2PeerSender{peer: p}

		// Dial up front so the timed section measures steady-state
		// traffic, not connection setup.
		_, err := p.h2Sender()
		require.NoError(b, err)
	}

	do := func(s *h2PeerSender) {
		req, err := http.NewRequest(http.MethodGet, server.URL, nil)
		if err != nil {
			b.Error(err)
			return
		}
		resp, err := s.Do(req)
		if err != nil {
			b.Error(err)
			return
		}
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}

	var next atomic.Uint32
	b.SetParallelism(16)
	b.ReportAllocs()
	b.ResetTimer()
	waitBefore := mutexWaitSeconds()
	b.RunParallel(func(pb *testing.PB) {
		s := senders[int(next.Add(1)-1)%len(senders)]
		for pb.Next() {
			do(s)
		}
	})
	waitAfter := mutexWaitSeconds()
	b.StopTimer()

	b.ReportMetric((waitAfter-waitBefore)*1e9/float64(b.N), "mutex-wait-ns/op")
	b.ReportMetric(float64(accepted.Load()), "conns")
}

func mutexWaitSeconds() float64 {
	sample := []metrics.Sample{{Name: "/sync/mutex/wait/total:seconds"}}
	metrics.Read(sample)
	if sample[0].Value.Kind() != metrics.KindFloat64 {
		return 0
	}
	return sample[0].Value.Float64()
}
