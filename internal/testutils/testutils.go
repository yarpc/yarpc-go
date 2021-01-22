// Copyright (c) 2021 Uber Technologies, Inc.
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

package testutils

import (
	"fmt"
	"net"
	"strconv"

	"go.uber.org/multierr"
	"go.uber.org/yarpc"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/encoding/protobuf"
	"go.uber.org/yarpc/internal/grpcctx"
	"go.uber.org/yarpc/transport/grpc"
	"go.uber.org/yarpc/transport/http"
	"go.uber.org/yarpc/transport/tchannel"
	"go.uber.org/zap"
	ggrpc "google.golang.org/grpc"
)

const (
	// TransportTypeHTTP represents using HTTP.
	TransportTypeHTTP TransportType = iota
	// TransportTypeTChannel represents using TChannel.
	TransportTypeTChannel
	// TransportTypeGRPC represents using GRPC.
	TransportTypeGRPC
)

var (
	// AllTransportTypes are all TransportTypes,
	AllTransportTypes = []TransportType{
		TransportTypeHTTP,
		TransportTypeTChannel,
		TransportTypeGRPC,
	}
)

// TransportType is a transport type.
type TransportType int

// String returns a string representation of t.
func (t TransportType) String() string {
	switch t {
	case TransportTypeHTTP:
		return "http"
	case TransportTypeTChannel:
		return "tchannel"
	case TransportTypeGRPC:
		return "grpc"
	default:
		return strconv.Itoa(int(t))
	}
}

// ParseTransportType parses a transport type from a string.
func ParseTransportType(s string) (TransportType, error) {
	switch s {
	case "http":
		return TransportTypeHTTP, nil
	case "tchannel":
		return TransportTypeTChannel, nil
	case "grpc":
		return TransportTypeGRPC, nil
	default:
		return 0, fmt.Errorf("invalid TransportType: %s", s)
	}
}

// ClientInfo holds the client info for testing.
type ClientInfo struct {
	ClientConfig   transport.ClientConfig
	GRPCClientConn *ggrpc.ClientConn
	ContextWrapper *grpcctx.ContextWrapper
}

// WithClientInfo wraps a function by setting up a client and server dispatcher and giving
// the function the client configuration to use in tests for the given TransportType.
//
// The server dispatcher will be brought up using all TransportTypes and with the serviceName.
// The client dispatcher will be brought up using the given TransportType for Unary, HTTP for
// Oneway, and the serviceName with a "-client" suffix.
func WithClientInfo(serviceName string, procedures []transport.Procedure, transportType TransportType, logger *zap.Logger, f func(*ClientInfo) error) (err error) {
	if logger == nil {
		logger = zap.NewNop()
	}
	dispatcherConfig, err := NewDispatcherConfig(serviceName)
	if err != nil {
		return err
	}
	serverDispatcher, err := NewServerDispatcher(procedures, dispatcherConfig, logger)
	if err != nil {
		return err
	}

	clientDispatcher, err := NewClientDispatcher(transportType, dispatcherConfig, logger)
	if err != nil {
		return err
	}

	if err := serverDispatcher.Start(); err != nil {
		return err
	}
	defer func() { err = multierr.Append(err, serverDispatcher.Stop()) }()

	if err := clientDispatcher.Start(); err != nil {
		return err
	}
	defer func() { err = multierr.Append(err, clientDispatcher.Stop()) }()
	grpcPort, err := dispatcherConfig.GetPort(TransportTypeGRPC)
	if err != nil {
		return err
	}
	grpcClientConn, err := ggrpc.Dial(fmt.Sprintf("127.0.0.1:%d", grpcPort), ggrpc.WithInsecure())
	if err != nil {
		return err
	}
	return f(
		&ClientInfo{
			clientDispatcher.ClientConfig(serviceName),
			grpcClientConn,
			grpcctx.NewContextWrapper().
				WithCaller(serviceName + "-client").
				WithService(serviceName).
				WithEncoding(string(protobuf.Encoding)),
		},
	)
}

// NewClientDispatcher returns a new client Dispatcher.
//
// HTTP always will be configured as an outbound for Oneway.
// gRPC always will be configured as an outbound for Stream.
func NewClientDispatcher(transportType TransportType, config *DispatcherConfig, logger *zap.Logger) (*yarpc.Dispatcher, error) {
	port, err := config.GetPort(transportType)
	if err != nil {
		return nil, err
	}
	httpPort, err := config.GetPort(TransportTypeHTTP)
	if err != nil {
		return nil, err
	}
	grpcPort, err := config.GetPort(TransportTypeGRPC)
	if err != nil {
		return nil, err
	}
	onewayOutbound := http.NewTransport(http.Logger(logger)).NewSingleOutbound(fmt.Sprintf("http://127.0.0.1:%d", httpPort))
	streamOutbound := grpc.NewTransport(grpc.Logger(logger)).NewSingleOutbound(fmt.Sprintf("127.0.0.1:%d", grpcPort))
	var unaryOutbound transport.UnaryOutbound
	switch transportType {
	case TransportTypeTChannel:
		tchannelTransport, err := tchannel.NewChannelTransport(tchannel.ServiceName(config.GetServiceName()), tchannel.Logger(logger))
		if err != nil {
			return nil, err
		}
		unaryOutbound = tchannelTransport.NewSingleOutbound(fmt.Sprintf("127.0.0.1:%d", port))
	case TransportTypeHTTP:
		unaryOutbound = onewayOutbound
	case TransportTypeGRPC:
		unaryOutbound = streamOutbound
	default:
		return nil, fmt.Errorf("invalid TransportType: %v", transportType)
	}
	return yarpc.NewDispatcher(
		yarpc.Config{
			Name: fmt.Sprintf("%s-client", config.GetServiceName()),
			Outbounds: yarpc.Outbounds{
				config.GetServiceName(): {
					Oneway: onewayOutbound,
					Unary:  unaryOutbound,
					Stream: streamOutbound,
				},
			},
		},
	), nil
}

// NewServerDispatcher returns a new server Dispatcher.
func NewServerDispatcher(procedures []transport.Procedure, config *DispatcherConfig, logger *zap.Logger) (*yarpc.Dispatcher, error) {
	tchannelPort, err := config.GetPort(TransportTypeTChannel)
	if err != nil {
		return nil, err
	}
	httpPort, err := config.GetPort(TransportTypeHTTP)
	if err != nil {
		return nil, err
	}
	grpcPort, err := config.GetPort(TransportTypeGRPC)
	if err != nil {
		return nil, err
	}
	tchannelTransport, err := tchannel.NewChannelTransport(
		tchannel.ServiceName(config.GetServiceName()),
		tchannel.ListenAddr(fmt.Sprintf("127.0.0.1:%d", tchannelPort)),
		tchannel.Logger(logger),
	)
	if err != nil {
		return nil, err
	}
	grpcListener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", grpcPort))
	if err != nil {
		return nil, err
	}
	dispatcher := yarpc.NewDispatcher(
		yarpc.Config{
			Name: config.GetServiceName(),
			Inbounds: yarpc.Inbounds{
				tchannelTransport.NewInbound(),
				http.NewTransport(http.Logger(logger)).NewInbound(fmt.Sprintf("127.0.0.1:%d", httpPort)),
				grpc.NewTransport(grpc.Logger(logger)).NewInbound(grpcListener),
			},
		},
	)
	dispatcher.Register(procedures)
	return dispatcher, nil
}

// DispatcherConfig is the configuration for a Dispatcher.
type DispatcherConfig struct {
	serviceName         string
	transportTypeToPort map[TransportType]uint16
}

// NewDispatcherConfig returns a new DispatcherConfig with assigned ports.
func NewDispatcherConfig(serviceName string) (*DispatcherConfig, error) {
	transportTypeToPort, err := getTransportTypeToPort()
	if err != nil {
		return nil, err
	}
	return &DispatcherConfig{
		serviceName,
		transportTypeToPort,
	}, nil
}

// GetServiceName gets the service name.
func (d *DispatcherConfig) GetServiceName() string {
	return d.serviceName
}

// GetPort gets the port for the TransportType.
func (d *DispatcherConfig) GetPort(transportType TransportType) (uint16, error) {
	port, ok := d.transportTypeToPort[transportType]
	if !ok {
		return 0, fmt.Errorf("no port for TransportType %v", transportType)
	}
	return port, nil
}

func getTransportTypeToPort() (map[TransportType]uint16, error) {
	m := make(map[TransportType]uint16, len(AllTransportTypes))
	for _, transportType := range AllTransportTypes {
		port, err := getFreePort()
		if err != nil {
			return nil, err
		}
		m[transportType] = port
	}
	return m, nil
}

func getFreePort() (uint16, error) {
	address, err := net.ResolveTCPAddr("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}

	listener, err := net.ListenTCP("tcp", address)
	if err != nil {
		return 0, err
	}
	port := uint16(listener.Addr().(*net.TCPAddr).Port)
	if err := listener.Close(); err != nil {
		return 0, err
	}
	return port, nil
}
