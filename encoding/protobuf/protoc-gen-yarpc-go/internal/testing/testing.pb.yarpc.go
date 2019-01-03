// Code generated by protoc-gen-yarpc-go
// source: encoding/protobuf/protoc-gen-yarpc-go/internal/testing/testing.proto
// DO NOT EDIT!

// Copyright (c) 2019 Uber Technologies, Inc.
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

package testing

import (
	"context"
	"io/ioutil"
	"reflect"

	"github.com/gogo/protobuf/proto"
	"go.uber.org/fx"
	"go.uber.org/yarpc"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/encoding/protobuf"
	"go.uber.org/yarpc/encoding/protobuf/reflection"
	"go.uber.org/yarpc/yarpcproto"
)

var _ = ioutil.NopCloser

// KeyValueYARPCClient is the YARPC client-side interface for the KeyValue service.
type KeyValueYARPCClient interface {
	GetValue(context.Context, *GetValueRequest, ...yarpc.CallOption) (*GetValueResponse, error)
	SetValue(context.Context, *SetValueRequest, ...yarpc.CallOption) (*SetValueResponse, error)
}

// NewKeyValueYARPCClient builds a new YARPC client for the KeyValue service.
func NewKeyValueYARPCClient(clientConfig transport.ClientConfig, options ...protobuf.ClientOption) KeyValueYARPCClient {
	return &_KeyValueYARPCCaller{protobuf.NewStreamClient(
		protobuf.ClientParams{
			ServiceName:  "uber.yarpc.encoding.protobuf.protocgenyarpcgo.internal.testing.KeyValue",
			ClientConfig: clientConfig,
			Options:      options,
		},
	)}
}

// KeyValueYARPCServer is the YARPC server-side interface for the KeyValue service.
type KeyValueYARPCServer interface {
	GetValue(context.Context, *GetValueRequest) (*GetValueResponse, error)
	SetValue(context.Context, *SetValueRequest) (*SetValueResponse, error)
}

// BuildKeyValueYARPCProcedures prepares an implementation of the KeyValue service for YARPC registration.
func BuildKeyValueYARPCProcedures(server KeyValueYARPCServer) []transport.Procedure {
	handler := &_KeyValueYARPCHandler{server}
	return protobuf.BuildProcedures(
		protobuf.BuildProceduresParams{
			ServiceName: "uber.yarpc.encoding.protobuf.protocgenyarpcgo.internal.testing.KeyValue",
			UnaryHandlerParams: []protobuf.BuildProceduresUnaryHandlerParams{
				{
					MethodName: "GetValue",
					Handler: protobuf.NewUnaryHandler(
						protobuf.UnaryHandlerParams{
							Handle:     handler.GetValue,
							NewRequest: newKeyValueServiceGetValueYARPCRequest,
						},
					),
				},
				{
					MethodName: "SetValue",
					Handler: protobuf.NewUnaryHandler(
						protobuf.UnaryHandlerParams{
							Handle:     handler.SetValue,
							NewRequest: newKeyValueServiceSetValueYARPCRequest,
						},
					),
				},
			},
			OnewayHandlerParams: []protobuf.BuildProceduresOnewayHandlerParams{},
			StreamHandlerParams: []protobuf.BuildProceduresStreamHandlerParams{},
		},
	)
}

// FxKeyValueYARPCClientParams defines the input
// for NewFxKeyValueYARPCClient. It provides the
// paramaters to get a KeyValueYARPCClient in an
// Fx application.
type FxKeyValueYARPCClientParams struct {
	fx.In

	Provider yarpc.ClientConfig
}

// FxKeyValueYARPCClientResult defines the output
// of NewFxKeyValueYARPCClient. It provides a
// KeyValueYARPCClient to an Fx application.
type FxKeyValueYARPCClientResult struct {
	fx.Out

	Client KeyValueYARPCClient

	// We are using an fx.Out struct here instead of just returning a client
	// so that we can add more values or add named versions of the client in
	// the future without breaking any existing code.
}

// NewFxKeyValueYARPCClient provides a KeyValueYARPCClient
// to an Fx application using the given name for routing.
//
//  fx.Provide(
//    testing.NewFxKeyValueYARPCClient("service-name"),
//    ...
//  )
func NewFxKeyValueYARPCClient(name string, options ...protobuf.ClientOption) interface{} {
	return func(params FxKeyValueYARPCClientParams) FxKeyValueYARPCClientResult {
		return FxKeyValueYARPCClientResult{
			Client: NewKeyValueYARPCClient(params.Provider.ClientConfig(name), options...),
		}
	}
}

// FxKeyValueYARPCProceduresParams defines the input
// for NewFxKeyValueYARPCProcedures. It provides the
// paramaters to get KeyValueYARPCServer procedures in an
// Fx application.
type FxKeyValueYARPCProceduresParams struct {
	fx.In

	Server KeyValueYARPCServer
}

// FxKeyValueYARPCProceduresResult defines the output
// of NewFxKeyValueYARPCProcedures. It provides
// KeyValueYARPCServer procedures to an Fx application.
//
// The procedures are provided to the "yarpcfx" value group.
// Dig 1.2 or newer must be used for this feature to work.
type FxKeyValueYARPCProceduresResult struct {
	fx.Out

	Procedures     []transport.Procedure `group:"yarpcfx"`
	ReflectionMeta reflection.ServerMeta `group:"yarpcfx"`
}

// NewFxKeyValueYARPCProcedures provides KeyValueYARPCServer procedures to an Fx application.
// It expects a KeyValueYARPCServer to be present in the container.
//
//  fx.Provide(
//    testing.NewFxKeyValueYARPCProcedures(),
//    ...
//  )
func NewFxKeyValueYARPCProcedures() interface{} {
	return func(params FxKeyValueYARPCProceduresParams) FxKeyValueYARPCProceduresResult {
		return FxKeyValueYARPCProceduresResult{
			Procedures: BuildKeyValueYARPCProcedures(params.Server),
			ReflectionMeta: reflection.ServerMeta{
				ServiceName:     "uber.yarpc.encoding.protobuf.protocgenyarpcgo.internal.testing.KeyValue",
				FileDescriptors: yarpcFileDescriptorClosure301ba429865f230b,
			},
		}
	}
}

type _KeyValueYARPCCaller struct {
	streamClient protobuf.StreamClient
}

func (c *_KeyValueYARPCCaller) GetValue(ctx context.Context, request *GetValueRequest, options ...yarpc.CallOption) (*GetValueResponse, error) {
	responseMessage, err := c.streamClient.Call(ctx, "GetValue", request, newKeyValueServiceGetValueYARPCResponse, options...)
	if responseMessage == nil {
		return nil, err
	}
	response, ok := responseMessage.(*GetValueResponse)
	if !ok {
		return nil, protobuf.CastError(emptyKeyValueServiceGetValueYARPCResponse, responseMessage)
	}
	return response, err
}

func (c *_KeyValueYARPCCaller) SetValue(ctx context.Context, request *SetValueRequest, options ...yarpc.CallOption) (*SetValueResponse, error) {
	responseMessage, err := c.streamClient.Call(ctx, "SetValue", request, newKeyValueServiceSetValueYARPCResponse, options...)
	if responseMessage == nil {
		return nil, err
	}
	response, ok := responseMessage.(*SetValueResponse)
	if !ok {
		return nil, protobuf.CastError(emptyKeyValueServiceSetValueYARPCResponse, responseMessage)
	}
	return response, err
}

type _KeyValueYARPCHandler struct {
	server KeyValueYARPCServer
}

func (h *_KeyValueYARPCHandler) GetValue(ctx context.Context, requestMessage proto.Message) (proto.Message, error) {
	var request *GetValueRequest
	var ok bool
	if requestMessage != nil {
		request, ok = requestMessage.(*GetValueRequest)
		if !ok {
			return nil, protobuf.CastError(emptyKeyValueServiceGetValueYARPCRequest, requestMessage)
		}
	}
	response, err := h.server.GetValue(ctx, request)
	if response == nil {
		return nil, err
	}
	return response, err
}

func (h *_KeyValueYARPCHandler) SetValue(ctx context.Context, requestMessage proto.Message) (proto.Message, error) {
	var request *SetValueRequest
	var ok bool
	if requestMessage != nil {
		request, ok = requestMessage.(*SetValueRequest)
		if !ok {
			return nil, protobuf.CastError(emptyKeyValueServiceSetValueYARPCRequest, requestMessage)
		}
	}
	response, err := h.server.SetValue(ctx, request)
	if response == nil {
		return nil, err
	}
	return response, err
}

func newKeyValueServiceGetValueYARPCRequest() proto.Message {
	return &GetValueRequest{}
}

func newKeyValueServiceGetValueYARPCResponse() proto.Message {
	return &GetValueResponse{}
}

func newKeyValueServiceSetValueYARPCRequest() proto.Message {
	return &SetValueRequest{}
}

func newKeyValueServiceSetValueYARPCResponse() proto.Message {
	return &SetValueResponse{}
}

var (
	emptyKeyValueServiceGetValueYARPCRequest  = &GetValueRequest{}
	emptyKeyValueServiceGetValueYARPCResponse = &GetValueResponse{}
	emptyKeyValueServiceSetValueYARPCRequest  = &SetValueRequest{}
	emptyKeyValueServiceSetValueYARPCResponse = &SetValueResponse{}
)

// SinkYARPCClient is the YARPC client-side interface for the Sink service.
type SinkYARPCClient interface {
	Fire(context.Context, *FireRequest, ...yarpc.CallOption) (yarpc.Ack, error)
}

// NewSinkYARPCClient builds a new YARPC client for the Sink service.
func NewSinkYARPCClient(clientConfig transport.ClientConfig, options ...protobuf.ClientOption) SinkYARPCClient {
	return &_SinkYARPCCaller{protobuf.NewStreamClient(
		protobuf.ClientParams{
			ServiceName:  "uber.yarpc.encoding.protobuf.protocgenyarpcgo.internal.testing.Sink",
			ClientConfig: clientConfig,
			Options:      options,
		},
	)}
}

// SinkYARPCServer is the YARPC server-side interface for the Sink service.
type SinkYARPCServer interface {
	Fire(context.Context, *FireRequest) error
}

// BuildSinkYARPCProcedures prepares an implementation of the Sink service for YARPC registration.
func BuildSinkYARPCProcedures(server SinkYARPCServer) []transport.Procedure {
	handler := &_SinkYARPCHandler{server}
	return protobuf.BuildProcedures(
		protobuf.BuildProceduresParams{
			ServiceName:        "uber.yarpc.encoding.protobuf.protocgenyarpcgo.internal.testing.Sink",
			UnaryHandlerParams: []protobuf.BuildProceduresUnaryHandlerParams{},
			OnewayHandlerParams: []protobuf.BuildProceduresOnewayHandlerParams{
				{
					MethodName: "Fire",
					Handler: protobuf.NewOnewayHandler(
						protobuf.OnewayHandlerParams{
							Handle:     handler.Fire,
							NewRequest: newSinkServiceFireYARPCRequest,
						},
					),
				},
			},
			StreamHandlerParams: []protobuf.BuildProceduresStreamHandlerParams{},
		},
	)
}

// FxSinkYARPCClientParams defines the input
// for NewFxSinkYARPCClient. It provides the
// paramaters to get a SinkYARPCClient in an
// Fx application.
type FxSinkYARPCClientParams struct {
	fx.In

	Provider yarpc.ClientConfig
}

// FxSinkYARPCClientResult defines the output
// of NewFxSinkYARPCClient. It provides a
// SinkYARPCClient to an Fx application.
type FxSinkYARPCClientResult struct {
	fx.Out

	Client SinkYARPCClient

	// We are using an fx.Out struct here instead of just returning a client
	// so that we can add more values or add named versions of the client in
	// the future without breaking any existing code.
}

// NewFxSinkYARPCClient provides a SinkYARPCClient
// to an Fx application using the given name for routing.
//
//  fx.Provide(
//    testing.NewFxSinkYARPCClient("service-name"),
//    ...
//  )
func NewFxSinkYARPCClient(name string, options ...protobuf.ClientOption) interface{} {
	return func(params FxSinkYARPCClientParams) FxSinkYARPCClientResult {
		return FxSinkYARPCClientResult{
			Client: NewSinkYARPCClient(params.Provider.ClientConfig(name), options...),
		}
	}
}

// FxSinkYARPCProceduresParams defines the input
// for NewFxSinkYARPCProcedures. It provides the
// paramaters to get SinkYARPCServer procedures in an
// Fx application.
type FxSinkYARPCProceduresParams struct {
	fx.In

	Server SinkYARPCServer
}

// FxSinkYARPCProceduresResult defines the output
// of NewFxSinkYARPCProcedures. It provides
// SinkYARPCServer procedures to an Fx application.
//
// The procedures are provided to the "yarpcfx" value group.
// Dig 1.2 or newer must be used for this feature to work.
type FxSinkYARPCProceduresResult struct {
	fx.Out

	Procedures     []transport.Procedure `group:"yarpcfx"`
	ReflectionMeta reflection.ServerMeta `group:"yarpcfx"`
}

// NewFxSinkYARPCProcedures provides SinkYARPCServer procedures to an Fx application.
// It expects a SinkYARPCServer to be present in the container.
//
//  fx.Provide(
//    testing.NewFxSinkYARPCProcedures(),
//    ...
//  )
func NewFxSinkYARPCProcedures() interface{} {
	return func(params FxSinkYARPCProceduresParams) FxSinkYARPCProceduresResult {
		return FxSinkYARPCProceduresResult{
			Procedures: BuildSinkYARPCProcedures(params.Server),
			ReflectionMeta: reflection.ServerMeta{
				ServiceName:     "uber.yarpc.encoding.protobuf.protocgenyarpcgo.internal.testing.Sink",
				FileDescriptors: yarpcFileDescriptorClosure301ba429865f230b,
			},
		}
	}
}

type _SinkYARPCCaller struct {
	streamClient protobuf.StreamClient
}

func (c *_SinkYARPCCaller) Fire(ctx context.Context, request *FireRequest, options ...yarpc.CallOption) (yarpc.Ack, error) {
	return c.streamClient.CallOneway(ctx, "Fire", request, options...)
}

type _SinkYARPCHandler struct {
	server SinkYARPCServer
}

func (h *_SinkYARPCHandler) Fire(ctx context.Context, requestMessage proto.Message) error {
	var request *FireRequest
	var ok bool
	if requestMessage != nil {
		request, ok = requestMessage.(*FireRequest)
		if !ok {
			return protobuf.CastError(emptySinkServiceFireYARPCRequest, requestMessage)
		}
	}
	return h.server.Fire(ctx, request)
}

func newSinkServiceFireYARPCRequest() proto.Message {
	return &FireRequest{}
}

func newSinkServiceFireYARPCResponse() proto.Message {
	return &yarpcproto.Oneway{}
}

var (
	emptySinkServiceFireYARPCRequest  = &FireRequest{}
	emptySinkServiceFireYARPCResponse = &yarpcproto.Oneway{}
)

// AllYARPCClient is the YARPC client-side interface for the All service.
type AllYARPCClient interface {
	GetValue(context.Context, *GetValueRequest, ...yarpc.CallOption) (*GetValueResponse, error)
	SetValue(context.Context, *SetValueRequest, ...yarpc.CallOption) (*SetValueResponse, error)
	Fire(context.Context, *FireRequest, ...yarpc.CallOption) (yarpc.Ack, error)
	HelloOne(context.Context, ...yarpc.CallOption) (AllServiceHelloOneYARPCClient, error)
	HelloTwo(context.Context, *HelloRequest, ...yarpc.CallOption) (AllServiceHelloTwoYARPCClient, error)
	HelloThree(context.Context, ...yarpc.CallOption) (AllServiceHelloThreeYARPCClient, error)
}

// AllServiceHelloOneYARPCClient sends HelloRequests and receives the single HelloResponse when sending is done.
type AllServiceHelloOneYARPCClient interface {
	Context() context.Context
	Send(*HelloRequest, ...yarpc.StreamOption) error
	CloseAndRecv(...yarpc.StreamOption) (*HelloResponse, error)
}

// AllServiceHelloTwoYARPCClient receives HelloResponses, returning io.EOF when the stream is complete.
type AllServiceHelloTwoYARPCClient interface {
	Context() context.Context
	Recv(...yarpc.StreamOption) (*HelloResponse, error)
	CloseSend(...yarpc.StreamOption) error
}

// AllServiceHelloThreeYARPCClient sends HelloRequests and receives HelloResponses, returning io.EOF when the stream is complete.
type AllServiceHelloThreeYARPCClient interface {
	Context() context.Context
	Send(*HelloRequest, ...yarpc.StreamOption) error
	Recv(...yarpc.StreamOption) (*HelloResponse, error)
	CloseSend(...yarpc.StreamOption) error
}

// NewAllYARPCClient builds a new YARPC client for the All service.
func NewAllYARPCClient(clientConfig transport.ClientConfig, options ...protobuf.ClientOption) AllYARPCClient {
	return &_AllYARPCCaller{protobuf.NewStreamClient(
		protobuf.ClientParams{
			ServiceName:  "uber.yarpc.encoding.protobuf.protocgenyarpcgo.internal.testing.All",
			ClientConfig: clientConfig,
			Options:      options,
		},
	)}
}

// AllYARPCServer is the YARPC server-side interface for the All service.
type AllYARPCServer interface {
	GetValue(context.Context, *GetValueRequest) (*GetValueResponse, error)
	SetValue(context.Context, *SetValueRequest) (*SetValueResponse, error)
	Fire(context.Context, *FireRequest) error
	HelloOne(AllServiceHelloOneYARPCServer) (*HelloResponse, error)
	HelloTwo(*HelloRequest, AllServiceHelloTwoYARPCServer) error
	HelloThree(AllServiceHelloThreeYARPCServer) error
}

// AllServiceHelloOneYARPCServer receives HelloRequests.
type AllServiceHelloOneYARPCServer interface {
	Context() context.Context
	Recv(...yarpc.StreamOption) (*HelloRequest, error)
}

// AllServiceHelloTwoYARPCServer sends HelloResponses.
type AllServiceHelloTwoYARPCServer interface {
	Context() context.Context
	Send(*HelloResponse, ...yarpc.StreamOption) error
}

// AllServiceHelloThreeYARPCServer receives HelloRequests and sends HelloResponse.
type AllServiceHelloThreeYARPCServer interface {
	Context() context.Context
	Recv(...yarpc.StreamOption) (*HelloRequest, error)
	Send(*HelloResponse, ...yarpc.StreamOption) error
}

// BuildAllYARPCProcedures prepares an implementation of the All service for YARPC registration.
func BuildAllYARPCProcedures(server AllYARPCServer) []transport.Procedure {
	handler := &_AllYARPCHandler{server}
	return protobuf.BuildProcedures(
		protobuf.BuildProceduresParams{
			ServiceName: "uber.yarpc.encoding.protobuf.protocgenyarpcgo.internal.testing.All",
			UnaryHandlerParams: []protobuf.BuildProceduresUnaryHandlerParams{
				{
					MethodName: "GetValue",
					Handler: protobuf.NewUnaryHandler(
						protobuf.UnaryHandlerParams{
							Handle:     handler.GetValue,
							NewRequest: newAllServiceGetValueYARPCRequest,
						},
					),
				},
				{
					MethodName: "SetValue",
					Handler: protobuf.NewUnaryHandler(
						protobuf.UnaryHandlerParams{
							Handle:     handler.SetValue,
							NewRequest: newAllServiceSetValueYARPCRequest,
						},
					),
				},
			},
			OnewayHandlerParams: []protobuf.BuildProceduresOnewayHandlerParams{
				{
					MethodName: "Fire",
					Handler: protobuf.NewOnewayHandler(
						protobuf.OnewayHandlerParams{
							Handle:     handler.Fire,
							NewRequest: newAllServiceFireYARPCRequest,
						},
					),
				},
			},
			StreamHandlerParams: []protobuf.BuildProceduresStreamHandlerParams{
				{
					MethodName: "HelloThree",
					Handler: protobuf.NewStreamHandler(
						protobuf.StreamHandlerParams{
							Handle: handler.HelloThree,
						},
					),
				},

				{
					MethodName: "HelloTwo",
					Handler: protobuf.NewStreamHandler(
						protobuf.StreamHandlerParams{
							Handle: handler.HelloTwo,
						},
					),
				},

				{
					MethodName: "HelloOne",
					Handler: protobuf.NewStreamHandler(
						protobuf.StreamHandlerParams{
							Handle: handler.HelloOne,
						},
					),
				},
			},
		},
	)
}

// FxAllYARPCClientParams defines the input
// for NewFxAllYARPCClient. It provides the
// paramaters to get a AllYARPCClient in an
// Fx application.
type FxAllYARPCClientParams struct {
	fx.In

	Provider yarpc.ClientConfig
}

// FxAllYARPCClientResult defines the output
// of NewFxAllYARPCClient. It provides a
// AllYARPCClient to an Fx application.
type FxAllYARPCClientResult struct {
	fx.Out

	Client AllYARPCClient

	// We are using an fx.Out struct here instead of just returning a client
	// so that we can add more values or add named versions of the client in
	// the future without breaking any existing code.
}

// NewFxAllYARPCClient provides a AllYARPCClient
// to an Fx application using the given name for routing.
//
//  fx.Provide(
//    testing.NewFxAllYARPCClient("service-name"),
//    ...
//  )
func NewFxAllYARPCClient(name string, options ...protobuf.ClientOption) interface{} {
	return func(params FxAllYARPCClientParams) FxAllYARPCClientResult {
		return FxAllYARPCClientResult{
			Client: NewAllYARPCClient(params.Provider.ClientConfig(name), options...),
		}
	}
}

// FxAllYARPCProceduresParams defines the input
// for NewFxAllYARPCProcedures. It provides the
// paramaters to get AllYARPCServer procedures in an
// Fx application.
type FxAllYARPCProceduresParams struct {
	fx.In

	Server AllYARPCServer
}

// FxAllYARPCProceduresResult defines the output
// of NewFxAllYARPCProcedures. It provides
// AllYARPCServer procedures to an Fx application.
//
// The procedures are provided to the "yarpcfx" value group.
// Dig 1.2 or newer must be used for this feature to work.
type FxAllYARPCProceduresResult struct {
	fx.Out

	Procedures     []transport.Procedure `group:"yarpcfx"`
	ReflectionMeta reflection.ServerMeta `group:"yarpcfx"`
}

// NewFxAllYARPCProcedures provides AllYARPCServer procedures to an Fx application.
// It expects a AllYARPCServer to be present in the container.
//
//  fx.Provide(
//    testing.NewFxAllYARPCProcedures(),
//    ...
//  )
func NewFxAllYARPCProcedures() interface{} {
	return func(params FxAllYARPCProceduresParams) FxAllYARPCProceduresResult {
		return FxAllYARPCProceduresResult{
			Procedures: BuildAllYARPCProcedures(params.Server),
			ReflectionMeta: reflection.ServerMeta{
				ServiceName:     "uber.yarpc.encoding.protobuf.protocgenyarpcgo.internal.testing.All",
				FileDescriptors: yarpcFileDescriptorClosure301ba429865f230b,
			},
		}
	}
}

type _AllYARPCCaller struct {
	streamClient protobuf.StreamClient
}

func (c *_AllYARPCCaller) GetValue(ctx context.Context, request *GetValueRequest, options ...yarpc.CallOption) (*GetValueResponse, error) {
	responseMessage, err := c.streamClient.Call(ctx, "GetValue", request, newAllServiceGetValueYARPCResponse, options...)
	if responseMessage == nil {
		return nil, err
	}
	response, ok := responseMessage.(*GetValueResponse)
	if !ok {
		return nil, protobuf.CastError(emptyAllServiceGetValueYARPCResponse, responseMessage)
	}
	return response, err
}

func (c *_AllYARPCCaller) SetValue(ctx context.Context, request *SetValueRequest, options ...yarpc.CallOption) (*SetValueResponse, error) {
	responseMessage, err := c.streamClient.Call(ctx, "SetValue", request, newAllServiceSetValueYARPCResponse, options...)
	if responseMessage == nil {
		return nil, err
	}
	response, ok := responseMessage.(*SetValueResponse)
	if !ok {
		return nil, protobuf.CastError(emptyAllServiceSetValueYARPCResponse, responseMessage)
	}
	return response, err
}

func (c *_AllYARPCCaller) Fire(ctx context.Context, request *FireRequest, options ...yarpc.CallOption) (yarpc.Ack, error) {
	return c.streamClient.CallOneway(ctx, "Fire", request, options...)
}

func (c *_AllYARPCCaller) HelloOne(ctx context.Context, options ...yarpc.CallOption) (AllServiceHelloOneYARPCClient, error) {
	stream, err := c.streamClient.CallStream(ctx, "HelloOne", options...)
	if err != nil {
		return nil, err
	}
	return &_AllServiceHelloOneYARPCClient{stream: stream}, nil
}

func (c *_AllYARPCCaller) HelloTwo(ctx context.Context, request *HelloRequest, options ...yarpc.CallOption) (AllServiceHelloTwoYARPCClient, error) {
	stream, err := c.streamClient.CallStream(ctx, "HelloTwo", options...)
	if err != nil {
		return nil, err
	}
	if err := stream.Send(request); err != nil {
		return nil, err
	}
	return &_AllServiceHelloTwoYARPCClient{stream: stream}, nil
}

func (c *_AllYARPCCaller) HelloThree(ctx context.Context, options ...yarpc.CallOption) (AllServiceHelloThreeYARPCClient, error) {
	stream, err := c.streamClient.CallStream(ctx, "HelloThree", options...)
	if err != nil {
		return nil, err
	}
	return &_AllServiceHelloThreeYARPCClient{stream: stream}, nil
}

type _AllYARPCHandler struct {
	server AllYARPCServer
}

func (h *_AllYARPCHandler) GetValue(ctx context.Context, requestMessage proto.Message) (proto.Message, error) {
	var request *GetValueRequest
	var ok bool
	if requestMessage != nil {
		request, ok = requestMessage.(*GetValueRequest)
		if !ok {
			return nil, protobuf.CastError(emptyAllServiceGetValueYARPCRequest, requestMessage)
		}
	}
	response, err := h.server.GetValue(ctx, request)
	if response == nil {
		return nil, err
	}
	return response, err
}

func (h *_AllYARPCHandler) SetValue(ctx context.Context, requestMessage proto.Message) (proto.Message, error) {
	var request *SetValueRequest
	var ok bool
	if requestMessage != nil {
		request, ok = requestMessage.(*SetValueRequest)
		if !ok {
			return nil, protobuf.CastError(emptyAllServiceSetValueYARPCRequest, requestMessage)
		}
	}
	response, err := h.server.SetValue(ctx, request)
	if response == nil {
		return nil, err
	}
	return response, err
}

func (h *_AllYARPCHandler) Fire(ctx context.Context, requestMessage proto.Message) error {
	var request *FireRequest
	var ok bool
	if requestMessage != nil {
		request, ok = requestMessage.(*FireRequest)
		if !ok {
			return protobuf.CastError(emptyAllServiceFireYARPCRequest, requestMessage)
		}
	}
	return h.server.Fire(ctx, request)
}

func (h *_AllYARPCHandler) HelloOne(serverStream *protobuf.ServerStream) error {
	response, err := h.server.HelloOne(&_AllServiceHelloOneYARPCServer{serverStream: serverStream})
	if err != nil {
		return err
	}
	return serverStream.Send(response)
}

func (h *_AllYARPCHandler) HelloTwo(serverStream *protobuf.ServerStream) error {
	requestMessage, err := serverStream.Receive(newAllServiceHelloTwoYARPCRequest)
	if requestMessage == nil {
		return err
	}

	request, ok := requestMessage.(*HelloRequest)
	if !ok {
		return protobuf.CastError(emptyAllServiceHelloTwoYARPCRequest, requestMessage)
	}
	return h.server.HelloTwo(request, &_AllServiceHelloTwoYARPCServer{serverStream: serverStream})
}

func (h *_AllYARPCHandler) HelloThree(serverStream *protobuf.ServerStream) error {
	return h.server.HelloThree(&_AllServiceHelloThreeYARPCServer{serverStream: serverStream})
}

type _AllServiceHelloOneYARPCClient struct {
	stream *protobuf.ClientStream
}

func (c *_AllServiceHelloOneYARPCClient) Context() context.Context {
	return c.stream.Context()
}

func (c *_AllServiceHelloOneYARPCClient) Send(request *HelloRequest, options ...yarpc.StreamOption) error {
	return c.stream.Send(request, options...)
}

func (c *_AllServiceHelloOneYARPCClient) CloseAndRecv(options ...yarpc.StreamOption) (*HelloResponse, error) {
	if err := c.stream.Close(options...); err != nil {
		return nil, err
	}
	responseMessage, err := c.stream.Receive(newAllServiceHelloOneYARPCResponse, options...)
	if responseMessage == nil {
		return nil, err
	}
	response, ok := responseMessage.(*HelloResponse)
	if !ok {
		return nil, protobuf.CastError(emptyAllServiceHelloOneYARPCResponse, responseMessage)
	}
	return response, err
}

type _AllServiceHelloTwoYARPCClient struct {
	stream *protobuf.ClientStream
}

func (c *_AllServiceHelloTwoYARPCClient) Context() context.Context {
	return c.stream.Context()
}

func (c *_AllServiceHelloTwoYARPCClient) Recv(options ...yarpc.StreamOption) (*HelloResponse, error) {
	responseMessage, err := c.stream.Receive(newAllServiceHelloTwoYARPCResponse, options...)
	if responseMessage == nil {
		return nil, err
	}
	response, ok := responseMessage.(*HelloResponse)
	if !ok {
		return nil, protobuf.CastError(emptyAllServiceHelloTwoYARPCResponse, responseMessage)
	}
	return response, err
}

func (c *_AllServiceHelloTwoYARPCClient) CloseSend(options ...yarpc.StreamOption) error {
	return c.stream.Close(options...)
}

type _AllServiceHelloThreeYARPCClient struct {
	stream *protobuf.ClientStream
}

func (c *_AllServiceHelloThreeYARPCClient) Context() context.Context {
	return c.stream.Context()
}

func (c *_AllServiceHelloThreeYARPCClient) Send(request *HelloRequest, options ...yarpc.StreamOption) error {
	return c.stream.Send(request, options...)
}

func (c *_AllServiceHelloThreeYARPCClient) Recv(options ...yarpc.StreamOption) (*HelloResponse, error) {
	responseMessage, err := c.stream.Receive(newAllServiceHelloThreeYARPCResponse, options...)
	if responseMessage == nil {
		return nil, err
	}
	response, ok := responseMessage.(*HelloResponse)
	if !ok {
		return nil, protobuf.CastError(emptyAllServiceHelloThreeYARPCResponse, responseMessage)
	}
	return response, err
}

func (c *_AllServiceHelloThreeYARPCClient) CloseSend(options ...yarpc.StreamOption) error {
	return c.stream.Close(options...)
}

type _AllServiceHelloOneYARPCServer struct {
	serverStream *protobuf.ServerStream
}

func (s *_AllServiceHelloOneYARPCServer) Context() context.Context {
	return s.serverStream.Context()
}

func (s *_AllServiceHelloOneYARPCServer) Recv(options ...yarpc.StreamOption) (*HelloRequest, error) {
	requestMessage, err := s.serverStream.Receive(newAllServiceHelloOneYARPCRequest, options...)
	if requestMessage == nil {
		return nil, err
	}
	request, ok := requestMessage.(*HelloRequest)
	if !ok {
		return nil, protobuf.CastError(emptyAllServiceHelloOneYARPCRequest, requestMessage)
	}
	return request, err
}

type _AllServiceHelloTwoYARPCServer struct {
	serverStream *protobuf.ServerStream
}

func (s *_AllServiceHelloTwoYARPCServer) Context() context.Context {
	return s.serverStream.Context()
}

func (s *_AllServiceHelloTwoYARPCServer) Send(response *HelloResponse, options ...yarpc.StreamOption) error {
	return s.serverStream.Send(response, options...)
}

type _AllServiceHelloThreeYARPCServer struct {
	serverStream *protobuf.ServerStream
}

func (s *_AllServiceHelloThreeYARPCServer) Context() context.Context {
	return s.serverStream.Context()
}

func (s *_AllServiceHelloThreeYARPCServer) Recv(options ...yarpc.StreamOption) (*HelloRequest, error) {
	requestMessage, err := s.serverStream.Receive(newAllServiceHelloThreeYARPCRequest, options...)
	if requestMessage == nil {
		return nil, err
	}
	request, ok := requestMessage.(*HelloRequest)
	if !ok {
		return nil, protobuf.CastError(emptyAllServiceHelloThreeYARPCRequest, requestMessage)
	}
	return request, err
}

func (s *_AllServiceHelloThreeYARPCServer) Send(response *HelloResponse, options ...yarpc.StreamOption) error {
	return s.serverStream.Send(response, options...)
}

func newAllServiceGetValueYARPCRequest() proto.Message {
	return &GetValueRequest{}
}

func newAllServiceGetValueYARPCResponse() proto.Message {
	return &GetValueResponse{}
}

func newAllServiceSetValueYARPCRequest() proto.Message {
	return &SetValueRequest{}
}

func newAllServiceSetValueYARPCResponse() proto.Message {
	return &SetValueResponse{}
}

func newAllServiceFireYARPCRequest() proto.Message {
	return &FireRequest{}
}

func newAllServiceFireYARPCResponse() proto.Message {
	return &yarpcproto.Oneway{}
}

func newAllServiceHelloOneYARPCRequest() proto.Message {
	return &HelloRequest{}
}

func newAllServiceHelloOneYARPCResponse() proto.Message {
	return &HelloResponse{}
}

func newAllServiceHelloTwoYARPCRequest() proto.Message {
	return &HelloRequest{}
}

func newAllServiceHelloTwoYARPCResponse() proto.Message {
	return &HelloResponse{}
}

func newAllServiceHelloThreeYARPCRequest() proto.Message {
	return &HelloRequest{}
}

func newAllServiceHelloThreeYARPCResponse() proto.Message {
	return &HelloResponse{}
}

var (
	emptyAllServiceGetValueYARPCRequest    = &GetValueRequest{}
	emptyAllServiceGetValueYARPCResponse   = &GetValueResponse{}
	emptyAllServiceSetValueYARPCRequest    = &SetValueRequest{}
	emptyAllServiceSetValueYARPCResponse   = &SetValueResponse{}
	emptyAllServiceFireYARPCRequest        = &FireRequest{}
	emptyAllServiceFireYARPCResponse       = &yarpcproto.Oneway{}
	emptyAllServiceHelloOneYARPCRequest    = &HelloRequest{}
	emptyAllServiceHelloOneYARPCResponse   = &HelloResponse{}
	emptyAllServiceHelloTwoYARPCRequest    = &HelloRequest{}
	emptyAllServiceHelloTwoYARPCResponse   = &HelloResponse{}
	emptyAllServiceHelloThreeYARPCRequest  = &HelloRequest{}
	emptyAllServiceHelloThreeYARPCResponse = &HelloResponse{}
)

var yarpcFileDescriptorClosure301ba429865f230b = [][]byte{
	// encoding/protobuf/protoc-gen-yarpc-go/internal/testing/testing.proto
	[]byte{
		0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xec, 0x94, 0x41, 0x6f, 0x94, 0x40,
		0x14, 0xc7, 0x3b, 0xdb, 0xea, 0xee, 0xbe, 0xaa, 0x6d, 0x26, 0xc6, 0x54, 0x0e, 0xb5, 0xc1, 0xcb,
		0x5e, 0x76, 0x68, 0xd0, 0x8b, 0x17, 0xa3, 0xa6, 0x51, 0x93, 0x6a, 0xd6, 0x40, 0x63, 0xa2, 0x17,
		0xc3, 0xc2, 0x13, 0x27, 0x90, 0x19, 0x0a, 0x83, 0x84, 0xcf, 0xe4, 0xc1, 0xef, 0xe4, 0x07, 0xf0,
		0x33, 0x18, 0x66, 0xa0, 0x90, 0x6a, 0x62, 0xe2, 0xa2, 0xe9, 0xc1, 0x13, 0x8f, 0x99, 0xff, 0xfc,
		0xde, 0xff, 0xbd, 0x37, 0x00, 0x27, 0x28, 0x42, 0x19, 0x71, 0x11, 0x3b, 0x59, 0x2e, 0x95, 0x5c,
		0x97, 0x1f, 0x4d, 0x10, 0x2e, 0x63, 0x14, 0xcb, 0x3a, 0xc8, 0xb3, 0x70, 0x19, 0x4b, 0x87, 0x0b,
		0x85, 0xb9, 0x08, 0x52, 0x47, 0x61, 0xa1, 0x1a, 0x75, 0xfb, 0x64, 0x5a, 0x4c, 0x1f, 0x97, 0x6b,
		0xcc, 0x99, 0x56, 0xb3, 0x0e, 0xc8, 0x3a, 0xa0, 0x09, 0xc2, 0x18, 0x85, 0x16, 0xc4, 0x92, 0x75,
		0x34, 0xd6, 0x52, 0xac, 0x27, 0x7f, 0xe8, 0x22, 0xc2, 0xcc, 0xd0, 0xad, 0x3b, 0x5a, 0xa4, 0x63,
		0xc7, 0xf8, 0x30, 0xeb, 0x87, 0xb1, 0x94, 0x71, 0x8a, 0x3d, 0x37, 0x2a, 0xf3, 0x40, 0x71, 0x29,
		0xcc, 0xbe, 0xfd, 0x0e, 0xf6, 0x5e, 0xa0, 0x7a, 0x1b, 0xa4, 0x25, 0x7a, 0x78, 0x5e, 0x62, 0xa1,
		0xe8, 0x3e, 0x6c, 0x27, 0x58, 0x1f, 0x90, 0x23, 0xb2, 0x98, 0x7b, 0x4d, 0x48, 0x1f, 0xc2, 0xac,
		0x4a, 0xd4, 0x87, 0x26, 0xeb, 0xc1, 0xe4, 0x88, 0x2c, 0x76, 0xdd, 0xbb, 0xcc, 0x70, 0xfb, 0x22,
		0x4f, 0x5a, 0xae, 0x37, 0xad, 0x12, 0x75, 0x86, 0x85, 0xb2, 0x17, 0xb0, 0xdf, 0xa3, 0x8b, 0x4c,
		0x8a, 0x02, 0xe9, 0x6d, 0xb8, 0xf6, 0xb9, 0x59, 0xd0, 0x98, 0xb9, 0x67, 0x5e, 0xec, 0x47, 0xb0,
		0xe7, 0xff, 0xd6, 0xc4, 0xaf, 0x8f, 0xde, 0x87, 0xdd, 0xe7, 0x3c, 0xbf, 0x38, 0x76, 0x21, 0x22,
		0x43, 0xd1, 0x21, 0xdc, 0x78, 0x89, 0x69, 0x2a, 0x3b, 0xd5, 0x2d, 0x98, 0xf0, 0xa8, 0x95, 0x4c,
		0x78, 0x64, 0xdf, 0x83, 0x9b, 0xed, 0x7e, 0x6b, 0xf3, 0x92, 0xc0, 0xfd, 0x3e, 0x81, 0xd9, 0x29,
		0xd6, 0xda, 0x21, 0xfd, 0x4a, 0x60, 0xd6, 0x15, 0x46, 0x57, 0x6c, 0xb3, 0xd1, 0xb3, 0x4b, 0xdd,
		0xb7, 0xde, 0x8c, 0x07, 0x34, 0xc5, 0xd8, 0x5b, 0xda, 0xb1, 0x3f, 0x9a, 0x63, 0x7f, 0x6c, 0xc7,
		0xfe, 0x4f, 0x8e, 0xdd, 0x73, 0xd8, 0xf1, 0xb9, 0x48, 0x28, 0x87, 0x9d, 0x66, 0xbc, 0xf4, 0x74,
		0xd3, 0x1c, 0x83, 0x4b, 0x62, 0xd1, 0x21, 0x6c, 0x25, 0xb0, 0x0a, 0x6a, 0x7b, 0xcb, 0xfd, 0x36,
		0x85, 0xed, 0xa7, 0x69, 0xfa, 0x7f, 0xbc, 0x7f, 0x7f, 0xbc, 0xff, 0x70, 0xac, 0xf4, 0x0b, 0x81,
		0x99, 0xfe, 0xb8, 0x57, 0x02, 0xe9, 0xab, 0x4d, 0xf3, 0x0d, 0x7f, 0x23, 0xd6, 0xeb, 0x91, 0x68,
		0x5d, 0x5b, 0x16, 0xa4, 0x77, 0x7b, 0x56, 0xc9, 0x2b, 0xee, 0xf6, 0x98, 0x34, 0x17, 0x0f, 0x8c,
		0xdb, 0x4f, 0x39, 0x5e, 0xfd, 0xee, 0x1e, 0x93, 0x67, 0xf3, 0xf7, 0xd3, 0x76, 0x7b, 0x7d, 0x5d,
		0x13, 0x1e, 0xfc, 0x08, 0x00, 0x00, 0xff, 0xff, 0xd2, 0xae, 0xef, 0xde, 0x02, 0x08, 0x00, 0x00,
	},
	// encoding/protobuf/protoc-gen-yarpc-go/internal/testing/dep.proto
	[]byte{
		0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x34, 0xcc, 0xb1, 0x0a, 0xc2, 0x30,
		0x14, 0x85, 0xe1, 0x4d, 0xb1, 0x93, 0xf4, 0x11, 0xdc, 0x73, 0x33, 0xb8, 0x8b, 0xf8, 0x08, 0x0a,
		0x0e, 0x6e, 0x49, 0x7a, 0x0c, 0x85, 0x72, 0x6f, 0x48, 0x6e, 0x06, 0xdf, 0x5e, 0x9a, 0x36, 0xdb,
		0x81, 0xf3, 0xf3, 0x0d, 0x77, 0x70, 0x90, 0x69, 0xe6, 0x68, 0x53, 0x16, 0x15, 0x5f, 0xbf, 0xdb,
		0x08, 0x26, 0x82, 0xcd, 0xcf, 0xe5, 0x14, 0x4c, 0x14, 0x3b, 0xb3, 0x22, 0xb3, 0x5b, 0xac, 0xa2,
		0xe8, 0x5a, 0x4f, 0x48, 0xd4, 0xc2, 0xf1, 0x56, 0x3d, 0x32, 0xb5, 0x92, 0x3a, 0x46, 0x1d, 0xdb,
		0x46, 0x88, 0xe0, 0x16, 0x44, 0xa1, 0x2e, 0xd1, 0x2e, 0x5d, 0xc6, 0xe1, 0xfc, 0x82, 0xbe, 0xdd,
		0x52, 0xf1, 0x44, 0x49, 0xc2, 0x05, 0x8f, 0xd3, 0xe7, 0xb8, 0xdf, 0xfe, 0xd0, 0x84, 0xeb, 0x3f,
		0x00, 0x00, 0xff, 0xff, 0x58, 0x5b, 0x57, 0x08, 0xa9, 0x00, 0x00, 0x00,
	},
	// yarpcproto/yarpc.proto
	[]byte{
		0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xe2, 0x12, 0xab, 0x4c, 0x2c, 0x2a,
		0x48, 0x2e, 0x28, 0xca, 0x2f, 0xc9, 0xd7, 0x07, 0x33, 0xf5, 0xc0, 0x6c, 0x21, 0xae, 0xd2, 0xa4,
		0xd4, 0x22, 0x3d, 0xb0, 0x88, 0x92, 0x14, 0x17, 0x9b, 0x7f, 0x5e, 0x6a, 0x79, 0x62, 0xa5, 0x90,
		0x00, 0x17, 0x73, 0x62, 0x72, 0xb6, 0x04, 0xa3, 0x02, 0xa3, 0x06, 0x47, 0x10, 0x88, 0xe9, 0xc4,
		0x13, 0xc5, 0x85, 0x30, 0x21, 0x89, 0x0d, 0x4c, 0x19, 0x03, 0x02, 0x00, 0x00, 0xff, 0xff, 0x71,
		0x5c, 0xc2, 0xd9, 0x56, 0x00, 0x00, 0x00,
	},
	// google/protobuf/duration.proto
	[]byte{
		0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xe2, 0x92, 0x4b, 0xcf, 0xcf, 0x4f,
		0xcf, 0x49, 0xd5, 0x2f, 0x28, 0xca, 0x2f, 0xc9, 0x4f, 0x2a, 0x4d, 0xd3, 0x4f, 0x29, 0x2d, 0x4a,
		0x2c, 0xc9, 0xcc, 0xcf, 0xd3, 0x03, 0x8b, 0x08, 0xf1, 0x43, 0xe4, 0xf5, 0x60, 0xf2, 0x4a, 0x56,
		0x5c, 0x1c, 0x2e, 0x50, 0x25, 0x42, 0x12, 0x5c, 0xec, 0xc5, 0xa9, 0xc9, 0xf9, 0x79, 0x29, 0xc5,
		0x12, 0x8c, 0x0a, 0x8c, 0x1a, 0xcc, 0x41, 0x30, 0xae, 0x90, 0x08, 0x17, 0x6b, 0x5e, 0x62, 0x5e,
		0x7e, 0xb1, 0x04, 0x93, 0x02, 0xa3, 0x06, 0x6b, 0x10, 0x84, 0xe3, 0x14, 0xce, 0x25, 0x9c, 0x9c,
		0x9f, 0xab, 0x87, 0x66, 0xa4, 0x13, 0x2f, 0xcc, 0xc0, 0x00, 0x90, 0x48, 0x00, 0x63, 0x14, 0x6b,
		0x49, 0x65, 0x41, 0x6a, 0xf1, 0x0f, 0x46, 0xc6, 0x45, 0x4c, 0xcc, 0xee, 0x01, 0x4e, 0xab, 0x98,
		0xe4, 0xdc, 0x21, 0x5a, 0x02, 0xa0, 0x5a, 0xf4, 0xc2, 0x53, 0x73, 0x72, 0xbc, 0xf3, 0xf2, 0xcb,
		0xf3, 0x42, 0x40, 0x2a, 0x93, 0xd8, 0xc0, 0x66, 0x19, 0x03, 0x02, 0x00, 0x00, 0xff, 0xff, 0xbe,
		0x4a, 0xe8, 0x81, 0xce, 0x00, 0x00, 0x00,
	},
}

func init() {
	yarpc.RegisterClientBuilder(
		func(clientConfig transport.ClientConfig, structField reflect.StructField) KeyValueYARPCClient {
			return NewKeyValueYARPCClient(clientConfig, protobuf.ClientBuilderOptions(clientConfig, structField)...)
		},
	)
	yarpc.RegisterClientBuilder(
		func(clientConfig transport.ClientConfig, structField reflect.StructField) SinkYARPCClient {
			return NewSinkYARPCClient(clientConfig, protobuf.ClientBuilderOptions(clientConfig, structField)...)
		},
	)
	yarpc.RegisterClientBuilder(
		func(clientConfig transport.ClientConfig, structField reflect.StructField) AllYARPCClient {
			return NewAllYARPCClient(clientConfig, protobuf.ClientBuilderOptions(clientConfig, structField)...)
		},
	)
}
