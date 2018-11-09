// Copyright (c) 2018 Uber Technologies, Inc.
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

package yarpcthrift

import (
	"context"

	"go.uber.org/thriftrw/wire"
	yarpc "go.uber.org/yarpc/v2"
	"go.uber.org/yarpc/v2/yarpcerror"
)

var _ yarpc.UnaryEncodingHandler = EncodingHandler(nil)

// EncodingHandler wraps a Thrift handler into a yarpc.UnaryEncodingHandler.
type EncodingHandler func(context.Context, wire.Value) (Response, error)

// Handle implements yarpc.UnaryEncodingHandler.
func (e EncodingHandler) Handle(ctx context.Context, reqBody interface{}) (interface{}, error) {
	reqValue, ok := reqBody.(wire.Value)
	if !ok {
		return nil, yarpcerror.InternalErrorf("tried to handle a non-wire.Value in thrift handler")
	}

	thriftRes, err := e(ctx, reqValue)
	if err != nil {
		return nil, err
	}

	if resType := thriftRes.Body.EnvelopeType(); resType != wire.Reply {
		return nil, errUnexpectedEnvelopeType(resType)
	}

	resValue, err := thriftRes.Body.ToWire()
	if err != nil {
		return nil, err
	}

	return resValue, thriftRes.Exception
}
