// Code generated by protoc-gen-yarpc-go
// source: src/stream/stream.proto
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

package streampb

import (
	context "context"
	fmt "fmt"
	fx "go.uber.org/fx"
	yarpc "go.uber.org/yarpc/v2"
	yarpcprotobuf "go.uber.org/yarpc/v2/yarpcprotobuf"
	reflection "go.uber.org/yarpc/v2/yarpcprotobuf/reflection"
)

// HelloYARPCClient is the Hello service's client interface.
type HelloYARPCClient interface {
	In(
		context.Context,
		*HelloRequest,
		...yarpc.CallOption,
	) (HelloInYARPCStreamClient, error)
	Out(
		context.Context,
		...yarpc.CallOption,
	) (HelloOutYARPCStreamClient, error)
	Bidirectional(
		context.Context,
		...yarpc.CallOption,
	) (HelloBidirectionalYARPCStreamClient, error)
}

// NewHelloYARPCClient builds a new YARPC client for the Hello service.
func NewHelloYARPCClient(c yarpc.Client, opts ...yarpcprotobuf.ClientOption) HelloYARPCClient {
	return &_HelloYARPCClient{stream: yarpcprotobuf.NewStreamClient(c, "stream.Hello", opts...)}
}

type _HelloYARPCClient struct {
	stream yarpcprotobuf.StreamClient
}

var _ HelloYARPCClient = (*_HelloYARPCClient)(nil)

func (c *_HelloYARPCClient) In(ctx context.Context, req *HelloRequest, opts ...yarpc.CallOption) (HelloInYARPCStreamClient, error) {
	s, err := c.stream.CallStream(ctx, "In", opts...)
	if err != nil {
		return nil, err
	}
	if err := s.Send(req); err != nil {
		return nil, err
	}
	return &_HelloInYARPCStreamClient{stream: s}, nil
}

func (c *_HelloYARPCClient) Out(ctx context.Context, opts ...yarpc.CallOption) (HelloOutYARPCStreamClient, error) {
	s, err := c.stream.CallStream(ctx, "Out", opts...)
	if err != nil {
		return nil, err
	}
	return &_HelloOutYARPCStreamClient{stream: s}, nil
}

func (c *_HelloYARPCClient) Bidirectional(ctx context.Context, opts ...yarpc.CallOption) (HelloBidirectionalYARPCStreamClient, error) {
	s, err := c.stream.CallStream(ctx, "Bidirectional", opts...)
	if err != nil {
		return nil, err
	}
	return &_HelloBidirectionalYARPCStreamClient{stream: s}, nil
}

// HelloInYARPCStreamClient is a streaming interface used in the HelloYARPCClient interface.
type HelloInYARPCStreamClient interface {
	Context() context.Context
	Recv(...yarpc.StreamOption) (*HelloResponse, error)
	CloseSend(...yarpc.StreamOption) error
}

// HelloOutYARPCStreamClient is a streaming interface used in the HelloYARPCClient interface.
type HelloOutYARPCStreamClient interface {
	Context() context.Context
	Send(*HelloRequest, ...yarpc.StreamOption) error
	CloseAndRecv(...yarpc.StreamOption) (*HelloResponse, error)
}

// HelloBidirectionalYARPCStreamClient is a streaming interface used in the HelloYARPCClient interface.
type HelloBidirectionalYARPCStreamClient interface {
	Context() context.Context
	Send(*HelloRequest, ...yarpc.StreamOption) error
	Recv(...yarpc.StreamOption) (*HelloResponse, error)
	CloseSend(...yarpc.StreamOption) error
}

type _HelloInYARPCStreamClient struct {
	stream *yarpcprotobuf.ClientStream
}

var _ HelloInYARPCStreamClient = (*_HelloInYARPCStreamClient)(nil)

func (c *_HelloInYARPCStreamClient) Context() context.Context {
	return c.stream.Context()
}

func (c *_HelloInYARPCStreamClient) Recv(opts ...yarpc.StreamOption) (*HelloResponse, error) {
	msg, err := c.stream.Receive(new(HelloResponse), opts...)
	if err != nil {
		return nil, err
	}
	res, ok := msg.(*HelloResponse)
	if !ok {
		return nil, yarpcprotobuf.CastError(new(HelloResponse), msg)
	}
	return res, nil
}

func (c *_HelloInYARPCStreamClient) CloseSend(opts ...yarpc.StreamOption) error {
	return c.stream.Close(opts...)
}

type _HelloOutYARPCStreamClient struct {
	stream *yarpcprotobuf.ClientStream
}

var _ HelloOutYARPCStreamClient = (*_HelloOutYARPCStreamClient)(nil)

func (c *_HelloOutYARPCStreamClient) Context() context.Context {
	return c.stream.Context()
}

func (c *_HelloOutYARPCStreamClient) Send(req *HelloRequest, opts ...yarpc.StreamOption) error {
	return c.stream.Send(req, opts...)
}

func (c *_HelloOutYARPCStreamClient) CloseAndRecv(opts ...yarpc.StreamOption) (*HelloResponse, error) {
	if err := c.stream.Close(opts...); err != nil {
		return nil, err
	}
	msg, err := c.stream.Receive(new(HelloResponse), opts...)
	if err != nil {
		return nil, err
	}
	res, ok := msg.(*HelloResponse)
	if !ok {
		return nil, yarpcprotobuf.CastError(new(HelloResponse), msg)
	}
	return res, err
}

type _HelloBidirectionalYARPCStreamClient struct {
	stream *yarpcprotobuf.ClientStream
}

var _ HelloBidirectionalYARPCStreamClient = (*_HelloBidirectionalYARPCStreamClient)(nil)

func (c *_HelloBidirectionalYARPCStreamClient) Context() context.Context {
	return c.stream.Context()
}

func (c *_HelloBidirectionalYARPCStreamClient) Send(req *HelloRequest, opts ...yarpc.StreamOption) error {
	return c.stream.Send(req, opts...)
}

func (c *_HelloBidirectionalYARPCStreamClient) Recv(opts ...yarpc.StreamOption) (*HelloResponse, error) {
	msg, err := c.stream.Receive(new(HelloResponse), opts...)
	if err != nil {
		return nil, err
	}
	res, ok := msg.(*HelloResponse)
	if !ok {
		return nil, yarpcprotobuf.CastError(new(HelloResponse), msg)
	}
	return res, nil
}

func (c *_HelloBidirectionalYARPCStreamClient) CloseSend(opts ...yarpc.StreamOption) error {
	return c.stream.Close(opts...)
}

// HelloYARPCServer is the Hello service's server interface.
type HelloYARPCServer interface {
	In(
		*HelloRequest,
		HelloInYARPCStreamServer,
	) error
	Out(
		HelloOutYARPCStreamServer,
	) (*HelloResponse, error)
	Bidirectional(
		HelloBidirectionalYARPCStreamServer,
	) error
}

// BuildHelloYARPCProcedures constructs the YARPC procedures for the Hello service.
func BuildHelloYARPCProcedures(s HelloYARPCServer) []yarpc.TransportProcedure {
	h := &_HelloYARPCServer{server: s}
	return yarpcprotobuf.Procedures(
		yarpcprotobuf.ProceduresParams{
			Service: "stream.Hello",
			Unary:   []yarpcprotobuf.UnaryProceduresParams{},
			Stream: []yarpcprotobuf.StreamProceduresParams{
				{
					Method: "In",
					Handler: yarpcprotobuf.NewStreamHandler(
						yarpcprotobuf.StreamHandlerParams{
							Handle: h.In,
						},
					),
				},
				{
					Method: "Out",
					Handler: yarpcprotobuf.NewStreamHandler(
						yarpcprotobuf.StreamHandlerParams{
							Handle: h.Out,
						},
					),
				},
				{
					Method: "Bidirectional",
					Handler: yarpcprotobuf.NewStreamHandler(
						yarpcprotobuf.StreamHandlerParams{
							Handle: h.Bidirectional,
						},
					),
				},
			},
		},
	)
}

type _HelloYARPCServer struct {
	server HelloYARPCServer
}

func (h *_HelloYARPCServer) In(s *yarpcprotobuf.ServerStream) error {
	recv, err := s.Receive(new(HelloRequest))
	if err != nil {
		return err
	}
	req, _ := recv.(*HelloRequest)
	if req == nil {
		return yarpcprotobuf.CastError(new(HelloRequest), recv)
	}
	return h.server.In(req, &_HelloInYARPCStreamServer{stream: s})
}

func (h *_HelloYARPCServer) Out(s *yarpcprotobuf.ServerStream) error {
	res, err := h.server.Out(&_HelloOutYARPCStreamServer{stream: s})
	if err != nil {
		return err
	}
	return s.Send(res)
}

func (h *_HelloYARPCServer) Bidirectional(s *yarpcprotobuf.ServerStream) error {
	return h.server.Bidirectional(&_HelloBidirectionalYARPCStreamServer{stream: s})
}

// HelloInYARPCStreamServer is a streaming interface used in the HelloYARPCServer interface.
type HelloInYARPCStreamServer interface {
	Context() context.Context
	Send(*HelloResponse, ...yarpc.StreamOption) error
}

// HelloOutYARPCStreamServer is a streaming interface used in the HelloYARPCServer interface.
type HelloOutYARPCStreamServer interface {
	Context() context.Context
	Recv(...yarpc.StreamOption) (*HelloRequest, error)
}

// HelloBidirectionalYARPCStreamServer is a streaming interface used in the HelloYARPCServer interface.
type HelloBidirectionalYARPCStreamServer interface {
	Context() context.Context
	Recv(...yarpc.StreamOption) (*HelloRequest, error)
	Send(*HelloResponse, ...yarpc.StreamOption) error
}

type _HelloInYARPCStreamServer struct {
	stream *yarpcprotobuf.ServerStream
}

var _ HelloInYARPCStreamServer = (*_HelloInYARPCStreamServer)(nil)

func (s *_HelloInYARPCStreamServer) Context() context.Context {
	return s.stream.Context()
}

func (s *_HelloInYARPCStreamServer) Send(res *HelloResponse, opts ...yarpc.StreamOption) error {
	return s.stream.Send(res, opts...)
}

type _HelloOutYARPCStreamServer struct {
	stream *yarpcprotobuf.ServerStream
}

var _ HelloOutYARPCStreamServer = (*_HelloOutYARPCStreamServer)(nil)

func (s *_HelloOutYARPCStreamServer) Context() context.Context {
	return s.stream.Context()
}

func (s *_HelloOutYARPCStreamServer) Recv(opts ...yarpc.StreamOption) (*HelloRequest, error) {
	msg, err := s.stream.Receive(new(HelloRequest), opts...)
	if err != nil {
		return nil, err
	}
	req, ok := msg.(*HelloRequest)
	if !ok {
		return nil, yarpcprotobuf.CastError(new(HelloRequest), msg)
	}
	return req, nil
}

type _HelloBidirectionalYARPCStreamServer struct {
	stream *yarpcprotobuf.ServerStream
}

var _ HelloBidirectionalYARPCStreamServer = (*_HelloBidirectionalYARPCStreamServer)(nil)

func (s *_HelloBidirectionalYARPCStreamServer) Context() context.Context {
	return s.stream.Context()
}

func (s *_HelloBidirectionalYARPCStreamServer) Recv(opts ...yarpc.StreamOption) (*HelloRequest, error) {
	msg, err := s.stream.Receive(new(HelloRequest), opts...)
	if err != nil {
		return nil, err
	}
	req, ok := msg.(*HelloRequest)
	if !ok {
		return nil, yarpcprotobuf.CastError(new(HelloRequest), msg)
	}
	return req, nil
}

func (s *_HelloBidirectionalYARPCStreamServer) Send(res *HelloResponse, opts ...yarpc.StreamOption) error {
	return s.stream.Send(res, opts...)
}

// FxHelloYARPCClientParams defines the parameters
// required to provide a HelloYARPCClient into an
// Fx application.
type FxHelloYARPCClientParams struct {
	fx.In

	ClientProvider yarpc.ClientProvider
}

// FxHelloYARPCClientResult provides a HelloYARPCClient
// into an Fx application.
type FxHelloYARPCClientResult struct {
	fx.Out

	Client HelloYARPCClient
}

// NewFxHelloYARPCClient provides a HelloYARPCClient
// into an Fx application, using the given
// name for routing.
//
//  fx.Provide(
//    streampb.NewFxHelloYARPCClient("service-name"),
//    ...
//  )
func NewFxHelloYARPCClient(name string, opts ...yarpcprotobuf.ClientOption) interface{} {
	return func(p FxHelloYARPCClientParams) (FxHelloYARPCClientResult, error) {
		client, ok := p.ClientProvider.Client(name)
		if !ok {
			return FxHelloYARPCClientResult{},
				fmt.Errorf("generated code could not retrieve client for %q", name)
		}
		return FxHelloYARPCClientResult{
			Client: NewHelloYARPCClient(client, opts...),
		}, nil
	}
}

// FxHelloYARPCServerParams defines the paramaters
// required to provide the HelloYARPCServer procedures
// into an Fx application.
type FxHelloYARPCServerParams struct {
	fx.In

	Server HelloYARPCServer
}

// FxHelloYARPCServerResult provides the HelloYARPCServer
// procedures into an Fx application.
type FxHelloYARPCServerResult struct {
	fx.Out

	Procedures     []yarpc.TransportProcedure `group:"yarpcfx"`
	ReflectionMeta reflection.ServerMeta      `group:"yarpcfx"`
}

// NewFxHelloYARPCServer provides the HelloYARPCServer
// procedures to an Fx application. It expects
// a HelloYARPCServer to be present in the container.
//
//  fx.Provide(
//    streampb.NewFxHelloYARPCServer(),
//    ...
//  )
func NewFxHelloYARPCServer() interface{} {
	return func(p FxHelloYARPCServerParams) FxHelloYARPCServerResult {
		return FxHelloYARPCServerResult{
			Procedures: BuildHelloYARPCProcedures(p.Server),
			ReflectionMeta: reflection.ServerMeta{
				ServiceName:     "stream.Hello",
				FileDescriptors: yarpcFileDescriptorClosureda84cc27cf0e7d6a,
			},
		}
	}
}

var yarpcFileDescriptorClosureda84cc27cf0e7d6a = [][]byte{
	// src/stream/stream.proto
	[]byte{
		0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xe2, 0x12, 0x2f, 0x2e, 0x4a, 0xd6,
		0x2f, 0x2e, 0x29, 0x4a, 0x4d, 0xcc, 0x85, 0x52, 0x7a, 0x05, 0x45, 0xf9, 0x25, 0xf9, 0x42, 0x6c,
		0x10, 0x9e, 0x92, 0x16, 0x17, 0x8f, 0x47, 0x6a, 0x4e, 0x4e, 0x7e, 0x50, 0x6a, 0x61, 0x69, 0x6a,
		0x71, 0x89, 0x90, 0x14, 0x17, 0x47, 0x7a, 0x51, 0x6a, 0x6a, 0x49, 0x66, 0x5e, 0xba, 0x04, 0xa3,
		0x02, 0xa3, 0x06, 0x67, 0x10, 0x9c, 0xaf, 0xa4, 0xcd, 0xc5, 0x0b, 0x55, 0x5b, 0x5c, 0x90, 0x9f,
		0x57, 0x9c, 0x0a, 0x52, 0x5c, 0x04, 0x65, 0xc3, 0x14, 0xc3, 0xf8, 0x46, 0xbb, 0x18, 0xb9, 0x58,
		0xc1, 0xaa, 0x85, 0x4c, 0xb9, 0x98, 0x3c, 0xf3, 0x84, 0x44, 0xf4, 0xa0, 0xf6, 0x23, 0x5b, 0x27,
		0x25, 0x8a, 0x26, 0x0a, 0xd1, 0xac, 0xc4, 0x60, 0xc0, 0x28, 0x64, 0xc6, 0xc5, 0xec, 0x5f, 0x5a,
		0x42, 0xa2, 0x3e, 0x0d, 0x46, 0x21, 0x27, 0x2e, 0x5e, 0xa7, 0xcc, 0x94, 0xcc, 0xa2, 0xd4, 0xe4,
		0x92, 0xcc, 0xfc, 0xbc, 0xc4, 0x1c, 0x92, 0x4d, 0x30, 0x60, 0x74, 0xe2, 0x8a, 0xe2, 0x80, 0xc8,
		0x16, 0x24, 0x25, 0xb1, 0x81, 0x03, 0xcc, 0x18, 0x10, 0x00, 0x00, 0xff, 0xff, 0xef, 0x5b, 0x59,
		0xb3, 0x4b, 0x01, 0x00, 0x00,
	},
}
