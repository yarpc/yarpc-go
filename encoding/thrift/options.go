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

import "github.com/yarpc/yarpc-go/transport"

type optDisableEnveloping struct{}

// DisableEnvelopingForTransport disables enveloping on the given transport
// options.
//
// Use this from a transport implementation that wishes to disable Thrift
// enveloping.
//
// 	func (myOutbound) Options() (o transport.Options) {
// 		return thrift.DisableEnvelopingForTransport(o)
// 	}
func DisableEnvelopingForTransport(o transport.Options) transport.Options {
	return o.With(optDisableEnveloping{}, true)
}

func isEnvelopingDisabled(o transport.Options) bool {
	v, ok := o.Get(optDisableEnveloping{})
	if !ok {
		return false
	}
	return v.(bool)
}

type clientConfig struct {
	// TODO: disableEnveloping bool
}

// ClientOption customizes the behavior of a Thrift client.
type ClientOption interface {
	applyClientOption(*clientConfig)
}

type registerConfig struct {
	// TODO: disableEnveloping bool
}

// RegisterOption customizes the behavior of a Thrift handler during
// registration.
type RegisterOption interface {
	applyRegisterOption(*registerConfig)
}

// Option unifies options that apply to both, Thrift clients and handlers.
type Option interface {
	ClientOption
	RegisterOption
}
