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

package tch

import (
	"time"

	"github.com/uber/tchannel-go"
	"github.com/uber/tchannel-go/raw"
	"go.uber.org/yarpc/internal/errors"
	"golang.org/x/net/context"
)

// handlerTimeoutRawHandler returns a handler timeout to the client right away.
// On the other side, one can test if a yarpc client interpret the error
// properly.
type handlerTimeoutRawHandler struct{}

func (handlerTimeoutRawHandler) Handle(ctx context.Context, args *raw.Args) (*raw.Res, error) {
	start := time.Now()
	err := errors.HandlerTimeoutError("caller", "service",
		"handlertimeout/raw", time.Now().Sub(start))
	err = errors.AsHandlerError("service", "handlertimeout/raw", err)
	err = tchannel.NewSystemError(tchannel.ErrCodeTimeout, err.Error())
	return nil, err
}

func (handlerTimeoutRawHandler) OnError(ctx context.Context, err error) {
	onError(ctx, err)
}
