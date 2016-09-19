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

type clientConfig struct {
	DisableEnveloping bool
	Multiplexed       bool
}

// ClientOption customizes the behavior of a Thrift client.
type ClientOption interface {
	applyClientOption(*clientConfig)
}

type registerConfig struct {
	DisableEnveloping bool
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

// DisableEnveloping is an option that disables enveloping of Thrift requests
// and responses.
//
// It may be specified on the client side when the client is constructed.
//
// 	client := myserviceclient.New(channel, thrift.DisableEnveloping)
//
// It may be specified on the server side when the handler is registered.
//
// 	thrift.Register(dispatch, myserviceserver.New(handler), thrift.DisableEnveloping)
var DisableEnveloping Option = disableEnvelopingOption{}

type disableEnvelopingOption struct{}

func (disableEnvelopingOption) applyClientOption(c *clientConfig) {
	c.DisableEnveloping = true
}

func (disableEnvelopingOption) applyRegisterOption(c *registerConfig) {
	c.DisableEnveloping = true
}

// Multiplexed is an option that specifies that requests from a client should
// use Thrift multiplexing. This option should be used if the remote server is
// using Thrift's TMultiplexedProtocol. It includes the name of the service in
// the envelope name for all outbound requests.
//
// Specify this option when constructing the Thrift client.
//
// 	client := myserviceclient.New(channel, thrift.Multiplexed)
var Multiplexed ClientOption = multiplexedOption{}

type multiplexedOption struct{}

func (multiplexedOption) applyClientOption(c *clientConfig) {
	c.Multiplexed = true
}
