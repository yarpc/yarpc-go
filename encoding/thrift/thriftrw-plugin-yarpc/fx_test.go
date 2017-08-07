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

package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/yarpc"
	"go.uber.org/yarpc/encoding/thrift/thriftrw-plugin-yarpc/internal/tests/atomic/storeclient"
	"go.uber.org/yarpc/encoding/thrift/thriftrw-plugin-yarpc/internal/tests/atomic/storefx"
	"go.uber.org/yarpc/transport/http"
)

func TestFxClient(t *testing.T) {
	d := yarpc.NewDispatcher(yarpc.Config{
		Name: "myservice",
		Outbounds: yarpc.Outbounds{
			"store": {Unary: http.NewTransport().NewSingleOutbound("http://127.0.0.1/yarpc")},
		},
	})

	assert.NotPanics(t, func() {
		p := storefx.Params{
			Provider: d,
		}
		f := storefx.Client("store").(func(storefx.Params) storefx.Result)
		f(p)
	}, "failed to build client")

	assert.Panics(t, func() {
		f := storefx.Client("not-store").(func(*yarpc.Dispatcher) storeclient.Interface)
		f(d)
	}, "expected panic")
}
