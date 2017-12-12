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

package thrift

import (
	"io"

	"go.uber.org/thriftrw/wire"
)

type fakeEnveloper wire.EnvelopeType

func (fakeEnveloper) MethodName() string {
	return "someMethod"
}

func (e fakeEnveloper) EnvelopeType() wire.EnvelopeType {
	return wire.EnvelopeType(e)
}

func (fakeEnveloper) ToWire() (wire.Value, error) {
	return wire.NewValueStruct(wire.Struct{}), nil
}

type errorEnveloper struct {
	envelopeType wire.EnvelopeType
	err          error
}

func (errorEnveloper) MethodName() string {
	return "someMethod"
}

func (e errorEnveloper) EnvelopeType() wire.EnvelopeType {
	return e.envelopeType
}

func (e errorEnveloper) ToWire() (wire.Value, error) {
	return wire.Value{}, e.err
}

type errorResponder struct {
	err error
}

func (e errorResponder) EncodeResponse(v wire.Value, et wire.EnvelopeType, w io.Writer) error {
	return e.err
}
