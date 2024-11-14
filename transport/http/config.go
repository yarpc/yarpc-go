// Copyright (c) 2024 Uber Technologies, Inc.
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
	"errors"
	"fmt"
	"time"

	"go.uber.org/yarpc/api/transport"
	yarpctls "go.uber.org/yarpc/api/transport/tls"
	"go.uber.org/yarpc/peer/hostport"
	"go.uber.org/yarpc/yarpcconfig"
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
func TransportSpec(opts ...Option) yarpcconfig.TransportSpec {
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

func (ts *transportSpec) Spec() yarpcconfig.TransportSpec {
	return yarpcconfig.TransportSpec{
		Name:                TransportName,
		BuildTransport:      ts.buildTransport,
		BuildInbound:        ts.buildInbound,
		BuildUnaryOutbound:  ts.buildUnaryOutbound,
		BuildOnewayOutbound: ts.buildOnewayOutbound,
	}
}

// TransportConfig configures the shared HTTP Transport. This is shared
// between all HTTP outbounds and inbounds of a Dispatcher.
//
//	transports:
//	  http:
//	    keepAlive: 30s
//	    maxIdleConns: 2
//	    maxIdleConnsPerHost: 2
//	    idleConnTimeout: 90s
//	    disableKeepAlives: false
//	    disableCompression: false
//	    responseHeaderTimeout: 0s
//	    connTimeout: 500ms
//	    connBackoff:
//	      exponential:
//	        first: 10ms
//	        max: 30s
//
// All parameters of TransportConfig are optional. This section may be omitted
// in the transports section.
type TransportConfig struct {
	// Specifies the keep-alive period for all HTTP clients. This field is
	// optional.
	KeepAlive             time.Duration       `config:"keepAlive"`
	MaxIdleConns          int                 `config:"maxIdleConns"`
	MaxIdleConnsPerHost   int                 `config:"maxIdleConnsPerHost"`
	IdleConnTimeout       time.Duration       `config:"idleConnTimeout"`
	DisableKeepAlives     bool                `config:"disableKeepAlives"`
	DisableCompression    bool                `config:"disableCompression"`
	ResponseHeaderTimeout time.Duration       `config:"responseHeaderTimeout"`
	ConnTimeout           time.Duration       `config:"connTimeout"`
	ConnBackoff           yarpcconfig.Backoff `config:"connBackoff"`
}

func (ts *transportSpec) buildTransport(tc *TransportConfig, k *yarpcconfig.Kit) (transport.Transport, error) {
	options := newTransportOptions()

	for _, opt := range ts.TransportOptions {
		opt(&options)
	}

	if options.serviceName == "" {
		options.serviceName = k.ServiceName()
	}
	if tc.KeepAlive > 0 {
		options.keepAlive = tc.KeepAlive
	}
	if tc.MaxIdleConns > 0 {
		options.maxIdleConns = tc.MaxIdleConns
	}
	if tc.MaxIdleConnsPerHost > 0 {
		options.maxIdleConnsPerHost = tc.MaxIdleConnsPerHost
	}
	if tc.IdleConnTimeout > 0 {
		options.idleConnTimeout = tc.IdleConnTimeout
	}
	if tc.DisableKeepAlives {
		options.disableKeepAlives = true
	}
	if tc.DisableCompression {
		options.disableCompression = true
	}
	if tc.ResponseHeaderTimeout > 0 {
		options.responseHeaderTimeout = tc.ResponseHeaderTimeout
	}
	if tc.ConnTimeout > 0 {
		options.connTimeout = tc.ConnTimeout
	}

	strategy, err := tc.ConnBackoff.Strategy()
	if err != nil {
		return nil, err
	}
	options.connBackoffStrategy = strategy

	return options.newTransport(), nil
}

// InboundConfig configures an HTTP inbound.
//
//	inbounds:
//	  http:
//	    address: ":80"
//	    grabHeaders:
//	      - x-foo
//	      - x-bar
//	    shutdownTimeout: 5s
type InboundConfig struct {
	// Address to listen on. This field is required.
	Address string `config:"address,interpolate"`
	// The additional headers, starting with x, that should be
	// propagated to handlers. This field is optional.
	GrabHeaders []string `config:"grabHeaders"`
	// The maximum amount of time to wait for the inbound to shutdown.
	ShutdownTimeout *time.Duration `config:"shutdownTimeout"`
	// TLS configuration of the inbound.
	TLSConfig TLSConfig `config:"tls"`
	// DisableHTTP2 configure to reject http2 requests.
	DisableHTTP2 bool `config:"disableHTTP2"`
}

// TLSConfig specifies the TLS configuration of the HTTP inbound.
type TLSConfig struct {
	// Mode when set to Permissive or Enforced enables TLS inbound and
	// TLS configuration must be passed as an inbound option.
	Mode yarpctls.Mode `config:"mode,interpolate"`
}

func (ts *transportSpec) buildInbound(ic *InboundConfig, t transport.Transport, k *yarpcconfig.Kit) (transport.Inbound, error) {
	if ic.Address == "" {
		return nil, fmt.Errorf("inbound address is required")
	}

	// TLS mode provided in the inbound options takes higher precedence than
	// the TLS mode passed in YAML config.
	inboundOptions := append([]InboundOption{InboundTLSMode(ic.TLSConfig.Mode)}, ts.InboundOptions...)
	if len(ic.GrabHeaders) > 0 {
		inboundOptions = append(inboundOptions, GrabHeaders(ic.GrabHeaders...))
	}

	if ic.ShutdownTimeout != nil {
		if *ic.ShutdownTimeout < 0 {
			return nil, fmt.Errorf("shutdownTimeout must not be negative, got: %q", ic.ShutdownTimeout)
		}
		inboundOptions = append(inboundOptions, ShutdownTimeout(*ic.ShutdownTimeout))
	}

	inboundOptions = append(inboundOptions, DisableHTTP2(ic.DisableHTTP2))

	return t.(*Transport).NewInbound(ic.Address, inboundOptions...), nil
}

// OutboundConfig configures an HTTP outbound.
//
//	outbounds:
//	  keyvalueservice:
//	    http:
//	      url: "http://127.0.0.1:80/"
//
// The HTTP outbound supports both, Unary and Oneway transport types. To use
// it for only one of these, nest the section inside a "unary" or "onewy"
// section.
//
//	outbounds:
//	  keyvalueservice:
//	    unary:
//	      http:
//	        url: "http://127.0.0.1:80/"
//
// An HTTP outbound can also configure a peer list.
// In this case, there can still be a "url" and it serves as a template for the
// HTTP client, expressing whether to use "http:" or "https:" and what path to
// use. The address gets replaced with peers from the peer list.
//
//	outbounds:
//	  keyvalueservice:
//	    unary:
//	      http:
//	        url: "https://address/rpc"
//	        round-robin:
//	          peers:
//	            - 127.0.0.1:8080
//	            - 127.0.0.1:8081
type OutboundConfig struct {
	yarpcconfig.PeerChooser

	// URL to which requests will be sent for this outbound. This field is
	// required.
	URL string `config:"url,interpolate"`
	// HTTP headers that will be added to all requests made through this
	// outbound.
	//
	//  http:
	//    url: "http://localhost:8080/yarpc"
	//    addHeaders:
	//      X-Caller: myserice
	//      X-Token: foo
	AddHeaders map[string]string `config:"addHeaders"`
	// TLS config enables TLS outbound.
	//
	//  http:
	//    url: "http://localhost:8080/yarpc"
	//    tls:
	//      mode: enforced
	//      spiffe-ids:
	//        - destination-id
	TLS OutboundTLSConfig `config:"tls"`
}

// OutboundTLSConfig configures TLS for the HTTP outbound.
type OutboundTLSConfig struct {
	// Mode when set to Enforced enables outbound TLS.
	// Note: outbound TLS configuration provider must be given as an option
	// which is used for fetching client tls.Config.
	Mode yarpctls.Mode `config:"mode,interpolate"`
	// SpiffeIDs is list of accepted server spiffe IDs. This cannot be empty
	// list.
	SpiffeIDs []string `config:"spiffe-ids"`
}

func (o OutboundTLSConfig) options(provider yarpctls.OutboundTLSConfigProvider) ([]OutboundOption, error) {
	if o.Mode == yarpctls.Disabled {
		return nil, nil
	}

	if o.Mode == yarpctls.Permissive {
		return nil, errors.New("outbound does not support permissive TLS mode")
	}

	if provider == nil {
		return nil, errors.New("outbound TLS enforced but outbound TLS config provider is nil")
	}

	config, err := provider.ClientTLSConfig(o.SpiffeIDs)
	if err != nil {
		return nil, err
	}

	return []OutboundOption{OutboundTLSConfiguration(config)}, nil
}

func (ts *transportSpec) buildOutbound(oc *OutboundConfig, t transport.Transport, k *yarpcconfig.Kit) (*Outbound, error) {
	x := t.(*Transport)

	opts := []OutboundOption{OutboundDestinationServiceName(k.OutboundServiceName())}
	opts = append(opts, ts.OutboundOptions...)
	if len(oc.AddHeaders) > 0 {
		for k, v := range oc.AddHeaders {
			opts = append(opts, AddHeader(k, v))
		}
	}

	option, err := oc.TLS.options(x.ouboundTLSConfigProvider)
	if err != nil {
		return nil, err
	}
	opts = append(option, opts...)

	// Special case where the URL implies the single peer.
	if oc.Empty() {
		return x.NewSingleOutbound(oc.URL, opts...), nil
	}

	chooser, err := oc.BuildPeerChooser(x, hostport.Identify, k)
	if err != nil {
		return nil, fmt.Errorf("cannot configure peer chooser for HTTP outbound: %v", err)
	}

	if oc.URL != "" {
		opts = append(opts, URLTemplate(oc.URL))
	}
	return x.NewOutbound(chooser, opts...), nil
}

func (ts *transportSpec) buildUnaryOutbound(oc *OutboundConfig, t transport.Transport, k *yarpcconfig.Kit) (transport.UnaryOutbound, error) {
	return ts.buildOutbound(oc, t, k)
}

func (ts *transportSpec) buildOnewayOutbound(oc *OutboundConfig, t transport.Transport, k *yarpcconfig.Kit) (transport.OnewayOutbound, error) {
	return ts.buildOutbound(oc, t, k)
}
