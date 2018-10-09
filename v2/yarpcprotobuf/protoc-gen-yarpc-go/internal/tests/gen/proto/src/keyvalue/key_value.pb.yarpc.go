// Code generated by protoc-gen-yarpc-go
// source: src/keyvalue/key_value.proto
// DO NOT EDIT!

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

package keyvaluepb

import (
	context "context"
	proto "github.com/gogo/protobuf/proto"
	fx "go.uber.org/fx"
	yarpc "go.uber.org/yarpc/v2"
	yarpcprotobuf "go.uber.org/yarpc/v2/yarpcprotobuf"
	common "go.uber.org/yarpc/v2/yarpcprotobuf/protoc-gen-yarpc-go/internal/tests/gen/proto/src/common"
)

// StoreYARPCClient is the Store service's client interface.
type StoreYARPCClient interface {
	Get(
		context.Context,
		*common.GetRequest,
		...yarpc.CallOption,
	) (*common.GetResponse, error)
	Set(
		context.Context,
		*common.SetRequest,
		...yarpc.CallOption,
	) (*common.SetResponse, error)
}

// NewStoreYARPCClient builds a new YARPC client for the Store service.
func NewStoreYARPCClient(c yarpc.Client, opts ...yarpcprotobuf.ClientOption) StoreYARPCClient {
	return &_StoreYARPCClient{stream: yarpcprotobuf.NewStreamClient(c, "keyvalue.Store", opts...)}
}

type _StoreYARPCClient struct {
	stream yarpcprotobuf.StreamClient
}

var _ StoreYARPCClient = (*_StoreYARPCClient)(nil)

func (c *_StoreYARPCClient) Get(ctx context.Context, req *common.GetRequest, opts ...yarpc.CallOption) (*common.GetResponse, error) {
	msg, err := c.stream.Call(ctx, "Get", req, new(common.GetResponse), opts...)
	if err != nil {
		return nil, err
	}
	res, ok := msg.(*common.GetResponse)
	if !ok {
		return nil, yarpcprotobuf.CastError(new(common.GetResponse), res)
	}
	return res, nil
}

func (c *_StoreYARPCClient) Set(ctx context.Context, req *common.SetRequest, opts ...yarpc.CallOption) (*common.SetResponse, error) {
	msg, err := c.stream.Call(ctx, "Set", req, new(common.SetResponse), opts...)
	if err != nil {
		return nil, err
	}
	res, ok := msg.(*common.SetResponse)
	if !ok {
		return nil, yarpcprotobuf.CastError(new(common.SetResponse), res)
	}
	return res, nil
}

// StoreYARPCServer is the Store service's server interface.
type StoreYARPCServer interface {
	Get(
		context.Context,
		*common.GetRequest,
	) (*common.GetResponse, error)
	Set(
		context.Context,
		*common.SetRequest,
	) (*common.SetResponse, error)
}

// BuildStoreYARPCProcedures constructs the YARPC procedures for the Store service.
func BuildStoreYARPCProcedures(s StoreYARPCServer) []yarpc.Procedure {
	h := &_StoreYARPCServer{server: s}
	return yarpcprotobuf.Procedures(
		yarpcprotobuf.ProceduresParams{
			Service: "keyvalue.Store",
			Unary: []yarpcprotobuf.UnaryProceduresParams{
				{
					Method: "Get",
					Handler: yarpcprotobuf.NewUnaryHandler(
						yarpcprotobuf.UnaryHandlerParams{
							Handle:      h.Get,
							RequestType: new(common.GetRequest),
						},
					),
				},
				{
					Method: "Set",
					Handler: yarpcprotobuf.NewUnaryHandler(
						yarpcprotobuf.UnaryHandlerParams{
							Handle:      h.Set,
							RequestType: new(common.SetRequest),
						},
					),
				},
			},
			Stream: []yarpcprotobuf.StreamProceduresParams{},
		},
	)
}

type _StoreYARPCServer struct {
	server StoreYARPCServer
}

func (h *_StoreYARPCServer) Get(ctx context.Context, m proto.Message) (proto.Message, error) {
	req, _ := m.(*common.GetRequest)
	if req == nil {
		return nil, yarpcprotobuf.CastError(new(common.GetRequest), m)
	}
	return h.server.Get(ctx, req)
}

func (h *_StoreYARPCServer) Set(ctx context.Context, m proto.Message) (proto.Message, error) {
	req, _ := m.(*common.SetRequest)
	if req == nil {
		return nil, yarpcprotobuf.CastError(new(common.SetRequest), m)
	}
	return h.server.Set(ctx, req)
}

// FxStoreYARPCClientParams defines the parameters
// required to provide a StoreYARPCClient into an
// Fx application.
type FxStoreYARPCClientParams struct {
	fx.In

	Client yarpc.Client
}

// FxStoreYARPCClientResult provides a StoreYARPCClient
// into an Fx application.
type FxStoreYARPCClientResult struct {
	fx.Out

	Client StoreYARPCClient
}

// NewFxStoreYARPCClient provides a StoreYARPCClient
// into an Fx application, using the given
// name for routing.
//
//  fx.Provide(
//    keyvaluepb.NewFxStoreYARPCClient("service-name"),
//    ...
//  )
// TODO(mensch): How will this work in v2?
func NewFxStoreYARPCClient(_ string, opts ...yarpcprotobuf.ClientOption) interface{} {
	return func(p FxStoreYARPCClientParams) FxStoreYARPCClientResult {
		return FxStoreYARPCClientResult{
			Client: NewStoreYARPCClient(p.Client, opts...),
		}
	}
}

// FxStoreYARPCServerParams defines the paramaters
// required to provide the StoreYARPCServer procedures
// into an Fx application.
type FxStoreYARPCServerParams struct {
	fx.In

	Server StoreYARPCServer
}

// FxStoreYARPCServerResult provides the StoreYARPCServer
// procedures into an Fx application.
type FxStoreYARPCServerResult struct {
	fx.Out

	Procedures []yarpc.Procedure `group:"yarpcfx"`
}

// NewFxStoreYARPCServer provides the StoreYARPCServer
// procedures to an Fx application. It expects
// a StoreYARPCServer to be present in the container.
//
//  fx.Provide(
//    keyvaluepb.NewFxStoreYARPCServer(),
//    ...
//  )
func NewFxStoreYARPCServer() interface{} {
	return func(p FxStoreYARPCServerParams) FxStoreYARPCServerResult {
		return FxStoreYARPCServerResult{
			Procedures: BuildStoreYARPCProcedures(p.Server),
		}
	}
}
