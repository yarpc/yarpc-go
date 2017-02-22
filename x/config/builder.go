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

package config

import (
	"fmt"
	"strings"

	"go.uber.org/yarpc"
	"go.uber.org/yarpc/api/transport"
)

// Builder TODO
type Builder struct {
	Name       string
	Inbounds   []InboundConfig
	Outbounds  []OutboundConfig
	Transports []TransportConfig
}

// String returns a readable representation of the configuration loaded by the
// builder.
func (b *Builder) String() string {
	var inbounds, outbounds, transports []string
	for _, i := range b.Inbounds {
		inbounds = append(inbounds, fmt.Sprint(i))
	}
	for _, o := range b.Outbounds {
		outbounds = append(outbounds, fmt.Sprint(o))
	}
	for _, t := range b.Transports {
		transports = append(transports, fmt.Sprint(t))
	}
	return fmt.Sprintf(
		"{Name: %q, Inbounds: [%v], Outbounds: [%v], Transports: [%v]}",
		b.Name,
		strings.Join(inbounds, ", "),
		strings.Join(outbounds, ", "),
		strings.Join(transports, ", "),
	)
}

// BuildDispatcher TODO
func (b *Builder) BuildDispatcher() (*yarpc.Dispatcher, error) {
	cfg := yarpc.Config{Name: b.Name, Outbounds: make(yarpc.Outbounds)}

	transports := make(map[string]transport.Transport)
	for _, tcfg := range b.Transports {
		t, err := tcfg.Builder.BuildTransport()
		if err != nil {
			return nil, fmt.Errorf("failed to build transport %q: %v", tcfg.Name, err)
		}
		transports[tcfg.Name] = t
	}

	for _, icfg := range b.Inbounds {
		tname := icfg.TransportName
		// TODO: error if transport not found in map
		inbound, err := icfg.Builder.BuildInbound(transports[tname])
		if err != nil {
			return nil, fmt.Errorf("failed to build inbound %q: %v", tname, err)
		}
		cfg.Inbounds = append(cfg.Inbounds, inbound)
	}

	for _, ocfg := range b.Outbounds {
		outbounds := transport.Outbounds{ServiceName: ocfg.Service}
		if ocfg.Oneway != nil {
			tname := ocfg.Oneway.TransportName
			// TODO: error if transport not found in map
			oneway, err := ocfg.Oneway.Builder.BuildOnewayOutbound(transports[tname])
			if err != nil {
				return nil, fmt.Errorf("failed to build oneway outbound %q: %v", ocfg.Name, err)
			}
			outbounds.Oneway = oneway
		}
		if ocfg.Unary != nil {
			tname := ocfg.Unary.TransportName
			// TODO: error if transport not found in map
			unary, err := ocfg.Unary.Builder.BuildUnaryOutbound(transports[tname])
			if err != nil {
				return nil, fmt.Errorf("failed to build unary outbound %q: %v", ocfg.Name, err)
			}
			outbounds.Unary = unary
		}
		cfg.Outbounds[ocfg.Name] = outbounds
	}

	return yarpc.NewDispatcher(cfg), nil
}

// TransportConfig TODO
type TransportConfig struct {
	Name    string
	Builder TransportBuilder
}

// InboundConfig TODO
type InboundConfig struct {
	TransportName string
	Builder       InboundBuilder
}

// OutboundConfig TODO
type OutboundConfig struct {
	Name    string
	Service string
	Unary   *UnaryOutboundConfig
	Oneway  *OnewayOutboundConfig
}

// UnaryOutboundConfig TODO
type UnaryOutboundConfig struct {
	TransportName string
	Builder       UnaryOutboundBuilder
}

// OnewayOutboundConfig TODO
type OnewayOutboundConfig struct {
	TransportName string
	Builder       OnewayOutboundBuilder
}

// TransportBuilder TODO
type TransportBuilder interface {
	BuildTransport() (transport.Transport, error)
}

// InboundBuilder TODO
type InboundBuilder interface {
	BuildInbound(transport.Transport) (transport.Inbound, error)
}

// UnaryOutboundBuilder TODO
type UnaryOutboundBuilder interface {
	BuildUnaryOutbound(transport.Transport) (transport.UnaryOutbound, error)
}

// OnewayOutboundBuilder TODO
type OnewayOutboundBuilder interface {
	BuildOnewayOutbound(transport.Transport) (transport.OnewayOutbound, error)
}
