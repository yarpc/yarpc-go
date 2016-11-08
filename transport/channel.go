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

package transport

//go:generate mockgen -destination=transporttest/channel.go -package=transporttest go.uber.org/yarpc/transport Channel,ChannelProvider

// ChannelProvider builds channels from the current service to other services.
type ChannelProvider interface {
	// Retrieves a new Channel that will make requests to the given service.
	//
	// This MAY panic if the given service is unknown.
	Channel(service string) Channel
}

// A Channel is a stream of communication between a single caller-service
// pair.
type Channel interface {
	// Name of the service making the request.
	Caller() string

	// Name of the service to which the request is being made.
	Service() string

	// Returns an outbound to send the request through or nil if there is no
	// outbound for this service
	//
	// MAY be called multiple times for a request. The returned outbound MUST
	// have already been started.
	GetUnaryOutbound() UnaryOutbound
	GetOnewayOutbound() OnewayOutbound
}

// MultiOutboundChannel constructs a Channel backed by multiple outobund types
func MultiOutboundChannel(caller, service string, Outbounds Outbounds) Channel {
	return multiOutboundChannel{caller: caller, service: service, Outbounds: Outbounds}
}

type multiOutboundChannel struct {
	caller    string
	service   string
	Outbounds Outbounds
}

func (c multiOutboundChannel) Caller() string  { return c.caller }
func (c multiOutboundChannel) Service() string { return c.service }

func (c multiOutboundChannel) GetUnaryOutbound() UnaryOutbound {
	return c.Outbounds.Unary
}

func (c multiOutboundChannel) GetOnewayOutbound() OnewayOutbound {
	return c.Outbounds.Oneway
}
