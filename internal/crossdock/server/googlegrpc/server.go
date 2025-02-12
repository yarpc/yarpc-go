// Copyright (c) 2025 Uber Technologies, Inc.
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

package googlegrpc

import (
	"log"
	"net"

	"go.uber.org/yarpc/internal/crossdock/crossdockpb"
	"google.golang.org/grpc"
)

var grpcServer *grpc.Server

// Start starts the test grpc-go server.
func Start() {
	if err := start(); err != nil {
		log.Panic(err)
	}
}

// Stop stops the test grpc-go server.
func Stop() {
	if err := stop(); err != nil {
		log.Print(err)
	}
}

func start() error {
	listener, err := net.Listen("tcp", ":8090")
	if err != nil {
		return err
	}
	grpcServer = grpc.NewServer()
	crossdockpb.RegisterEchoServer(grpcServer, newEchoServer())
	go func() {
		if err := grpcServer.Serve(listener); err != nil {
			log.Print(err)
		}
	}()
	return nil
}

func stop() error {
	if grpcServer != nil {
		grpcServer.Stop()
	}
	return nil
}
