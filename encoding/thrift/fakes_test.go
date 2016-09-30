// Copyright (c) 2016 Uber Technologies, Inc.
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
	"fmt"
	"reflect"

	"go.uber.org/yarpc"
	"go.uber.org/yarpc/transport"

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

type fakeReqMeta struct {
	caller    string
	service   string
	procedure string
	encoding  transport.Encoding
	headers   yarpc.Headers
}

func (f fakeReqMeta) Matches(x interface{}) bool {
	reqMeta, ok := x.(yarpc.ReqMeta)
	if !ok {
		return false
	}

	if f.caller != reqMeta.Caller() {
		return false
	}
	if f.service != reqMeta.Service() {
		return false
	}
	if f.procedure != reqMeta.Procedure() {
		return false
	}
	if f.encoding != reqMeta.Encoding() {
		return false
	}
	if !reflect.DeepEqual(f.headers, reqMeta.Headers()) {
		return false
	}
	return true
}

func (f fakeReqMeta) String() string {
	return fmt.Sprintf("%#v", f)
}
