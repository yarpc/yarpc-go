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

package thrift

import "go.uber.org/thriftrw/protocol"

type clientConfig struct {
	ServiceName string
	Protocol    protocol.Protocol
	Enveloping  bool
	Multiplexed bool
}

// ClientOption customizes the behavior of a Thrift client.
type ClientOption interface {
	applyClientOption(*clientConfig)
}

type registerConfig struct {
	ServiceName string
	Protocol    protocol.Protocol
	Enveloping  bool
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

// Enveloped is an option that specifies that Thrift requests and responses
// should be enveloped. It defaults to false.
//
// It may be specified on the client side when the client is constructed.
//
// 	client := myserviceclient.New(clientConfig, thrift.Enveloped)
//
// It may be specified on the server side when the handler is registered.
//
// 	dispatcher.Register(myserviceserver.New(handler, thrift.Enveloped))
//
// Note that you will need to enable enveloping to communicate with Apache
// Thrift HTTP servers.
var Enveloped Option = envelopedOption{}

type envelopedOption struct{}

func (e envelopedOption) applyClientOption(c *clientConfig) {
	c.Enveloping = true
}

func (e envelopedOption) applyRegisterOption(c *registerConfig) {
	c.Enveloping = true
}

// Multiplexed is an option that specifies that requests from a client should
// use Thrift multiplexing. This option should be used if the remote server is
// using Thrift's TMultiplexedProtocol. It includes the name of the service in
// the envelope name for all outbound requests.
//
// Specify this option when constructing the Thrift client.
//
// 	client := myserviceclient.New(clientConfig, thrift.Multiplexed)
//
// This option has no effect if enveloping is disabled.
var Multiplexed ClientOption = multiplexedOption{}

type multiplexedOption struct{}

func (multiplexedOption) applyClientOption(c *clientConfig) {
	c.Multiplexed = true
}

type namedOption struct{ ServiceName string }

func (n namedOption) applyClientOption(c *clientConfig) {
	if c.ServiceName == "" {
		c.ServiceName = n.ServiceName
	}
}

func (n namedOption) applyRegisterOption(c *registerConfig) {
	if c.ServiceName == "" {
		c.ServiceName = n.ServiceName
	}
}

// Named is an option that specifies the name of a thrift.Client.
// This option should be used if a thrift service extends another
// thrift service. Note that the first Named ClientOption will
// trump all other Named options. This ensures that the
// inherited procedures are appropriately labelled with the
// furthest-inheriting service's name.
//
// Specify this option when constructing the Thrift client.
//
//  client := myserviceclient.New(clientConfig, thrift.Named("foo"))
//
// If not specified, the client's inherited procedures will be
// labelled with the service name from which they are inherited.
func Named(n string) Option {
	return namedOption{ServiceName: n}
}

type protocolOption struct{ Protocol protocol.Protocol }

func (p protocolOption) applyClientOption(c *clientConfig) {
	c.Protocol = p.Protocol
}

func (p protocolOption) applyRegisterOption(c *registerConfig) {
	c.Protocol = p.Protocol
}

// Protocol is an option that specifies which Thrift Protocol servers and
// clients should use. It may be specified on the client side when the client
// is constructed,
//
// 	client := myserviceclient.New(clientConfig, thrift.Protocol(protocol.Binary))
//
// It may be specified on the server side when the handler is registered.
//
// 	dispatcher.Register(myserviceserver.New(handler, thrift.Protocol(protocol.Binary)))
//
// It defaults to the Binary protocol.
func Protocol(p protocol.Protocol) Option {
	return protocolOption{Protocol: p}
}
