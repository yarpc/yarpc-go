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
	"fmt"

	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/peer/hostport"
	"go.uber.org/yarpc/x/config"

	"github.com/opentracing/opentracing-go"
)

const transportName = "tchannel"

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
// 	      peer: 127.0.0.1:4040
type OutboundConfig struct {
	config.PeerList
}

// TransportSpec returns a TransportSpec for the TChannel unary transport.
func TransportSpec(opts ...Option) config.TransportSpec {
	return (&transportSpec{}).Spec(opts...)
}

// transportSpec holds the configurable parts of the TChannel TransportSpec.
//
// These are usually runtime dependencies that cannot be parsed from
// configuration.
type transportSpec struct {
	transportOptions []TransportOption
}

func (ts *transportSpec) Spec(opts ...Option) config.TransportSpec {
	// Collect transport options
	for _, o := range opts {
		switch opt := o.(type) {
		case TransportOption:
			ts.transportOptions = append(ts.transportOptions, opt)
		default:
			panic(fmt.Sprintf("unknown option of type %T: %v", o, o))
		}
	}

	return config.TransportSpec{
		Name:               transportName,
		BuildTransport:     ts.buildTransport,
		BuildInbound:       ts.buildInbound,
		BuildUnaryOutbound: ts.buildUnaryOutbound,
	}
}

func (ts *transportSpec) buildTransport(tc *TransportConfig, k *config.Kit) (transport.Transport, error) {
	var config transportConfig
	// Default configuration.
	config.tracer = opentracing.GlobalTracer()
	config.name = k.ServiceName()

	// Apply the spec's transport options.
	for _, opt := range ts.transportOptions {
		opt(&config)
	}

	// Override options with configuration.
	if tc.Address != "" {
		config.addr = tc.Address
	}

	return config.newTransport(), nil
}

func (ts *transportSpec) buildInbound(_ *InboundConfig, t transport.Transport, k *config.Kit) (transport.Inbound, error) {
	return t.(*Transport).NewInbound(), nil
}

func (ts *transportSpec) buildUnaryOutbound(oc *OutboundConfig, t transport.Transport, k *config.Kit) (transport.UnaryOutbound, error) {
	x := t.(*Transport)
	chooser, err := oc.PeerList.BuildPeerList(x, hostport.Identify, k)
	if err != nil {
		return nil, err
	}
	return x.NewOutbound(chooser), nil
}

// TODO: Document configuration parameters
