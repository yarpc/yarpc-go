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

package observerware

import (
	"context"
	"time"

	"go.uber.org/yarpc/api/transport"
)

type fakeHandler struct {
	err error
}

func (h fakeHandler) Handle(_ context.Context, _ *transport.Request, _ transport.ResponseWriter) error {
	return h.err
}

func (h fakeHandler) HandleOneway(_ context.Context, _ *transport.Request, _ transport.ResponseWriter) error {
	return h.err
}

type fakeOutbound struct {
	transport.Outbound

	err error
}

func (o fakeOutbound) Call(_ context.Context, req *transport.Request) (*transport.Response, error) {
	if o.err != nil {
		return nil, o.err
	}
	return &transport.Response{}, nil
}

func stubTime() func() {
	prev := _timeNow
	_timeNow = func() time.Time { return time.Time{} }
	return func() { _timeNow = prev }
}
