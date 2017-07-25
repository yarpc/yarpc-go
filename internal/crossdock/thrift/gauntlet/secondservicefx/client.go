// Code generated by thriftrw-plugin-yarpc
// @generated

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

package secondservicefx

import (
	"go.uber.org/fx"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/encoding/thrift"
	"go.uber.org/yarpc/internal/crossdock/thrift/gauntlet/secondserviceclient"
)

// Params defines the dependencies for SecondService client.
type Params struct {
	fx.In

	Provider transport.ClientConfigProvider
}

// Result defines the object SecondService client provides.
type Result struct {
	fx.Out

	Client secondserviceclient.Interface
}

// Client provides a SecondService client to an Fx application using the given name
// for routing.
//
// 	fx.Provide(
// 		secondservicefx.Client("..."),
// 		newHandler,
// 	)
func Client(name string, opts ...thrift.ClientOption) interface{} {
	return func(p Params) Result {
		client := secondserviceclient.New(p.Provider.ClientConfig(name), opts...)
		return Result{Client: client}
	}
}
