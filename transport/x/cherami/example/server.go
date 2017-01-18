// Copyright (c) 2016 Uber Technologies, Inc.
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
	"fmt"

	"go.uber.org/yarpc"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/transport/x/cherami"
	"go.uber.org/yarpc/transport/x/cherami/example/thrift/example/exampleserviceserver"
)

// ServerConfig is the configuration needed to create a ExampleService
type ServerConfig struct {
	destination   string
	consumerGroup string
}

// Service demos a service that implements an oneway procedure(defined in thrift/example.thrift)
type Service struct {
	dispatcher *yarpc.Dispatcher
	config     ServerConfig
}

// NewService creates a example service
func NewService(config ServerConfig) *Service {
	return &Service{
		config: config,
	}
}

// Start starts the service
func (s *Service) Start() error {
	// frontend ip is only needed in local testing
	// for a real production server, frontend ip can be empty and hyperbahn will be used to connect to cherami
	frontend, err := getLocalIP()
	if err != nil {
		return nil
	}
	port := 4922

	// Server side needs to start the server dispatcher and register itself which implements the procedures
	s.dispatcher = yarpc.NewDispatcher(yarpc.Config{
		Name: "server",
		Inbounds: []transport.Inbound{cherami.NewInbound(cherami.InboundConfig{
			Destination:   s.config.destination,
			ConsumerGroup: s.config.consumerGroup,
			Frontend:      frontend,
			Port:          port,
		})},
	})
	if err := s.dispatcher.Start(); err != nil {
		return err
	}

	serverCalled = make(chan string)

	// Now register the service to dispatcher
	s.dispatcher.Register(exampleserviceserver.New(&Service{}))
	return nil
}

// Stop stops the service
func (s *Service) Stop() error {
	return s.dispatcher.Stop()
}

var serverCalled chan string

// Award implements the oneway function defined in thrift
func (s *Service) Award(ctx context.Context, Token *string) error {
	fmt.Println(`Got token: `, *Token)
	serverCalled <- *Token
	return nil
}
