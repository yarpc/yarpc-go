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

package grpc

import (
	"fmt"
	"net"
	"time"

	"go.uber.org/yarpc/api/peer"
	"go.uber.org/yarpc/api/transport"
	peerchooser "go.uber.org/yarpc/peer"
	"go.uber.org/yarpc/peer/hostport"
	"go.uber.org/yarpc/yarpcconfig"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"
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
		Name:                TransportName,
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
//      clientMaxHeaderListSize: 1024
//      serverMaxHeaderListSize: 2048
//
// All parameters of TransportConfig are optional. This section
// may be omitted in the transports section.
type TransportConfig struct {
	ServerMaxRecvMsgSize int `config:"serverMaxRecvMsgSize"`
	ServerMaxSendMsgSize int `config:"serverMaxSendMsgSize"`
	ClientMaxRecvMsgSize int `config:"clientMaxRecvMsgSize"`
	ClientMaxSendMsgSize int `config:"clientMaxSendMsgSize"`
	// GRPC header lise size options accept uint32 param.
	// see: https://pkg.go.dev/google.golang.org/grpc#WithMaxHeaderListSize
	ServerMaxHeaderListSize uint32              `config:"serverMaxHeaderListSize"`
	ClientMaxHeaderListSize uint32              `config:"clientMaxHeaderListSize"`
	Backoff                 yarpcconfig.Backoff `config:"backoff"`
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
	TLS     InboundTLSConfig `config:"tls"`
}

func (c InboundConfig) inboundOptions() ([]InboundOption, error) {
	return c.TLS.inboundOptions()
}

// InboundTLSConfig specifies the TLS configuration for the gRPC inbound.
type InboundTLSConfig struct {
	Enabled  bool   `config:"enabled"` // disabled by default
	CertFile string `config:"certFile,interpolate"`
	KeyFile  string `config:"keyFile,interpolate"`
}

func (c InboundTLSConfig) inboundOptions() ([]InboundOption, error) {
	if !c.Enabled {
		return nil, nil
	}
	creds, err := c.newInboundCredentials()
	if err != nil {
		return nil, err
	}
	return []InboundOption{InboundCredentials(creds)}, nil
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
//        round-robin:
//          peers:
//            - 127.0.0.1:8080
//            - 127.0.0.1:8081
//
// A gRPC outbound can enable TLS using the system cert.Pool.
//
//  outbounds:
//    theirsecureservice:
//      grpc:
//        address: ":443"
//        tls:
//          enabled: true
//        compressor: gzip
//        grpc-keepalive:
//          enabled: true
//          time:    10s
//          timeout: 30s
//          permit-without-stream: true
//
type OutboundConfig struct {
	yarpcconfig.PeerChooser

	// Address to connect to if no peer options set.
	Address string            `config:"address,interpolate"`
	TLS     OutboundTLSConfig `config:"tls"`
	// Compressor to use by default if the server side supports it
	Compressor string                  `config:"compressor"`
	Keepalive  OutboundKeepaliveConfig `config:"grpc-keepalive"`
}

func (c OutboundConfig) dialOptions(kit *yarpcconfig.Kit) ([]DialOption, error) {
	opts := c.TLS.dialOptions()
	opts = append(opts, Compressor(kit.Compressor(c.Compressor)))

	keepaliveOpts, err := c.Keepalive.dialOptions()
	if err != nil {
		return nil, err
	}

	opts = append(opts, keepaliveOpts...)
	return opts, nil
}

// OutboundTLSConfig configures TLS for a gRPC outbound.
type OutboundTLSConfig struct {
	Enabled bool `config:"enabled"`
}

func (c OutboundTLSConfig) dialOptions() []DialOption {
	if !c.Enabled {
		return nil
	}
	creds := credentials.NewClientTLSFromCert(nil, "")
	option := DialerCredentials(creds)
	return []DialOption{option}
}

// OutboundKeepaliveConfig configures gRPC keepalive for a gRPC outbound.
type OutboundKeepaliveConfig struct {
	Enabled             bool   `config:"enabled"`
	Time                string `config:"time"`
	Timeout             string `config:"timeout"`
	PermitWithoutStream bool   `config:"permit-without-stream"`
}

func (c OutboundKeepaliveConfig) dialOptions() ([]DialOption, error) {
	if !c.Enabled {
		return nil, nil
	}

	var err error

	// gRPC keepalive expects time to be minimum 10s.
	// read more: https://pkg.go.dev/google.golang.org/grpc/keepalive#ClientParameters
	keepaliveTime := time.Second * 10
	if c.Time != "" {
		keepaliveTime, err = time.ParseDuration(c.Time)
		if err != nil {
			return nil, fmt.Errorf("could not parse gRPC keepalive time: %v", err)
		}
	}

	// gRPC keepalive defaults timeout to 20s.
	// read more: https://pkg.go.dev/google.golang.org/grpc/keepalive#ClientParameters
	keepaliveTimeout := time.Second * 20
	if c.Timeout != "" {
		keepaliveTimeout, err = time.ParseDuration(c.Timeout)
		if err != nil {
			return nil, fmt.Errorf("could not parse gRPC keepalive timeout: %v", err)
		}
	}

	option := KeepaliveParams(keepalive.ClientParameters{
		Time:                keepaliveTime,
		Timeout:             keepaliveTimeout,
		PermitWithoutStream: c.PermitWithoutStream,
	})
	return []DialOption{option}, nil
}

type transportSpec struct {
	TransportOptions []TransportOption
	InboundOptions   []InboundOption
	OutboundOptions  []OutboundOption
	DialOptions      []DialOption
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
		case DialOption:
			transportSpec.DialOptions = append(transportSpec.DialOptions, opt)
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
	if transportConfig.ServerMaxHeaderListSize > 0 {
		options = append(options, ServerMaxHeaderListSize(transportConfig.ServerMaxHeaderListSize))
	}
	if transportConfig.ClientMaxHeaderListSize > 0 {
		options = append(options, ClientMaxHeaderListSize(transportConfig.ClientMaxHeaderListSize))
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
		return nil, fmt.Errorf("cannot build gRPC inbound from given configuration: %v", err)
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

	dialOpts, err := outboundConfig.dialOptions(kit)
	if err != nil {
		return nil, err
	}

	dialer := trans.NewDialer(append(dialOpts, t.DialOptions...)...)
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
	return fmt.Errorf("required field missing: %v", field)
}
