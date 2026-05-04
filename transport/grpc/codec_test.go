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
