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

package transport

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type unaryHandlerFunc func(context.Context, *Request, ResponseWriter) error
type onewayHandlerFunc func(context.Context, *Request) error

func (f unaryHandlerFunc) Handle(ctx context.Context, r *Request, w ResponseWriter) error {
	return f(ctx, r, w)
}
func (f onewayHandlerFunc) HandleOneway(ctx context.Context, r *Request) error {
	return f(ctx, r)
}

func TestDispatchUnaryHandlerWithPanic(t *testing.T) {
	msg := "I'm panicking in a unary handler!"
	handler := func(context.Context, *Request, ResponseWriter) error {
		panic(msg)
	}

	err := DispatchUnaryHandler(
		context.Background(),
		unaryHandlerFunc(handler),
		time.Now(),
		&Request{},
		nil)
	expectMsg := fmt.Sprintf("panic: %s", msg)
	assert.Equal(t, err.Error(), expectMsg)
}

func TestDispatchOnewayHandlerWithPanic(t *testing.T) {
	msg := "I'm panicking in a oneway handler!"
	handler := func(context.Context, *Request) error {
		panic(msg)
	}

	err := DispatchOnewayHandler(
		context.Background(),
		onewayHandlerFunc(handler),
		nil)
	expectMsg := fmt.Sprintf("panic: %s", msg)
	assert.Equal(t, err.Error(), expectMsg)
}
