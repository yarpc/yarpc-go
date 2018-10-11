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
	"context"
	"math"
	"net"

	"github.com/opentracing/opentracing-go"
	"go.uber.org/yarpc/v2"
	"go.uber.org/yarpc/v2/yarpcerror"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const (
	// defensive programming: these are copied from grpc-go but we set them
	// explicitly here in case these change in grpc-go so that YARPC stays
	// consistent.
	defaultServerMaxRecvMsgSize = 1024 * 1024 * 4
	defaultServerMaxSendMsgSize = math.MaxInt32
)

var errRouterNotSet = yarpcerror.Newf(yarpcerror.CodeInternal, "gRPC router not set")

// Inbound receives YARPC requests using a gRPC server.
type Inbound struct {
	// Listener is an open listener for inbound gRPC requests.
	Listener net.Listener

	// Addr is a host:port on which to listen if no Listener is provided.
	Addr string

	// Router is the router to handle requests.
	Router yarpc.Router

	// ServerMaxRecvMsgSize is the maximum message size the server can receive.
	//
	// The default is 4MB.
	ServerMaxRecvMsgSize int

	// ServerMaxSendMsgSize is the maximum message size the server can send.
	//
	// The default is unlimited.
	ServerMaxSendMsgSize int

	// Credentials specifies connection level security credentials (e.g.,
	// TLS/SSL) for incoming connections.
	Credentials credentials.TransportCredentials

	// Tracer specifies the tracer to use.
	//
	// By default, opentracing.GlobalTracer() is used.
	Tracer opentracing.Tracer

	// Logger configures a logger for the inbound.
	Logger *zap.Logger

	server *grpc.Server
}

// Start starts the gRPC inbound.
func (i *Inbound) Start(_ context.Context) error {
	if i.Router == nil {
		return errRouterNotSet
	}

	if i.Logger == nil {
		i.Logger = zap.NewNop()
	}
	if i.Tracer == nil {
		i.Tracer = opentracing.GlobalTracer()
	}

	// initialize gRPC options
	serverOptions := []grpc.ServerOption{
		grpc.CustomCodec(customCodec{}),
		grpc.UnknownServiceHandler(newHandler(i).handle),
	}

	serverMaxRecvMsgSize := defaultServerMaxRecvMsgSize
	if i.ServerMaxRecvMsgSize != 0 {
		serverMaxRecvMsgSize = i.ServerMaxRecvMsgSize
	}
	serverOptions = append(serverOptions, grpc.MaxRecvMsgSize(serverMaxRecvMsgSize))

	serverMaxSendMsgSize := defaultServerMaxSendMsgSize
	if i.ServerMaxSendMsgSize != 0 {
		serverMaxSendMsgSize = i.ServerMaxSendMsgSize
	}
	serverOptions = append(serverOptions, grpc.MaxSendMsgSize(serverMaxSendMsgSize))

	if i.Credentials != nil {
		serverOptions = append(serverOptions, grpc.Creds(i.Credentials))
	}
	server := grpc.NewServer(serverOptions...)

	// create a listener at addr if no listener was given
	if i.Listener == nil {
		var err error
		i.Listener, err = net.Listen("tcp", i.Addr)
		if err != nil {
			return err
		}
	}

	// start gRPC server
	go func() {
		i.Logger.Info("started GRPC inbound", zap.Stringer("address", i.Listener.Addr()))
		if len(i.Router.Procedures()) == 0 {
			i.Logger.Warn("no procedures specified for GRPC inbound")
		}
		// TODO there should be some mechanism to block here
		// there is a race because the listener gets set in the grpc
		// Server implementation and we should be able to block
		// until Serve initialization is done
		//
		// It would be even better if we could do this outside the
		// lock in i
		//
		// TODO Server always returns a non-nil error but should
		// we do something with some or all errors?
		_ = server.Serve(i.Listener)
	}()

	i.server = server
	return nil
}

// Stop stops the gRPC inbound.
func (i *Inbound) Stop(_ context.Context) error {
	if i.server != nil {
		i.server.GracefulStop()
	}
	i.server = nil
	return nil
}
