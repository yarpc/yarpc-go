// Copyright (c) 2017 Uber Technologies, Inc.
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

	"github.com/stretchr/testify/assert"
)

func TestPassThroughCodecMarshal(t *testing.T) {
	value := []byte("test")
	data, err := noopCodec{}.Marshal(&value)
	assert.Equal(t, value, data)
	assert.NoError(t, err)
}

func TestPassThroughCodecMarshalError(t *testing.T) {
	value := "test"
	data, err := noopCodec{}.Marshal(&value)
	assert.Equal(t, []byte(nil), data)
	assert.Equal(t, newBytesPointerCastError(&value), err)
}

func TestPassThroughCodecUnmarshal(t *testing.T) {
	data := []byte("test")
	var value []byte
	assert.NoError(t, noopCodec{}.Unmarshal(data, &value))
	assert.Equal(t, data, value)
}

func TestPassThroughCodecUnmarshalError(t *testing.T) {
	var value string
	err := noopCodec{}.Unmarshal([]byte("test"), &value)
	assert.Equal(t, "", value)
	assert.Equal(t, newBytesPointerCastError(&value), err)
}

func TestPassThroughCodecString(t *testing.T) {
	assert.Equal(t, "noop", noopCodec{}.String())
}
