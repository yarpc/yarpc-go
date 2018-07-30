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

package grpc

import (
	"crypto/tls"
	"fmt"
	"net"

	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/peer/hostport"
	"go.uber.org/yarpc/yarpcconfig"
)

// TransportSpec returns a TransportSpec for the gRPC transport.
//
// See TransportConfig, InboundConfig, and OutboundConfig for details on the
// different configuration parameters supported by this Transport.
//
// Any TransportOption, InboundOption, or OutboundOption may be passed to this function.
// These options will be applied BEFORE configuration parameters are
// interpreted. This allows configuration parameters to override Options
// provided to TransportSpec.
func TransportSpec(opts ...Option) yarpcconfig.TransportSpec {
	transportSpec, err := newTransportSpec(opts...)
	if err != nil {
		panic(err.Error())
	}
	return yarpcconfig.TransportSpec{
		Name:                transportName,
		BuildTransport:      transportSpec.buildTransport,
		BuildInbound:        transportSpec.buildInbound,
		BuildUnaryOutbound:  transportSpec.buildUnaryOutbound,
		BuildStreamOutbound: transportSpec.buildStreamOutbound,
	}
}

// TransportConfig configures a gRPC Transport. This is shared
// between all gRPC inbounds and outbounds of a Dispatcher.
//
//  transports:
//    grpc:
//      backoff:
//        exponential:
//          first: 10ms
//          max: 30s
//
// All parameters of TransportConfig are optional. This section
// may be omitted in the transports section.
type TransportConfig struct {
	ServerMaxRecvMsgSize int                 `config:"serverMaxRecvMsgSize"`
	ServerMaxSendMsgSize int                 `config:"serverMaxSendMsgSize"`
	ClientMaxRecvMsgSize int                 `config:"clientMaxRecvMsgSize"`
	ClientMaxSendMsgSize int                 `config:"clientMaxSendMsgSize"`
	ClientTLS            bool                `config:"clientTLS"`
	Backoff              yarpcconfig.Backoff `config:"backoff"`
}

// InboundConfig configures a gRPC Inbound.
//
// inbounds:
//   grpc:
//     address: ":80"
type InboundConfig struct {
	// Address to listen on. This field is required.
	Address string `config:"address,interpolate"`
}

// OutboundConfig configures a gRPC Outbound.
//
// outbounds:
//   myservice:
//     grpc:
//       address: ":80"
//
// A gRPC outbound can also configure a peer list.
//
//  outbounds:
//    myservice:
//      grpc:
//        round-robin:
//          peers:
//            - 127.0.0.1:8080
//            - 127.0.0.1:8081
type OutboundConfig struct {
	yarpcconfig.PeerChooser

	// Address to connect to if no peer options set.
	Address   string      `config:"address,interpolate"`
	TLSConfig *tls.Config `config:"tls"`
}

type transportSpec struct {
	TransportOptions []TransportOption
	InboundOptions   []InboundOption
	OutboundOptions  []OutboundOption
}

func newTransportSpec(opts ...Option) (*transportSpec, error) {
	transportSpec := &transportSpec{}
	for _, o := range opts {
		switch opt := o.(type) {
		case TransportOption:
			transportSpec.TransportOptions = append(transportSpec.TransportOptions, opt)
		case InboundOption:
			transportSpec.InboundOptions = append(transportSpec.InboundOptions, opt)
		case OutboundOption:
			transportSpec.OutboundOptions = append(transportSpec.OutboundOptions, opt)
		default:
			return nil, fmt.Errorf("unknown option of type %T: %v", o, o)
		}
	}
	return transportSpec, nil
}

func (t *transportSpec) buildTransport(transportConfig *TransportConfig, _ *yarpcconfig.Kit) (transport.Transport, error) {
	options := t.TransportOptions
	if transportConfig.ServerMaxRecvMsgSize > 0 {
		options = append(options, ServerMaxRecvMsgSize(transportConfig.ServerMaxRecvMsgSize))
	}
	if transportConfig.ServerMaxSendMsgSize > 0 {
		options = append(options, ServerMaxSendMsgSize(transportConfig.ServerMaxSendMsgSize))
	}
	if transportConfig.ClientMaxRecvMsgSize > 0 {
		options = append(options, ClientMaxRecvMsgSize(transportConfig.ClientMaxRecvMsgSize))
	}
	if transportConfig.ClientMaxSendMsgSize > 0 {
		options = append(options, ClientMaxSendMsgSize(transportConfig.ClientMaxSendMsgSize))
	}
	if transportConfig.ClientTLS {
		options = append(options, ClientTLS())
	}
	backoffStrategy, err := transportConfig.Backoff.Strategy()
	if err != nil {
		return nil, err
	}
	options = append(options, BackoffStrategy(backoffStrategy))
	return newTransport(newTransportOptions(options)), nil
}

func (t *transportSpec) buildInbound(inboundConfig *InboundConfig, tr transport.Transport, _ *yarpcconfig.Kit) (transport.Inbound, error) {
	trans, ok := tr.(*Transport)
	if !ok {
		return nil, newTransportCastError(tr)
	}
	if inboundConfig.Address == "" {
		return nil, newRequiredFieldMissingError("address")
	}
	listener, err := net.Listen("tcp", inboundConfig.Address)
	if err != nil {
		return nil, err
	}
	return trans.NewInbound(listener, t.InboundOptions...), nil
}

func (t *transportSpec) buildUnaryOutbound(outboundConfig *OutboundConfig, tr transport.Transport, kit *yarpcconfig.Kit) (transport.UnaryOutbound, error) {
	return t.buildOutbound(outboundConfig, tr, kit)
}

func (t *transportSpec) buildStreamOutbound(outboundConfig *OutboundConfig, tr transport.Transport, kit *yarpcconfig.Kit) (transport.StreamOutbound, error) {
	return t.buildOutbound(outboundConfig, tr, kit)
}

func (t *transportSpec) buildOutbound(outboundConfig *OutboundConfig, tr transport.Transport, kit *yarpcconfig.Kit) (*Outbound, error) {
	trans, ok := tr.(*Transport)
	if !ok {
		return nil, newTransportCastError(tr)
	}
	if outboundConfig.Empty() {
		if outboundConfig.Address == "" {
			return nil, newRequiredFieldMissingError("address")
		}
		return trans.NewSingleOutbound(outboundConfig.Address, t.OutboundOptions...), nil
	}

	// Optionally decorate the transport with a TLS configuration.  The peer
	// chooser receives this decorated transport so it can annotate peer
	// identifiers with the desired TLS configuration for individual peers.
	outTrans := trans.WithTLS(outboundConfig.TLSConfig)

	chooser, err := outboundConfig.BuildPeerChooser(outTrans, hostport.Identify, kit)
	if err != nil {
		return nil, err
	}
	return trans.NewOutbound(chooser, t.OutboundOptions...), nil
}

func newTransportCastError(tr transport.Transport) error {
	return fmt.Errorf("could not cast %T to a *grpc.Transport", tr)
}

func newRequiredFieldMissingError(field string) error {
	return fmt.Errorf("required field missing: %s", field)
}
