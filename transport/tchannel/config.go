// Copyright (c) 2022 Uber Technologies, Inc.
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
	"fmt"
	"time"

	"go.uber.org/yarpc/api/transport"
	yarpctls "go.uber.org/yarpc/api/transport/tls"
	"go.uber.org/yarpc/peer/hostport"
	"go.uber.org/yarpc/yarpcconfig"
)

// TransportConfig configures a shared TChannel transport. This is shared
// between all TChannel outbounds and inbounds of a Dispatcher.
//
//  transports:
//    tchannel:
//      connTimeout: 500ms
//      connBackoff:
//        exponential:
//          first: 10ms
//          max: 30s
type TransportConfig struct {
	ConnTimeout time.Duration       `config:"connTimeout"`
	ConnBackoff yarpcconfig.Backoff `config:"connBackoff"`
}

// InboundConfig configures a TChannel inbound.
//
// 	inbounds:
// 	  tchannel:
// 	    address: :4040
//      tls:
//        mode: permissive
//
// At most one TChannel inbound may be defined in a single YARPC service.
type InboundConfig struct {
	// Address to listen on. Defaults to ":0" (all network interfaces and a
	// random OS-assigned port).
	Address string `config:"address,interpolate"`
	// TLS configuration of the inbound.
	TLS InboundTLSConfig `config:"tls"`
}

// InboundTLSConfig specifies the TLS configuration of the tchannel inbound.
type InboundTLSConfig struct {
	// Mode when set to Permissive or Enforced enables TLS inbound and
	// TLS configuration must be passed as an inbound option.
	Mode yarpctls.Mode `config:"mode,interpolate"`
}

// OutboundConfig configures a TChannel outbound.
//
// 	outbounds:
// 	  myservice:
// 	    tchannel:
// 	      peer: 127.0.0.1:4040
type OutboundConfig struct {
	yarpcconfig.PeerChooser
}

// TransportSpec returns a TransportSpec for the TChannel unary transport.
func TransportSpec(opts ...Option) yarpcconfig.TransportSpec {
	var ts transportSpec
	for _, o := range opts {
		switch opt := o.(type) {
		case TransportOption:
			ts.transportOptions = append(ts.transportOptions, opt)
		default:
			panic(fmt.Sprintf("unknown option of type %T: %v", o, o))
		}
	}
	return ts.Spec()
}

// transportSpec holds the configurable parts of the TChannel TransportSpec.
//
// These are usually runtime dependencies that cannot be parsed from
// configuration.
type transportSpec struct {
	transportOptions []TransportOption
}

func (ts *transportSpec) Spec() yarpcconfig.TransportSpec {
	return yarpcconfig.TransportSpec{
		Name:               TransportName,
		BuildTransport:     ts.buildTransport,
		BuildInbound:       ts.buildInbound,
		BuildUnaryOutbound: ts.buildUnaryOutbound,
	}
}

func (ts *transportSpec) buildTransport(tc *TransportConfig, k *yarpcconfig.Kit) (transport.Transport, error) {
	options := newTransportOptions()

	for _, opt := range ts.transportOptions {
		opt(&options)
	}

	if tc.ConnTimeout != 0 {
		options.connTimeout = tc.ConnTimeout
	}

	strategy, err := tc.ConnBackoff.Strategy()
	if err != nil {
		return nil, err
	}
	options.connBackoffStrategy = strategy

	if options.name != "" {
		return nil, fmt.Errorf("TChannel TransportSpec does not accept ServiceName")
	}

	if options.addr != "" {
		return nil, fmt.Errorf("TChannel TransportSpec does not accept ListenAddr")
	}

	if options.ch != nil {
		return nil, fmt.Errorf("TChannel TransportSpec does not accept WithChannel")
	}

	options.name = k.ServiceName()
	return options.newTransport(), nil
}

func (ts *transportSpec) buildInbound(c *InboundConfig, t transport.Transport, k *yarpcconfig.Kit) (transport.Inbound, error) {
	if c.Address == "" {
		return nil, fmt.Errorf("inbound address is required")
	}

	trans := t.(*Transport)
	if trans.addr != "" {
		// We ensure that trans.addr is empty when buildTransport is called,
		// so if the string is non-empty right now, another TChannel inbound
		// already filled it with a value.
		return nil, fmt.Errorf("at most one TChannel inbound may be specified")
	}

	trans.addr = c.Address
	// Override inbound TLS mode when not set by an option.
	if trans.inboundTLSMode == nil {
		trans.inboundTLSMode = &c.TLS.Mode
	}
	return trans.NewInbound(), nil
}

func (ts *transportSpec) buildUnaryOutbound(oc *OutboundConfig, t transport.Transport, k *yarpcconfig.Kit) (transport.UnaryOutbound, error) {
	x := t.(*Transport)
	chooser, err := oc.BuildPeerChooser(x, hostport.Identify, k)
	if err != nil {
		return nil, err
	}
	return x.NewOutbound(chooser), nil
}
