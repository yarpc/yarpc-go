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
	"fmt"
	"net"

	"go.uber.org/yarpc/api/peer"
	"go.uber.org/yarpc/api/transport"
	peerchooser "go.uber.org/yarpc/peer"
	"go.uber.org/yarpc/peer/hostport"
	"go.uber.org/yarpc/yarpcconfig"
	"google.golang.org/grpc/credentials"
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
	Backoff              yarpcconfig.Backoff `config:"backoff"`
}

// InboundConfig configures a gRPC Inbound.
//
// inbounds:
//   grpc:
//     address: ":80"
//
// A gRPC inbound can also enable TLS from key and cert files.
//
// inbounds:
//   grpc:
//     address: ":443"
//     tls:
//       enabled: true
//       keyFile: "/path/to/key"
//       certFile: "/path/to/cert"
type InboundConfig struct {
	// Address to listen on. This field is required.
	Address string           `config:"address,interpolate"`
	TLS     InboundTLSConfig `config:"tls,optional"`
}

func (c InboundConfig) inboundOptions() ([]InboundOption, error) {
	return c.TLS.inboundOptions()
}

// InboundTLSConfig configures a gRPC inbound TLS credentials.
type InboundTLSConfig struct {
	Enabled  bool   `config:"enabled,optional"`
	CertFile string `config:"certFile,optional"`
	KeyFile  string `config:"keyFile,optional"`
}

func (c InboundTLSConfig) inboundOptions() ([]InboundOption, error) {
	if c.Enabled {
		creds, err := c.newInboundCredentials()
		if err != nil {
			return nil, err
		}
		return []InboundOption{InboundCredentials(creds)}, nil
	}
	return nil, nil
}

func (c InboundTLSConfig) newInboundCredentials() (credentials.TransportCredentials, error) {
	if c.CertFile != "" && c.KeyFile != "" {
		return credentials.NewServerTLSFromFile(c.CertFile, c.KeyFile)
	}
	return nil, fmt.Errorf("both certFile and keyFile are necessary to construct gRPC transport credentials, got certFile=%q and keyFile=%q", c.CertFile, c.KeyFile)
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
//        tls:
//          enabled: true
//        round-robin:
//          peers:
//            - 127.0.0.1:8080
//            - 127.0.0.1:8081
//
// A gRPC outbound can enable TLS using system cert.Pool.
//
//  outbounds:
//    mysecureservice:
//      grpc:
//        address: ":443"
//        tls:
//          enabled: true
//
type OutboundConfig struct {
	yarpcconfig.PeerChooser

	// Address to connect to if no peer options set.
	Address string            `config:"address,interpolate"`
	TLS     OutboundTLSConfig `config:"tls,optional"`
}

func (c OutboundConfig) dialOptions() []DialOption {
	return c.TLS.dialOptions()
}

// OutboundTLSConfig configures TLS for a gRPC outbound.
type OutboundTLSConfig struct {
	Enabled bool `config:"enabled,optional"`
}

func (c OutboundTLSConfig) dialOptions() []DialOption {
	if !c.Enabled {
		return nil
	}
	creds := credentials.NewClientTLSFromCert(nil, "")
	option := DialerCredentials(creds)
	return []DialOption{option}
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
	inboundOptions, err := inboundConfig.inboundOptions()
	if err != nil {
		return nil, fmt.Errorf("cannot build gRPC inbound from given configuration: %s", err)
	}
	return trans.NewInbound(listener, append(t.InboundOptions, inboundOptions...)...), nil
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

	dialer := trans.NewDialer(outboundConfig.dialOptions()...)

	var chooser peer.Chooser
	if outboundConfig.Empty() {
		if outboundConfig.Address == "" {
			return nil, newRequiredFieldMissingError("address")
		}
		chooser = peerchooser.NewSingle(hostport.PeerIdentifier(outboundConfig.Address), dialer)
	} else {
		var err error
		chooser, err = outboundConfig.BuildPeerChooser(dialer, hostport.Identify, kit)
		if err != nil {
			return nil, err
		}
	}

	return trans.NewOutbound(chooser, t.OutboundOptions...), nil
}

func newTransportCastError(tr transport.Transport) error {
	return fmt.Errorf("could not cast %T to a *grpc.Transport", tr)
}

func newRequiredFieldMissingError(field string) error {
	return fmt.Errorf("required field missing: %s", field)
}
