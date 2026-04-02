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

package compressor_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc/api/transport"
	yarpcgzip "go.uber.org/yarpc/compressor/gzip"
	yarpczstd "go.uber.org/yarpc/compressor/zstd"
)

type compressorFactory struct {
	name string
	new  func() transport.Compressor
}

var compressors = []compressorFactory{
	{"gzip", func() transport.Compressor { return yarpcgzip.New() }},
	{"zstd", func() transport.Compressor { return yarpczstd.New() }},
}

type payload struct {
	name string
	data []byte
}

// makePayloads builds payloads that approximate what the compressor sees in
// production: serialized protocol buffer and JSON bytes produced by YARPC
// encodings before they hit the wire.
//
// Sizes are chosen to match the tiers used by other benchmarks in the repo
// (encoding/protobuf/codec_bench_test.go, encoding/thrift/benchmark_test.go).
func makePayloads() []payload {
	return []payload{
		{"proto-350B", makeProtoLike(350)},
		{"proto-10KB", makeProtoLike(10 * 1024)},
		{"proto-1MB", makeProtoLike(1024 * 1024)},
		{"json-10KB", makeJSON(10 * 1024)},
		{"random-1KB", makeRandom(1024)},
	}
}

// Pools of realistic field values that provide moderate entropy — higher than
// strings.Repeat("x", N) but lower than fully random a-z.  Real protobuf
// messages contain UUIDs, service names, status strings, and addresses that
// repeat across fields and across messages in a batch.
var (
	sampleStrings = []string{
		"user-service", "payment-gateway", "order-processor",
		"550e8400-e29b-41d4-a716-446655440000",
		"7c9e6679-7425-40de-944b-e07fc1f90ae7",
		"https://internal.example.com/api/v2/users",
		"https://internal.example.com/api/v2/orders",
		"REQUEST_COMPLETED", "REQUEST_FAILED", "REQUEST_PENDING",
		"us-east-1", "us-west-2", "eu-west-1",
		"john.doe@example.com", "jane.smith@example.com",
		"192.168.1.100:8080", "10.0.0.42:9090",
		"error: context deadline exceeded",
		"error: connection refused",
		`{"retry": true, "attempt": 3, "backoff_ms": 500}`,
	}
)

// makeProtoLike builds a byte slice that approximates serialized protobuf wire
// format.  It mixes length-delimited string fields (wire type 2) drawn from
// realistic value pools with varint integer fields (wire type 0), producing the
// same byte patterns proto.Marshal emits for messages with mixed field types.
func makeProtoLike(targetSize int) []byte {
	rng := rand.New(rand.NewSource(42))
	var buf bytes.Buffer

	fieldNum := byte(1)
	for buf.Len() < targetSize {
		remaining := targetSize - buf.Len()
		if remaining <= 2 {
			break
		}

		if rng.Intn(4) == 0 {
			// ~25% varint integer fields (wire type 0).
			buf.WriteByte(fieldNum << 3) // wire type 0 = varint
			writeVarint(&buf, uint64(rng.Int63n(1<<32)))
		} else {
			// ~75% string fields (wire type 2) drawn from sample pool.
			s := sampleStrings[rng.Intn(len(sampleStrings))]

			// Occasionally duplicate the value to simulate repeated/list
			// fields in a batch response.
			reps := 1
			if rng.Intn(5) == 0 {
				reps = 2 + rng.Intn(4)
			}

			content := ""
			for r := 0; r < reps; r++ {
				content += s
			}
			if len(content) > remaining-3 {
				content = content[:remaining-3]
			}

			buf.WriteByte(fieldNum<<3 | 0x02)
			writeVarint(&buf, uint64(len(content)))
			buf.WriteString(content)
		}

		fieldNum++
		if fieldNum > 15 {
			fieldNum = 1
		}
	}

	return buf.Bytes()
}

func writeVarint(buf *bytes.Buffer, v uint64) {
	for v >= 0x80 {
		buf.WriteByte(byte(v&0x7f) | 0x80)
		v >>= 7
	}
	buf.WriteByte(byte(v))
}

// makeJSON builds a JSON object that approximates serialized JSON-encoded YARPC
// messages: nested objects with string keys and varied string/number values.
func makeJSON(targetSize int) []byte {
	rng := rand.New(rand.NewSource(99))
	obj := make(map[string]interface{})

	keys := []string{
		"request_id", "service", "method", "caller", "status",
		"error_message", "endpoint", "region", "user_email",
		"trace_id", "span_id", "duration_ms", "retry_count",
	}

	for i := 0; ; i++ {
		key := keys[i%len(keys)]
		if i >= len(keys) {
			key = fmt.Sprintf("%s_%d", key, i/len(keys))
		}

		if i%4 == 0 {
			obj[key] = rng.Int63n(1 << 32)
		} else {
			obj[key] = sampleStrings[rng.Intn(len(sampleStrings))]
		}

		data, _ := json.Marshal(obj)
		if len(data) >= targetSize {
			return data
		}
	}
}

func makeRandom(size int) []byte {
	rng := rand.New(rand.NewSource(7))
	buf := make([]byte, size)
	rng.Read(buf)
	return buf
}

func compress(b *testing.B, c transport.Compressor, data []byte) []byte {
	b.Helper()
	var buf bytes.Buffer
	w, err := c.Compress(&buf)
	require.NoError(b, err)
	_, err = w.Write(data)
	require.NoError(b, err)
	require.NoError(b, w.Close())
	return buf.Bytes()
}

func BenchmarkCompress(b *testing.B) {
	payloads := makePayloads()

	for _, cf := range compressors {
		for _, p := range payloads {
			c := cf.new()
			b.Run(fmt.Sprintf("%s/%s", cf.name, p.name), func(b *testing.B) {
				b.SetBytes(int64(len(p.data)))
				b.ReportAllocs()

				sample := compress(b, c, p.data)
				ratio := float64(len(sample)) / float64(len(p.data))

				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					var buf bytes.Buffer
					w, err := c.Compress(&buf)
					require.NoError(b, err)
					_, err = w.Write(p.data)
					require.NoError(b, err)
					require.NoError(b, w.Close())
				}
				b.StopTimer()
				b.ReportMetric(ratio, "ratio")
			})
		}
	}
}

func BenchmarkDecompress(b *testing.B) {
	payloads := makePayloads()

	for _, cf := range compressors {
		for _, p := range payloads {
			c := cf.new()
			compressed := compress(b, c, p.data)

			b.Run(fmt.Sprintf("%s/%s", cf.name, p.name), func(b *testing.B) {
				b.SetBytes(int64(len(p.data)))
				b.ReportAllocs()
				b.ResetTimer()

				for i := 0; i < b.N; i++ {
					r, err := c.Decompress(bytes.NewReader(compressed))
					require.NoError(b, err)
					_, err = io.Copy(io.Discard, r)
					require.NoError(b, err)
					r.Close()
				}
			})
		}
	}
}

