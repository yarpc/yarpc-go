// Code generated by protoc-gen-yarpc-go
// source: hello.proto
// DO NOT EDIT!

package hello

import (
	"context"
	"reflect"

	"github.com/gogo/protobuf/proto"
	"go.uber.org/yarpc"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/encoding/protobuf"
)

// TacoTruckYARPCClient is the YARPC client-side interface for the TacoTruck service.
type TacoTruckYARPCClient interface {
	Order(context.Context, *OrderRequest, ...yarpc.CallOption) (*OrderResponse, error)
}

// NewTacoTruckYARPCClient builds a new YARPC client for the TacoTruck service.
func NewTacoTruckYARPCClient(clientConfig transport.ClientConfig, options ...protobuf.ClientOption) TacoTruckYARPCClient {
	return &_TacoTruckYARPCCaller{protobuf.NewClient(
		protobuf.ClientParams{
			ServiceName:  "hello.TacoTruck",
			ClientConfig: clientConfig,
			Options:      options,
		},
	)}
}

// TacoTruckYARPCServer is the YARPC server-side interface for the TacoTruck service.
type TacoTruckYARPCServer interface {
	Order(context.Context, *OrderRequest) (*OrderResponse, error)
}

// BuildTacoTruckYARPCProcedures prepares an implementation of the TacoTruck service for YARPC registration.
func BuildTacoTruckYARPCProcedures(server TacoTruckYARPCServer) []transport.Procedure {
	handler := &_TacoTruckYARPCHandler{server}
	return protobuf.BuildProcedures(
		protobuf.BuildProceduresParams{
			ServiceName: "hello.TacoTruck",
			UnaryHandlerParams: []protobuf.BuildProceduresUnaryHandlerParams{
				{
					MethodName: "Order",
					Handler: protobuf.NewUnaryHandler(
						protobuf.UnaryHandlerParams{
							Handle:     handler.Order,
							NewRequest: newTacoTruck_OrderYARPCRequest,
						},
					),
				},
			},
			OnewayHandlerParams: []protobuf.BuildProceduresOnewayHandlerParams{},
		},
	)
}

type _TacoTruckYARPCCaller struct {
	client protobuf.Client
}

func (c *_TacoTruckYARPCCaller) Order(ctx context.Context, request *OrderRequest, options ...yarpc.CallOption) (*OrderResponse, error) {
	responseMessage, err := c.client.Call(ctx, "Order", request, newTacoTruck_OrderYARPCResponse, options...)
	if responseMessage == nil {
		return nil, err
	}
	response, ok := responseMessage.(*OrderResponse)
	if !ok {
		return nil, protobuf.CastError(emptyTacoTruck_OrderYARPCResponse, responseMessage)
	}
	return response, err
}

type _TacoTruckYARPCHandler struct {
	server TacoTruckYARPCServer
}

func (h *_TacoTruckYARPCHandler) Order(ctx context.Context, requestMessage proto.Message) (proto.Message, error) {
	var request *OrderRequest
	var ok bool
	if requestMessage != nil {
		request, ok = requestMessage.(*OrderRequest)
		if !ok {
			return nil, protobuf.CastError(emptyTacoTruck_OrderYARPCRequest, requestMessage)
		}
	}
	response, err := h.server.Order(ctx, request)
	if response == nil {
		return nil, err
	}
	return response, err
}

func newTacoTruck_OrderYARPCRequest() proto.Message {
	return &OrderRequest{}
}

func newTacoTruck_OrderYARPCResponse() proto.Message {
	return &OrderResponse{}
}

var (
	emptyTacoTruck_OrderYARPCRequest  = &OrderRequest{}
	emptyTacoTruck_OrderYARPCResponse = &OrderResponse{}
)

func init() {
	yarpc.RegisterClientBuilder(
		func(clientConfig transport.ClientConfig, structField reflect.StructField) TacoTruckYARPCClient {
			return NewTacoTruckYARPCClient(clientConfig, protobuf.ClientBuilderOptions(clientConfig, structField)...)
		},
	)
}
