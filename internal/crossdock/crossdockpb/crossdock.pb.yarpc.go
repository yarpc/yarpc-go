// Code generated by protoc-gen-yarpc-go
// source: internal/crossdock/crossdockpb/crossdock.proto
// DO NOT EDIT!

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

package crossdockpb

import (
	"context"
	"reflect"

	"github.com/gogo/protobuf/proto"
	"go.uber.org/yarpc"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/encoding/protobuf"
	"go.uber.org/yarpc/yarpcproto"
)

// EchoYARPCClient is the YARPC client-side interface for the Echo service.
type EchoYARPCClient interface {
	Echo(context.Context, *Ping, ...yarpc.CallOption) (*Pong, error)
}

// NewEchoYARPCClient builds a new YARPC client for the Echo service.
func NewEchoYARPCClient(clientConfig transport.ClientConfig, options ...protobuf.ClientOption) EchoYARPCClient {
	return &_EchoYARPCCaller{protobuf.NewClient(
		protobuf.ClientParams{
			ServiceName:  "uber.yarpc.internal.crossdock.Echo",
			ClientConfig: clientConfig,
			Options:      options,
		},
	)}
}

// EchoYARPCServer is the YARPC server-side interface for the Echo service.
type EchoYARPCServer interface {
	Echo(context.Context, *Ping) (*Pong, error)
}

// BuildEchoYARPCProcedures prepares an implementation of the Echo service for YARPC registration.
func BuildEchoYARPCProcedures(server EchoYARPCServer) []transport.Procedure {
	handler := &_EchoYARPCHandler{server}
	return protobuf.BuildProcedures(
		protobuf.BuildProceduresParams{
			ServiceName: "uber.yarpc.internal.crossdock.Echo",
			UnaryHandlerParams: []protobuf.BuildProceduresUnaryHandlerParams{
				protobuf.BuildProceduresUnaryHandlerParams{
					MethodName: "Echo",
					Handler: protobuf.NewUnaryHandler(
						protobuf.UnaryHandlerParams{
							Handle:     handler.Echo,
							NewRequest: newEcho_EchoYARPCRequest,
						},
					),
				},
			},
			OnewayHandlerParams: []protobuf.BuildProceduresOnewayHandlerParams{},
		},
	)
}

type _EchoYARPCCaller struct {
	client protobuf.Client
}

func (c *_EchoYARPCCaller) Echo(ctx context.Context, request *Ping, options ...yarpc.CallOption) (*Pong, error) {
	responseMessage, err := c.client.Call(ctx, "Echo", request, newEcho_EchoYARPCResponse, options...)
	if responseMessage == nil {
		return nil, err
	}
	response, ok := responseMessage.(*Pong)
	if !ok {
		return nil, protobuf.CastError(emptyEcho_EchoYARPCResponse, responseMessage)
	}
	return response, err
}

type _EchoYARPCHandler struct {
	server EchoYARPCServer
}

func (h *_EchoYARPCHandler) Echo(ctx context.Context, requestMessage proto.Message) (proto.Message, error) {
	var request *Ping
	var ok bool
	if requestMessage != nil {
		request, ok = requestMessage.(*Ping)
		if !ok {
			return nil, protobuf.CastError(emptyEcho_EchoYARPCRequest, requestMessage)
		}
	}
	response, err := h.server.Echo(ctx, request)
	if response == nil {
		return nil, err
	}
	return response, err
}

func newEcho_EchoYARPCRequest() proto.Message {
	return &Ping{}
}

func newEcho_EchoYARPCResponse() proto.Message {
	return &Pong{}
}

var (
	emptyEcho_EchoYARPCRequest  = &Ping{}
	emptyEcho_EchoYARPCResponse = &Pong{}
)

// OnewayYARPCClient is the YARPC client-side interface for the Oneway service.
type OnewayYARPCClient interface {
	Echo(context.Context, *Token, ...yarpc.CallOption) (yarpc.Ack, error)
}

// NewOnewayYARPCClient builds a new YARPC client for the Oneway service.
func NewOnewayYARPCClient(clientConfig transport.ClientConfig, options ...protobuf.ClientOption) OnewayYARPCClient {
	return &_OnewayYARPCCaller{protobuf.NewClient(
		protobuf.ClientParams{
			ServiceName:  "uber.yarpc.internal.crossdock.Oneway",
			ClientConfig: clientConfig,
			Options:      options,
		},
	)}
}

// OnewayYARPCServer is the YARPC server-side interface for the Oneway service.
type OnewayYARPCServer interface {
	Echo(context.Context, *Token) error
}

// BuildOnewayYARPCProcedures prepares an implementation of the Oneway service for YARPC registration.
func BuildOnewayYARPCProcedures(server OnewayYARPCServer) []transport.Procedure {
	handler := &_OnewayYARPCHandler{server}
	return protobuf.BuildProcedures(
		protobuf.BuildProceduresParams{
			ServiceName:        "uber.yarpc.internal.crossdock.Oneway",
			UnaryHandlerParams: []protobuf.BuildProceduresUnaryHandlerParams{},
			OnewayHandlerParams: []protobuf.BuildProceduresOnewayHandlerParams{
				protobuf.BuildProceduresOnewayHandlerParams{
					MethodName: "Echo",
					Handler: protobuf.NewOnewayHandler(
						protobuf.OnewayHandlerParams{
							Handle:     handler.Echo,
							NewRequest: newOneway_EchoYARPCRequest,
						},
					),
				},
			},
		},
	)
}

type _OnewayYARPCCaller struct {
	client protobuf.Client
}

func (c *_OnewayYARPCCaller) Echo(ctx context.Context, request *Token, options ...yarpc.CallOption) (yarpc.Ack, error) {
	return c.client.CallOneway(ctx, "Echo", request, options...)
}

type _OnewayYARPCHandler struct {
	server OnewayYARPCServer
}

func (h *_OnewayYARPCHandler) Echo(ctx context.Context, requestMessage proto.Message) error {
	var request *Token
	var ok bool
	if requestMessage != nil {
		request, ok = requestMessage.(*Token)
		if !ok {
			return protobuf.CastError(emptyOneway_EchoYARPCRequest, requestMessage)
		}
	}
	return h.server.Echo(ctx, request)
}

func newOneway_EchoYARPCRequest() proto.Message {
	return &Token{}
}

func newOneway_EchoYARPCResponse() proto.Message {
	return &yarpcproto.Oneway{}
}

var (
	emptyOneway_EchoYARPCRequest  = &Token{}
	emptyOneway_EchoYARPCResponse = &yarpcproto.Oneway{}
)

func init() {
	yarpc.RegisterClientBuilder(
		func(clientConfig transport.ClientConfig, structField reflect.StructField) EchoYARPCClient {
			return NewEchoYARPCClient(clientConfig, protobuf.ClientBuilderOptions(clientConfig, structField)...)
		},
	)
	yarpc.RegisterClientBuilder(
		func(clientConfig transport.ClientConfig, structField reflect.StructField) OnewayYARPCClient {
			return NewOnewayYARPCClient(clientConfig, protobuf.ClientBuilderOptions(clientConfig, structField)...)
		},
	)
}
