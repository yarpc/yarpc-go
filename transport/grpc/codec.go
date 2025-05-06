// Copyright (c) 2025 Uber Technologies, Inc.
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
	"fmt"

	"google.golang.org/grpc/encoding"
	"google.golang.org/grpc/mem"
)

// Name is the name registered for the CustomCodec.
const Name = "yarpc"

// CustomCodec pass bytes to/from the wire without modification.
type CustomCodec struct{}

func init() {
	encoding.RegisterCodecV2(CustomCodec{})
}

// Marshal takes a []byte and passes it through as a mem.BufferSlice
func (CustomCodec) Marshal(v any) (mem.BufferSlice, error) {
	var bytes []byte
	var err error

	switch value := v.(type) {
	case []byte:
		bytes, err = value, nil
	default:
		return nil, newCustomCodecMarshalCastError(v)
	}

	return mem.BufferSlice{mem.SliceBuffer(bytes)}, err
}

// Unmarshal takes a mem.BufferSlice and copies the bytes to a []byte
func (CustomCodec) Unmarshal(data mem.BufferSlice, v any) error {
	switch value := v.(type) {
	case *[]byte:
		*value = data.Materialize()
		return nil
	default:
		return newCustomCodecUnmarshalCastError(v)
	}
}

func (CustomCodec) Name() string {
	return Name
}

func newCustomCodecMarshalCastError(actualObject interface{}) error {
	return fmt.Errorf("expected object to be of type []byte but got %T", actualObject)
}

func newCustomCodecUnmarshalCastError(actualObject interface{}) error {
	return fmt.Errorf("expected object to be of type *[]byte but got %T", actualObject)
}
