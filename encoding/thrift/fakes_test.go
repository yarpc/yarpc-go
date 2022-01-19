// Copyright (c) 2022 Uber Technologies, Inc.
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

	"go.uber.org/thriftrw/protocol/stream"
	"go.uber.org/thriftrw/wire"
)

const _irrelevant = "irrelevant"

type fakeBodyReader struct {
	body string
}

func (f *fakeBodyReader) Decode(sr stream.Reader) error {
	s, err := sr.ReadString()
	f.body = s
	return err
}

// TODO(witriew): fakeEnveloper should be created with a constructor, allowing
// its uses to dictate the returned values for MethodName and encoded string.
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

func (fakeEnveloper) Encode(sw stream.Writer) error {
	return sw.WriteString(_irrelevant)
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

func (e errorEnveloper) Encode(stream.Writer) error {
	return e.err
}

type errorResponder struct {
	err error
}

func (e errorResponder) EncodeResponse(v wire.Value, et wire.EnvelopeType, w io.Writer) error {
	return e.err
}

func (e errorResponder) WriteResponse(wire.EnvelopeType, io.Writer, stream.Enveloper) error {
	return e.err
}
