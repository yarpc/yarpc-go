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

package http

import (
	"fmt"
	"time"

	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/x/config"
)

// TransportSpec returns a TransportSpec for the HTTP transport.
//
// See TransportConfig, InboundConfig, and OutboundConfig for details on the
// different configuration parameters supported by this Transport.
//
// Any Transport, Inbound or Outbound option may be passed to this function.
// These options will be applied BEFORE configuration parameters are
// interpreted. This allows configuration parameters to override Option
// provided to TransportSpec.
func TransportSpec(opts ...Option) config.TransportSpec {
	// TODO: Presets. Support "with:" and allow passing those in using
	// varargs on TransportSpec().
	var ts transportSpec
	for _, o := range opts {
		switch opt := o.(type) {
		case TransportOption:
			ts.TransportOptions = append(ts.TransportOptions, opt)
		case InboundOption:
			ts.InboundOptions = append(ts.InboundOptions, opt)
		case OutboundOption:
			ts.OutboundOptions = append(ts.OutboundOptions, opt)
		default:
			panic(fmt.Sprintf("unknown option of type %T: %v", o, o))
		}
	}
	return ts.Spec()
}

// transportSpec holds the configurable parts of the HTTP TransportSpec.
//
// These are usually runtime dependencies that cannot be parsed from
// configuration.
type transportSpec struct {
	TransportOptions []TransportOption
	InboundOptions   []InboundOption
	OutboundOptions  []OutboundOption
}

func (ts *transportSpec) Spec() config.TransportSpec {
	return config.TransportSpec{
		Name:                "http",
		BuildTransport:      ts.buildTransport,
		BuildInbound:        ts.buildInbound,
		BuildUnaryOutbound:  ts.buildUnaryOutbound,
		BuildOnewayOutbound: ts.buildOnewayOutbound,
	}
}

// TransportConfig configures the shared HTTP Transport. This is shared
// between all HTTP outbounds and inbounds of a Dispatcher.
//
// 	transports:
// 	  http:
// 	    keepAlive: 30s
//
// All parameters of TransportConfig are optional. This section may be omitted
// in the transports section.
type TransportConfig struct {
	// Specifies the keep-alive period for all HTTP clients. This field is
	// optional.
	KeepAlive time.Duration `config:"keepAlive"`
}

func (ts *transportSpec) buildTransport(tc *TransportConfig, k *config.Kit) (transport.Transport, error) {
	opts := ts.TransportOptions
	if tc.KeepAlive > 0 {
		opts = append(opts, KeepAlive(tc.KeepAlive))
	}
	return NewTransport(opts...), nil
}

// InboundConfig configures an HTTP inbound.
//
// 	inbounds:
// 	  http:
// 	    address: ":80"
type InboundConfig struct {
	// Address to listen on. This field is required.
	Address string `config:"address"`
}

func (ts *transportSpec) buildInbound(ic *InboundConfig, t transport.Transport, k *config.Kit) (transport.Inbound, error) {
	if ic.Address == "" {
		return nil, fmt.Errorf("inbound address is required")
	}
	return t.(*Transport).NewInbound(ic.Address, ts.InboundOptions...), nil
}

// OutboundConfig configures an HTTP outbound.
//
// 	outbounds:
// 	  keyvalueservice:
// 	    http:
// 	      url: "http://127.0.0.1:80/"
//
// The HTTP outbound supports both, Unary and Oneway transport types. To use
// it for only one of these, nest the section inside a "unary" or "onewy"
// section.
//
// 	outbounds:
// 	  keyvalueservice:
// 	    unary:
// 	      http:
// 	        url: "http://127.0.0.1:80/"
type OutboundConfig struct {
	// URL to which requests will be sent for this outbound. This field is
	// required.
	URL string `config:"url"`
}

func (ts *transportSpec) buildOutbound(oc *OutboundConfig, t transport.Transport, k *config.Kit) (*Outbound, error) {
	return t.(*Transport).NewSingleOutbound(oc.URL, ts.OutboundOptions...), nil
}

func (ts *transportSpec) buildUnaryOutbound(oc *OutboundConfig, t transport.Transport, k *config.Kit) (transport.UnaryOutbound, error) {
	return ts.buildOutbound(oc, t, k)
}

func (ts *transportSpec) buildOnewayOutbound(oc *OutboundConfig, t transport.Transport, k *config.Kit) (transport.OnewayOutbound, error) {
	return ts.buildOutbound(oc, t, k)
}
