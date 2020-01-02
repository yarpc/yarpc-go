// Code generated by thriftrw-plugin-yarpc
// @generated

// Copyright (c) 2020 Uber Technologies, Inc.
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

// Package keyvaluefx provides better integration for Fx for services
// implementing or calling KeyValue.
//
// Clients
//
// If you are making requests to KeyValue, use the Client function to inject a
// KeyValue client into your container.
//
// 	fx.Provide(keyvaluefx.Client("..."))
//
// Servers
//
// If you are implementing KeyValue, provide a keyvalueserver.Interface into
// the container and use the Server function.
//
// Given,
//
// 	func NewKeyValueHandler() keyvalueserver.Interface
//
// You can do the following to have the procedures of KeyValue made available
// to an Fx application.
//
// 	fx.Provide(
// 		NewKeyValueHandler,
// 		keyvaluefx.Server(),
// 	)
package keyvaluefx
