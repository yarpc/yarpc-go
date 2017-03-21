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

package example

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"go.uber.org/yarpc"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/internal/examples/protobuf/examplepb"
	"go.uber.org/yarpc/transport/http"
	"go.uber.org/yarpc/transport/tchannel"
)

const (
	// DefaultTChannelPort is the default TChannel port.
	DefaultTChannelPort = 28941
	// DefaultHTTPPort is the default HTTP port.
	DefaultHTTPPort = 24034
	// DefaultServiceName is the default service name.
	DefaultServiceName = "example"

	// ClientTransportTChannel represents using TChannel on the client.
	ClientTransportTChannel ClientTransport = iota
	// ClientTransportHTTP represents using HTTP on the client.
	ClientTransportHTTP
)

var (
	errRequestNil    = errors.New("request nil")
	errRequestKeyNil = errors.New("request key nil")
)

// WithKeyValueClient calls f on a KeyValueClient.
func WithKeyValueClient(clientTransport ClientTransport, f func(examplepb.KeyValueClient) error, options ...DispatcherOption) error {
	serverDispatcher, err := NewServerDispatcher(options...)
	if err != nil {
		return err
	}
	clientDispatcher, err := NewClientDispatcher(clientTransport, options...)
	if err != nil {
		return err
	}
	if err := serverDispatcher.Start(); err != nil {
		return err
	}
	defer serverDispatcher.Stop()
	if err := clientDispatcher.Start(); err != nil {
		return err
	}
	defer clientDispatcher.Stop()
	return f(examplepb.NewKeyValueClient(clientDispatcher.ClientConfig("example")))
}

// NewClientDispatcher returns a new client Dispatcher.
func NewClientDispatcher(clientTransport ClientTransport, options ...DispatcherOption) (*yarpc.Dispatcher, error) {
	dispatcherOptions := newDispatcherOptions(options)
	var outbound transport.UnaryOutbound
	switch clientTransport {
	case ClientTransportTChannel:
		tchannelTransport, err := tchannel.NewChannelTransport(tchannel.ServiceName("example"))
		if err != nil {
			return nil, err
		}
		outbound = tchannelTransport.NewSingleOutbound(fmt.Sprintf("localhost:%d", dispatcherOptions.TChannelPort))
	case ClientTransportHTTP:
		outbound = http.NewTransport().NewSingleOutbound(fmt.Sprintf("http://127.0.0.1:%d", dispatcherOptions.HTTPPort))
	default:
		return nil, fmt.Errorf("invalid client transport: %v", clientTransport)
	}
	return yarpc.NewDispatcher(
		yarpc.Config{
			Name: "example-client",
			Outbounds: yarpc.Outbounds{
				"example": {
					Unary: outbound,
				},
			},
		},
	), nil
}

// NewServerDispatcher returns a new server Dispatcher.
func NewServerDispatcher(options ...DispatcherOption) (*yarpc.Dispatcher, error) {
	dispatcherOptions := newDispatcherOptions(options)
	tchannelTransport, err := tchannel.NewChannelTransport(
		tchannel.ServiceName("example"),
		tchannel.ListenAddr(fmt.Sprintf(":%d", dispatcherOptions.TChannelPort)),
	)
	if err != nil {
		return nil, err
	}
	dispatcher := yarpc.NewDispatcher(
		yarpc.Config{
			Name: "example",
			Inbounds: yarpc.Inbounds{
				tchannelTransport.NewInbound(),
				http.NewTransport().NewInbound(fmt.Sprintf(":%d", dispatcherOptions.HTTPPort)),
			},
		},
	)
	dispatcher.Register(examplepb.BuildKeyValueProcedures(newKeyValueServer()))
	return dispatcher, nil
}

// ClientTransport is a client transport.
type ClientTransport int

// ParseClientTransport parses a client transport from a string.
func ParseClientTransport(s string) (ClientTransport, error) {
	switch s {
	case "tchannel":
		return ClientTransportTChannel, nil
	case "http":
		return ClientTransportHTTP, nil
	default:
		return 0, fmt.Errorf("invalid client transport: %s", s)
	}
}

// DispatcherOption is an option for a Dispatcher.
type DispatcherOption func(*dispatcherOptions)

// WithTChannelPort changes the TChannel port.
func WithTChannelPort(port uint16) DispatcherOption {
	return func(dispatcherOptions *dispatcherOptions) {
		dispatcherOptions.TChannelPort = port
	}
}

// WithHTTPPort changes the HTTP port.
func WithHTTPPort(port uint16) DispatcherOption {
	return func(dispatcherOptions *dispatcherOptions) {
		dispatcherOptions.HTTPPort = port
	}
}

type dispatcherOptions struct {
	TChannelPort uint16
	HTTPPort     uint16
}

func newDispatcherOptions(options []DispatcherOption) *dispatcherOptions {
	dispatcherOptions := &dispatcherOptions{}
	for _, option := range options {
		option(dispatcherOptions)
	}
	if dispatcherOptions.TChannelPort == 0 {
		dispatcherOptions.TChannelPort = DefaultTChannelPort
	}
	if dispatcherOptions.HTTPPort == 0 {
		dispatcherOptions.HTTPPort = DefaultHTTPPort
	}
	return dispatcherOptions
}

type keyValueServer struct {
	sync.RWMutex
	items map[string]string
}

func newKeyValueServer() *keyValueServer {
	return &keyValueServer{sync.RWMutex{}, make(map[string]string)}
}

func (a *keyValueServer) GetValue(ctx context.Context, request *examplepb.GetValueRequest) (*examplepb.GetValueResponse, error) {
	if request == nil {
		return nil, errRequestNil
	}
	if request.Key == "" {
		return nil, errRequestKeyNil
	}
	a.RLock()
	if value, ok := a.items[request.Key]; ok {
		a.RUnlock()
		return &examplepb.GetValueResponse{value}, nil
	}
	a.RUnlock()
	return nil, fmt.Errorf("key not set: %s", request.Key)
}

func (a *keyValueServer) SetValue(ctx context.Context, request *examplepb.SetValueRequest) (*examplepb.SetValueResponse, error) {
	if request == nil {
		return nil, errRequestNil
	}
	if request.Key == "" {
		return nil, errRequestKeyNil
	}
	a.Lock()
	if request.Value == "" {
		delete(a.items, request.Key)
	} else {
		a.items[request.Key] = request.Value
	}
	a.Unlock()
	return nil, nil
}
