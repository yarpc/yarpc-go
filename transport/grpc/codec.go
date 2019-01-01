// Copyright (c) 2019 Uber Technologies, Inc.
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

import "fmt"

// customCodec pass bytes to/from the wire without modification.
type customCodec struct{}

// Marshal takes a []byte and passes it through as a []byte.
func (customCodec) Marshal(obj interface{}) ([]byte, error) {
	switch value := obj.(type) {
	case []byte:
		return value, nil
	default:
		return nil, newCustomCodecMarshalCastError(obj)
	}
}

// Unmarshal takes a []byte pointer as obj and points it to data.
func (customCodec) Unmarshal(data []byte, obj interface{}) error {
	switch value := obj.(type) {
	case *[]byte:
		*value = data
		return nil
	default:
		return newCustomCodecUnmarshalCastError(obj)
	}
}

func (customCodec) String() string {
	// Setting this to what amounts to a nonsense value.
	// The encoding should always be inferred from the headers.
	// Setting this to a name that is not an encoding will assure
	// we get an error if this is used as the encoding value.
	return "yarpc"
}

func newCustomCodecMarshalCastError(actualObject interface{}) error {
	return fmt.Errorf("expected object to be of type []byte but got %T", actualObject)
}

func newCustomCodecUnmarshalCastError(actualObject interface{}) error {
	return fmt.Errorf("expected object to be of type *[]byte but got %T", actualObject)
}
