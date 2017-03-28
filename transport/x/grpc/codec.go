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
	"fmt"

	"github.com/gogo/protobuf/proto"
)

// customCodec will either handle proto.Message objects, or
// pass bytes to/from the wire without modification.
type customCodec struct{}

// Marshal takes a proto.Message and marshals it, or
// takes a []byte pointer and passes it through as a []byte.
func (customCodec) Marshal(obj interface{}) ([]byte, error) {
	switch value := obj.(type) {
	case proto.Message:
		return proto.Marshal(value)
	case *[]byte:
		return *value, nil
	default:
		return nil, newCustomCodecCastError(obj)
	}
}

// Unmarshal takes a proto.Message and unmarshals it, or
// takes a []byte pointer and writes it to v.
func (customCodec) Unmarshal(data []byte, obj interface{}) error {
	switch value := obj.(type) {
	case proto.Message:
		return proto.Unmarshal(data, value)
	case *[]byte:
		*value = data
		return nil
	default:
		return newCustomCodecCastError(obj)
	}
}

func (customCodec) String() string {
	// TODO: we might have to fake this as proto
	// this is hacky
	return "custom"
}

func newCustomCodecCastError(actualObject interface{}) error {
	return fmt.Errorf("expected object of either type proto.Message or *[]byte but got %T", actualObject)
}
