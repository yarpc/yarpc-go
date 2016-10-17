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

	// Returns an outbound to send the request through.
	//
	// MAY be called multiple times for a request. The returned outbound MUST
	// have already been started.
	GetOutbound() Outbound
	GetOnewayOutbound() OnewayOutbound
}

// RemoteService encapsulates a service's outbounds
type RemoteService struct {
	Name string

	Outbound       Outbound
	OnewayOutbound OnewayOutbound
}

// MultiOutboundChannel constructs a Channel backed by multiple outobund types
func MultiOutboundChannel(caller string, rs RemoteService) Channel {
	return multiOutboundChannel{caller: caller, rs: rs}
}

type multiOutboundChannel struct {
	caller string
	rs     RemoteService
}

func (c multiOutboundChannel) Caller() string  { return c.caller }
func (c multiOutboundChannel) Service() string { return c.rs.Name }

func (c multiOutboundChannel) GetOutbound() Outbound {
	return c.rs.Outbound
}

func (c multiOutboundChannel) GetOnewayOutbound() OnewayOutbound {
	return c.rs.OnewayOutbound
}

// IdentityChannel constructs a simple Channel for the given caller-service pair
// which always returns the given Outbound.
func IdentityChannel(caller, service string, out Outbound) Channel {
	return identityChannel{caller: caller, service: service, outbound: out}
}

type identityChannel struct {
	caller   string
	service  string
	outbound Outbound
}

func (s identityChannel) Caller() string        { return s.caller }
func (s identityChannel) Service() string       { return s.service }
func (s identityChannel) GetOutbound() Outbound { return s.outbound }
func (s identityChannel) GetOnewayOutbound() OnewayOutbound {
	panic("Unsupported GetOnewayOutbound with identityChannel")
}
