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

package grpc

import (
	"testing"

	"google.golang.org/grpc/mem"

	"github.com/stretchr/testify/assert"
)

func TestCustomCodecMarshalBytes(t *testing.T) {
	value := []byte("test")
	data, err := customCodec{}.Marshal(value)
	assert.Equal(t, value, data.Materialize())
	assert.NoError(t, err)
}

func TestCustomCodecMarshalCastError(t *testing.T) {
	value := "test"
	data, err := customCodec{}.Marshal(&value)
	assert.Equal(t, mem.BufferSlice(mem.BufferSlice(nil)), data)
	assert.Equal(t, newCustomCodecMarshalCastError(&value), err)
}

func TestCustomCodecUnmarshalBytes(t *testing.T) {
	data := mem.BufferSlice{mem.SliceBuffer("test")}
	var value []byte
	assert.NoError(t, customCodec{}.Unmarshal(data, &value))
	assert.Equal(t, data.Materialize(), value)
}

func TestCustomCodecUnmarshalCastError(t *testing.T) {
	var value string
	err := customCodec{}.Unmarshal(mem.BufferSlice{mem.SliceBuffer("test")}, &value)
	assert.Equal(t, "", value)
	assert.Equal(t, newCustomCodecUnmarshalCastError(&value), err)
}

func TestCustomCodecName(t *testing.T) {
	assert.Equal(t, "proto", customCodec{}.Name())
}

func TestCustomCodecMarshalMemBuffer(t *testing.T) {
	t.Run("returns BufferSlice with single buffer", func(t *testing.T) {
		data := []byte("test data")
		pool := &testBufferPool{putFunc: func(buf *[]byte) {}}

		buffer := mem.NewBuffer(&data, pool)
		t.Cleanup(buffer.Free)

		result, err := customCodec{}.Marshal(buffer)
		t.Cleanup(result.Free)

		assert.NoError(t, err)
		assert.Equal(t, 1, len(result), "Should return BufferSlice with 1 buffer")
		assert.IsType(t, mem.BufferSlice{}, result, "Should return mem.BufferSlice type")
	})

	t.Run("preserves buffer content", func(t *testing.T) {
		data := []byte("test content")
		pool := &testBufferPool{putFunc: func(buf *[]byte) {}}
		buffer := mem.NewBuffer(&data, pool)
		t.Cleanup(buffer.Free)

		result, err := customCodec{}.Marshal(buffer)
		t.Cleanup(result.Free)

		assert.NoError(t, err)

		materialized := result.Materialize()
		assert.Equal(t, data, materialized, "Buffer content should be preserved")
	})
}

func TestCustomCodecMarshalMemBufferSlice(t *testing.T) {
	t.Run("returns BufferSlice with all buffers", func(t *testing.T) {
		data1 := []byte("test1")
		data2 := []byte("test2")
		pool := &testBufferPool{putFunc: func(buf *[]byte) {}}

		buf1 := mem.NewBuffer(&data1, pool)
		t.Cleanup(buf1.Free)

		buf2 := mem.NewBuffer(&data2, pool)
		t.Cleanup(buf2.Free)

		bufferSlice := mem.BufferSlice{buf1, buf2}

		result, err := customCodec{}.Marshal(bufferSlice)
		t.Cleanup(result.Free)

		assert.NoError(t, err)
		assert.Equal(t, 2, len(result), "Should return BufferSlice with 2 buffers")
	})

	t.Run("preserves buffer order and content", func(t *testing.T) {
		data1 := []byte("first")
		data2 := []byte("second")
		pool := &testBufferPool{putFunc: func(buf *[]byte) {}}

		buf1 := mem.NewBuffer(&data1, pool)
		t.Cleanup(buf1.Free)

		buf2 := mem.NewBuffer(&data2, pool)
		t.Cleanup(buf2.Free)

		bufferSlice := mem.BufferSlice{buf1, buf2}

		result, err := customCodec{}.Marshal(bufferSlice)
		t.Cleanup(result.Free)

		assert.NoError(t, err)

		materialized := result.Materialize()
		expected := append(data1, data2...)
		assert.Equal(t, expected, materialized, "Buffer order and content should be preserved")
	})
}

func TestCustomCodecMarshalEmptyBuffer(t *testing.T) {
	t.Run("handles zero-length buffer", func(t *testing.T) {
		emptyData := []byte{}
		pool := &testBufferPool{putFunc: func(buf *[]byte) {}}

		buffer := mem.NewBuffer(&emptyData, pool)
		t.Cleanup(buffer.Free)

		result, err := customCodec{}.Marshal(buffer)
		t.Cleanup(result.Free)

		assert.NoError(t, err)
		assert.Equal(t, 1, len(result), "Should return BufferSlice with 1 buffer")
		assert.Equal(t, 0, result.Len(), "Empty buffer should have zero length")
	})
}

type testBufferPool struct {
	putFunc func(buf *[]byte)
}

func (m *testBufferPool) Get(length int) *[]byte { return nil }
func (m *testBufferPool) Put(buf *[]byte) {
	if m.putFunc != nil {
		m.putFunc(buf)
	}
}
