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

package tchannel

import (
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/x/config"
)

// TransportConfig configures a shared TChannel transport. This is shared
// between all TChannel outbounds and inbounds of a Dispatcher.
//
// 	transports:
// 	  tchannel:
// 	    address: :4040
type TransportConfig struct {
	// Address to listen on. Defaults to ":0" (all network interfaces and a
	// random OS-assigned port).
	Address string `config:"address,interpolate"`

	// Name of the service for TChannel. This may be omitted to use the
	// Dispatcher's service name.
	Service string `config:"service,interpolate"`
}

// InboundConfig configures a TChannel inbound.
//
// TChannel inbounds do not support any configuration parameters at this time.
type InboundConfig struct{}

// OutboundConfig configures a TChannel outbound.
//
// 	outbounds:
// 	  myservice:
// 	    tchannel:
// 	      address: 127.0.0.1:4040
type OutboundConfig struct {
	Address string `config:"address,interpolate"`
}

// TransportSpec returns a TransportSpec for the TChannel unary transport. See
// TransportConfig, InboundConfig, and OutboundConfig for details on the
// various supported configuration parameters.
func TransportSpec() config.TransportSpec {
	return config.TransportSpec{
		Name:               "tchannel",
		BuildTransport:     buildTransport,
		BuildInbound:       buildInbound,
		BuildUnaryOutbound: buildUnaryOutbound,
	}
}

func buildTransport(tc *TransportConfig, k *config.Kit) (transport.Transport, error) {
	var opts []TransportOption
	if tc.Address != "" {
		opts = append(opts, ListenAddr(tc.Address))
	}
	if tc.Service != "" {
		opts = append(opts, ServiceName(tc.Service))
	}
	return NewTransport(opts...)
}

func buildInbound(_ *InboundConfig, t transport.Transport, k *config.Kit) (transport.Inbound, error) {
	return t.(*Transport).NewInbound(), nil
}

func buildUnaryOutbound(oc *OutboundConfig, t transport.Transport, k *config.Kit) (transport.UnaryOutbound, error) {
	return t.(*Transport).NewSingleOutbound(oc.Address), nil
}

// TODO: Document configuration parameters
