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

package yarpcgzip_test

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"io"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
	yarpcgzip "go.uber.org/yarpc/compressor/gzip"
)

// The tests in this file pin down the reuse-safety contract of the
// io.ReadCloser returned by (*Compressor).Decompress:
//
//  1. Close is idempotent for pooling — calling it more than once must not
//     place the same pooled instance into the internal sync.Pool twice.
//  2. As a consequence of (1), no two concurrent Decompress callers can ever
//     receive the same underlying *gzip.Reader. The wrapped flate state is
//     not safe for concurrent use.
//
// These invariants matter because the production gRPC compressor adapter
// (compressor/grpc/grpc.go) calls Close from Read on io.EOF, and the gRPC
// runtime may issue additional Reads / Closes on the same reader. If the
// invariants are violated, two RPC goroutines decompress through a shared
// flate state, corrupting heap memory that surfaces later as crashes in
// unrelated goroutines (e.g. gRPC's loopyWriter).

// TestDecompressorCloseIsIdempotentForPooling is a fast, deterministic
// regression check: after a caller has driven Decompress through the gRPC
// drain-after-EOF pattern (which calls Close twice on the same reader),
// subsequent independent Decompress calls must produce independent readers.
//
// On a correct implementation, two follow-up Decompress calls each operate
// on their own *gzip.Reader, so interleaved Reads produce the two distinct
// plaintexts. On a regressed implementation, the double-Close double-Puts
// the same pooled *reader instance into the sync.Pool; the next two Get
// calls return the same instance, the second Reset clobbers the first, and
// the two readers race over a shared *gzip.Reader / flate state — typically
// surfacing here as the first reader yielding the second reader's plaintext
// (or a flate error).
func TestDecompressorCloseIsIdempotentForPooling(t *testing.T) {
	c := yarpcgzip.New()

	plain1 := []byte("first-payload:" + strings.Repeat("a", 1024))
	plain2 := []byte("second-payload:" + strings.Repeat("b", 1024))
	plain3 := []byte("third-payload:" + strings.Repeat("c", 1024))

	// Step 1: drive a Decompress through the production gRPC drain pattern,
	// which calls Close twice on the underlying reader.
	rc, err := c.Decompress(bytes.NewReader(compressData(t, c, plain1)))
	require.NoError(t, err)
	gw := &grpcAdapter{reader: rc}
	_, err = io.ReadAll(gw) // reaches EOF -> wrapper calls Close once
	require.NoError(t, err)
	// gRPC's post-EOF drain: one more Read on the wrapper. On a regressed
	// implementation this triggers a second Close -> a second Put.
	var tail [16]byte
	_, _ = gw.Read(tail[:])

	// Step 2: two follow-up Decompress calls. On a regressed implementation
	// both Get the same pooled *reader, the second Reset overwrites the
	// first, and the readers share state.
	rc2, err := c.Decompress(bytes.NewReader(compressData(t, c, plain2)))
	require.NoError(t, err)
	defer rc2.Close()
	rc3, err := c.Decompress(bytes.NewReader(compressData(t, c, plain3)))
	require.NoError(t, err)
	defer rc3.Close()

	got2, err := io.ReadAll(rc2)
	require.NoError(t, err, "rc2 Read failed (likely sharing flate state with rc3)")
	got3, err := io.ReadAll(rc3)
	require.NoError(t, err, "rc3 Read failed (likely sharing flate state with rc2)")

	require.Equal(t, plain2, got2,
		"rc2 decoded the wrong plaintext; Close double-Put let rc2 and rc3 share a pooled *reader")
	require.Equal(t, plain3, got3,
		"rc3 decoded the wrong plaintext; Close double-Put let rc2 and rc3 share a pooled *reader")
}

// TestDecompressorErrorPaths covers the two error branches of Decompress:
//
//   - gzip.NewReader fails on the cold path (empty pool, invalid gzip input).
//   - *gzip.Reader.Reset fails on the warm path (pool primed, invalid gzip
//     input). The pooled *reader must be returned to the pool so it remains
//     available for subsequent valid Decompress calls.
func TestDecompressorErrorPaths(t *testing.T) {
	c := yarpcgzip.New()
	invalid := []byte("not gzip data")
	valid := compressData(t, c, input)

	t.Run("NewReader fails with empty pool", func(t *testing.T) {
		c := yarpcgzip.New()
		rc, err := c.Decompress(bytes.NewReader(invalid))
		require.Error(t, err)
		require.Nil(t, rc)
	})

	t.Run("Reset fails with primed pool and pooled reader is recovered", func(t *testing.T) {
		// Prime the pool with one valid *reader.
		rc, err := c.Decompress(bytes.NewReader(valid))
		require.NoError(t, err)
		_, err = io.ReadAll(rc)
		require.NoError(t, err)
		require.NoError(t, rc.Close())

		// Force the warm-path Reset error branch.
		rc, err = c.Decompress(bytes.NewReader(invalid))
		require.Error(t, err)
		require.Nil(t, rc)

		// The pooled *reader must have been returned to the pool: the next
		// valid Decompress must still succeed.
		rc, err = c.Decompress(bytes.NewReader(valid))
		require.NoError(t, err)
		got, err := io.ReadAll(rc)
		require.NoError(t, err)
		require.NoError(t, rc.Close())
		require.Equal(t, input, got)
	})
}

// TestDecompressorIsSafeUnderConcurrentGRPCDrain is the `-race` regression
// check for the same contract under concurrent load. It drives Decompress
// through the exact wrapper pattern used by compressor/grpc/grpc.go and
// includes the post-EOF drain Read that triggers a second Close.
//
// On a correct implementation, every iteration verifies its decompressed
// output and the race detector stays silent. On a regressed implementation,
// the race detector trips inside compress/flate.(*decompressor) and/or the
// goroutine panics with a nil deref / corrupted flate state.
//
// Run with the race detector enabled:
//
//	go test -race -run TestDecompressorIsSafeUnderConcurrentGRPCDrain ./compressor/gzip/ -count=1 -timeout 60s
//
// For stronger confidence (re-runs the test 10 times to surface flaky races):
//
//	go test -race -run TestDecompressorIsSafeUnderConcurrentGRPCDrain ./compressor/gzip/ -count=10 -timeout 120s
func TestDecompressorIsSafeUnderConcurrentGRPCDrain(t *testing.T) {
	compressor := yarpcgzip.New()

	// A small corpus of sizable, high-entropy payloads. The working set is
	// intentionally small so goroutines actually contend on a handful of
	// pooled *reader instances rather than each Get allocating fresh.
	const numPayloads = 4
	payloads := make([][]byte, numPayloads)
	plaintexts := make([][]byte, numPayloads)
	for i := 0; i < numPayloads; i++ {
		raw := make([]byte, 8*1024)
		_, err := rand.Read(raw)
		require.NoError(t, err)
		raw = append([]byte(fmt.Sprintf("payload-%d:", i)), raw...)
		plaintexts[i] = raw
		payloads[i] = compressData(t, compressor, raw)
	}

	// Warm the pool so subsequent goroutines reuse rather than allocate.
	for i := 0; i < 8; i++ {
		dr, err := compressor.Decompress(bytes.NewReader(payloads[i%numPayloads]))
		require.NoError(t, err)
		_, err = io.ReadAll(dr)
		require.NoError(t, err)
		require.NoError(t, dr.Close())
	}

	workers := runtime.GOMAXPROCS(0) * 4
	if workers < 32 {
		workers = 32
	}
	const iterationsPerWorker = 200

	var (
		wg         sync.WaitGroup
		mismatches int64
	)
	wg.Add(workers)
	for w := 0; w < workers; w++ {
		w := w
		go func() {
			defer wg.Done()
			buf := make([]byte, 0, 16*1024)
			for i := 0; i < iterationsPerWorker; i++ {
				idx := (w + i) % numPayloads
				want := plaintexts[idx]

				dr, err := compressor.Decompress(bytes.NewReader(payloads[idx]))
				if err != nil {
					t.Errorf("Decompress failed: %v", err)
					return
				}

				// Drive Decompress through the exact wrapper that
				// compressor/grpc/grpc.go uses: Read calls Close on io.EOF.
				gw := &grpcAdapter{reader: dr}

				buf = buf[:0]
				got, err := readAllInto(gw, buf)
				if err != nil {
					t.Errorf("Read failed: %v", err)
					return
				}

				// gRPC frequently issues one more Read after EOF to drain.
				// On a regressed implementation this triggers a second Close
				// and a double-Put into the pool.
				var tail [16]byte
				_, _ = gw.Read(tail[:])

				if !bytes.Equal(got, want) {
					atomic.AddInt64(&mismatches, 1)
				}
			}
		}()
	}
	wg.Wait()

	require.Zero(t, atomic.LoadInt64(&mismatches),
		"decompressed output differed from expected input; pooled *reader was shared across goroutines")
}

// grpcAdapter mirrors the production wrapper in compressor/grpc/grpc.go: it
// calls Close on the wrapped ReadCloser whenever Read returns io.EOF.
type grpcAdapter struct {
	reader io.ReadCloser
}

func (r *grpcAdapter) Read(buf []byte) (int, error) {
	n, err := r.reader.Read(buf)
	if err == io.EOF {
		_ = r.reader.Close()
	}
	return n, err
}

// readAllInto drains r into dst, growing it as needed, without allocating a
// fresh backing array per iteration.
func readAllInto(r io.Reader, dst []byte) ([]byte, error) {
	for {
		if len(dst) == cap(dst) {
			dst = append(dst, 0)[:len(dst)]
		}
		n, err := r.Read(dst[len(dst):cap(dst)])
		dst = dst[:len(dst)+n]
		if err == io.EOF {
			return dst, nil
		}
		if err != nil {
			return dst, err
		}
	}
}
