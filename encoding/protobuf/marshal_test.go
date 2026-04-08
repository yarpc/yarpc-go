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

package protobuf

import (
	"bytes"
	"runtime"
	"strings"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/gogo/protobuf/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/yarpcerrors"
)

func TestUnhandledEncoding(t *testing.T) {
	assert.Equal(t, yarpcerrors.CodeInternal,
		yarpcerrors.FromError(unmarshal(transport.Encoding("foo"), strings.NewReader("foo"), nil, newCodec(nil))).Code())
	_, cleanup, err := marshal(transport.Encoding("foo"), nil, newCodec(nil))
	assert.Equal(t, yarpcerrors.CodeInternal, yarpcerrors.FromError(err).Code())
	// Ensure cleanup is never nil to avoid nil pointer dereference
	assert.NotNil(t, cleanup, "cleanup function should never be nil")
	assert.NotPanics(t, func() { cleanup() }, "cleanup should be safe to call even on error")
}

// testBytesReader implements io.Reader and exposes Bytes() for the fast path.
// It tracks whether Bytes() and Read() were called to verify which path was taken.
type testBytesReader struct {
	data       []byte
	reader     *bytes.Reader
	bytesCalled bool
	readCalled  bool
}

func newTestBytesReader(data []byte) *testBytesReader {
	return &testBytesReader{data: data, reader: bytes.NewReader(data)}
}

func (r *testBytesReader) Read(p []byte) (int, error) {
	r.readCalled = true
	return r.reader.Read(p)
}

func (r *testBytesReader) Bytes() []byte {
	r.bytesCalled = true
	return r.data
}

func TestUnmarshalFastPath(t *testing.T) {
	c := newCodec(nil)

	t.Run("Bytes called and Read not called", func(t *testing.T) {
		original := &types.StringValue{Value: "hello"}
		data, err := proto.Marshal(original)
		require.NoError(t, err)

		reader := newTestBytesReader(data)
		got := &types.StringValue{}

		err = unmarshal(Encoding, reader, got, c)
		assert.NoError(t, err)
		assert.Equal(t, original.Value, got.Value, "Message should be deserialized correctly via fast path")
		assert.True(t, reader.bytesCalled, "Bytes() should be called on fast path")
		assert.False(t, reader.readCalled, "Read() should not be called on fast path")
	})

	t.Run("empty body returns nil", func(t *testing.T) {
		reader := newTestBytesReader([]byte{})

		err := unmarshal(Encoding, reader, nil, c)
		assert.NoError(t, err, "Empty body on fast path should return nil")
		assert.True(t, reader.bytesCalled, "Bytes() should still be called for empty body")
		assert.False(t, reader.readCalled, "Read() should not be called for empty body")
	})

	t.Run("invalid encoding returns error", func(t *testing.T) {
		reader := newTestBytesReader([]byte("data"))

		err := unmarshal(transport.Encoding("unknown"), reader, nil, c)
		assert.Equal(t, yarpcerrors.CodeInternal, yarpcerrors.FromError(err).Code(),
			"Fast path should still return encoding error for unrecognized encoding")
	})

	t.Run("malformed protobuf returns unmarshal error", func(t *testing.T) {
		reader := newTestBytesReader([]byte{0xff, 0xff, 0xff})

		err := unmarshal(Encoding, reader, nil, c)
		assert.Error(t, err, "Malformed protobuf should return an error on fast path")
	})
}

func BenchmarkUnmarshalBytesReader(b *testing.B) {
	sizes := []struct {
		name string
		size int
	}{
		{"100B", 100},
		{"1KB", 1024},
		{"10KB", 10 * 1024},
		{"100KB", 100 * 1024},
	}
	c := newCodec(nil)
	for _, sz := range sizes {
		original := &types.StringValue{Value: strings.Repeat("x", sz.size)}
		data, err := proto.Marshal(original)
		if err != nil {
			b.Fatal(err)
		}
		b.Run(sz.name, func(b *testing.B) {
			b.ReportAllocs()
			runtime.GC()
			var before runtime.MemStats
			runtime.ReadMemStats(&before)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				msg := &types.StringValue{}
				if err := unmarshal(Encoding, newTestBytesReader(data), msg, c); err != nil {
					b.Fatal(err)
				}
			}
			b.StopTimer()

			var after runtime.MemStats
			runtime.ReadMemStats(&after)
			b.ReportMetric(float64(after.TotalAlloc-before.TotalAlloc)/float64(b.N), "heap-B/op")
			b.ReportMetric(float64(after.NumGC-before.NumGC)/float64(b.N), "gc-cycles/op")
			b.ReportMetric(float64(after.PauseTotalNs-before.PauseTotalNs)/float64(b.N), "gc-pause-ns/op")
		})
	}
}

func TestUnmarshalSlowPath(t *testing.T) {
	c := newCodec(nil)

	t.Run("deserializes correctly without Bytes method", func(t *testing.T) {
		original := &types.StringValue{Value: "hello"}
		data, err := proto.Marshal(original)
		require.NoError(t, err)

		reader := bytes.NewReader(data) // no Bytes() method
		got := &types.StringValue{}

		err = unmarshal(Encoding, reader, got, c)
		assert.NoError(t, err)
		assert.Equal(t, original.Value, got.Value, "Message should be deserialized correctly via slow path")
	})

	t.Run("empty body returns nil", func(t *testing.T) {
		reader := bytes.NewReader([]byte{})

		err := unmarshal(Encoding, reader, nil, c)
		assert.NoError(t, err, "Empty body on slow path should return nil")
	})
}
