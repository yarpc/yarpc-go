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

package testutils

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"

	"github.com/nightlyone/lockfile"

	"go.uber.org/yarpc"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/transport/http"
	"go.uber.org/yarpc/transport/tchannel"

	"go.uber.org/multierr"
)

const (
	// TransportTypeHTTP represents using HTTP.
	TransportTypeHTTP TransportType = iota
	// TransportTypeTChannel represents using TChannel.
	TransportTypeTChannel

	_numFreePortRetries = 10
)

var (
	// AllTransportTypes are all TransportTypes,
	AllTransportTypes = []TransportType{
		TransportTypeHTTP,
		TransportTypeTChannel,
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
	default:
		return 0, fmt.Errorf("invalid TransportType: %s", s)
	}
}

// GetFreePorts gets count free ports.
func GetFreePorts(count int) ([]uint16, func() error, error) {
	ports := make([]uint16, count)
	funcs := make([]func() error, count)
	for i := 0; i < count; i++ {
		port, close, err := GetFreePort()
		if err != nil {
			return nil, nil, err
		}
		ports[i] = port
		funcs[i] = close
	}
	return ports, func() error {
		var err error
		for _, close := range funcs {
			err = errors.CombineErrors(err, close())
		}
		return err
	}, nil
}

// GetFreePort gets a free port on the system.
//
// The returned function should be called when the user of the port
// has bound themselves to it, the function controls a lock file
// that will not let this function allocate on this port again
// until the caller calls the function.
func GetFreePort() (uint16, func() error, error) {
	for i := 0; i < _numFreePortRetries; i++ {
		address, err := net.ResolveTCPAddr("tcp", "localhost:0")
		if err != nil {
			return 0, nil, err
		}
		listener, err := net.ListenTCP("tcp", address)
		if err != nil {
			return 0, nil, err
		}
		port := uint16(listener.Addr().(*net.TCPAddr).Port)
		if err := listener.Close(); err != nil {
			return 0, nil, err
		}
		lock, err := lockfile.New(filepath.Join(os.TempDir(), fmt.Sprintf("get-free-port.lock.%d", port)))
		if err != nil {
			return 0, nil, err
		}
		if err := lock.TryLock(); err == nil {
			return port, lock.Unlock, nil
		}
	}
	return 0, nil, fmt.Errorf("could not get a port after %d retries", _numFreePortRetries)
}

// WithClientConfig wraps a function by setting up a client and server dispatcher and giving
// the function the client configuration to use in tests for the given TransportType.
//
// The server dispatcher will be brought up using all TransportTypes and with the serviceName.
// The client dispatcher will be brought up using the given TransportType for Unary, HTTP for
// Oneway, and the serviceName with a "-client" suffix.
func WithClientConfig(serviceName string, procedures []transport.Procedure, transportType TransportType, f func(transport.ClientConfig) error) (err error) {
	dispatcherConfig, close, err := NewDispatcherConfig(serviceName)
	if err != nil {
		return err
	}
	defer func() {
		err = multierr.Append(err, close())
	)()

	serverDispatcher, err := NewServerDispatcher(procedures, dispatcherConfig)
	if err != nil {
		return err
	}

	clientDispatcher, err := NewClientDispatcher(transportType, dispatcherConfig)
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

	return f(clientDispatcher.ClientConfig(serviceName))
}

// NewClientDispatcher returns a new client Dispatcher.
//
// HTTP always will be configured as an outbound for Oneway
func NewClientDispatcher(transportType TransportType, config *DispatcherConfig) (*yarpc.Dispatcher, error) {
	port, err := config.GetPort(transportType)
	if err != nil {
		return nil, err
	}
	httpPort, err := config.GetPort(TransportTypeHTTP)
	if err != nil {
		return nil, err
	}
	var onewayOutbound transport.OnewayOutbound
	var unaryOutbound transport.UnaryOutbound
	switch transportType {
	case TransportTypeTChannel:
		tchannelTransport, err := tchannel.NewChannelTransport(tchannel.ServiceName(config.GetServiceName()))
		if err != nil {
			return nil, err
		}
		onewayOutbound = http.NewTransport().NewSingleOutbound(fmt.Sprintf("http://127.0.0.1:%d", httpPort))
		unaryOutbound = tchannelTransport.NewSingleOutbound(fmt.Sprintf("127.0.0.1:%d", port))
	case TransportTypeHTTP:
		httpOutbound := http.NewTransport().NewSingleOutbound(fmt.Sprintf("http://127.0.0.1:%d", port))
		onewayOutbound = httpOutbound
		unaryOutbound = httpOutbound
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
				},
			},
		},
	), nil
}

// NewServerDispatcher returns a new server Dispatcher.
func NewServerDispatcher(procedures []transport.Procedure, config *DispatcherConfig) (*yarpc.Dispatcher, error) {
	tchannelPort, err := config.GetPort(TransportTypeTChannel)
	if err != nil {
		return nil, err
	}
	httpPort, err := config.GetPort(TransportTypeHTTP)
	if err != nil {
		return nil, err
	}
	tchannelTransport, err := tchannel.NewChannelTransport(
		tchannel.ServiceName(config.GetServiceName()),
		tchannel.ListenAddr(fmt.Sprintf("127.0.0.1:%d", tchannelPort)),
	)
	if err != nil {
		return nil, err
	}
	dispatcher := yarpc.NewDispatcher(
		yarpc.Config{
			Name: config.GetServiceName(),
			Inbounds: yarpc.Inbounds{
				tchannelTransport.NewInbound(),
				http.NewTransport().NewInbound(fmt.Sprintf("127.0.0.1:%d", httpPort)),
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
func NewDispatcherConfig(serviceName string) (*DispatcherConfig, func() error, error) {
	transportTypeToPort, close, err := getTransportTypeToPort()
	if err != nil {
		return nil, nil, err
	}
	return &DispatcherConfig{
		serviceName,
		transportTypeToPort,
	}, close, nil
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

func getTransportTypeToPort() (map[TransportType]uint16, func() error, error) {
	ports, close, err := GetFreePorts(len(AllTransportTypes))
	if err != nil {
		return nil, nil, err
	}
	m := make(map[TransportType]uint16, len(AllTransportTypes))
	for i, transportType := range AllTransportTypes {
		m[transportType] = ports[i]
	}
	return m, close, nil
}
