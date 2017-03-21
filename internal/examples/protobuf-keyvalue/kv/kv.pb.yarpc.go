// Code generated by protoc-gen-yarpc-go
// source: internal/examples/protobuf-keyvalue/kv/kv.proto
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

package kv

import (
	"context"

	"github.com/golang/protobuf/proto"
	"go.uber.org/yarpc"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/encoding/x/protobuf"
)

// APIClient is the client-side interface for the API service.
type APIClient interface {
	GetValue(context.Context, *GetValueRequest, ...yarpc.CallOption) (*GetValueResponse, error)
	SetValue(context.Context, *SetValueRequest, ...yarpc.CallOption) (*SetValueResponse, error)
}

// NewAPIClient builds a new client for the API service.
func NewAPIClient(clientConfig transport.ClientConfig) APIClient {
	return &_APICaller{protobuf.NewClient("API", clientConfig)}
}

// APIServer is the server-side interface for the API service.
type APIServer interface {
	GetValue(context.Context, *GetValueRequest) (*GetValueResponse, error)
	SetValue(context.Context, *SetValueRequest) (*SetValueResponse, error)
}

// BuildAPIProcedures prepares an implementation of the API service for registration.
func BuildAPIProcedures(server APIServer) []transport.Procedure {
	handler := &_APIHandler{server}
	return protobuf.BuildProcedures(
		"API",
		map[string]transport.UnaryHandler{
			"GetValue": protobuf.NewUnaryHandler(handler.GetValue, newAPI_GetValueRequest),
			"SetValue": protobuf.NewUnaryHandler(handler.SetValue, newAPI_SetValueRequest),
		},
	)
}

// ***** all code below is private *****

type _APICaller struct {
	client protobuf.Client
}

func (c *_APICaller) GetValue(ctx context.Context, request *GetValueRequest, options ...yarpc.CallOption) (*GetValueResponse, error) {
	responseMessage, err := c.client.Call(ctx, "GetValue", request, newAPI_GetValueResponse, options...)
	if responseMessage == nil {
		return nil, err
	}
	response, ok := responseMessage.(*GetValueResponse)
	if !ok {
		return nil, protobuf.CastError(emptyAPI_GetValueResponse, responseMessage)
	}
	return response, err
}

func (c *_APICaller) SetValue(ctx context.Context, request *SetValueRequest, options ...yarpc.CallOption) (*SetValueResponse, error) {
	responseMessage, err := c.client.Call(ctx, "SetValue", request, newAPI_SetValueResponse, options...)
	if responseMessage == nil {
		return nil, err
	}
	response, ok := responseMessage.(*SetValueResponse)
	if !ok {
		return nil, protobuf.CastError(emptyAPI_SetValueResponse, responseMessage)
	}
	return response, err
}

type _APIHandler struct {
	server APIServer
}

func (h *_APIHandler) GetValue(ctx context.Context, requestMessage proto.Message) (proto.Message, error) {
	var request *GetValueRequest
	var ok bool
	if requestMessage != nil {
		request, ok = requestMessage.(*GetValueRequest)
		if !ok {
			return nil, protobuf.CastError(emptyAPI_GetValueRequest, requestMessage)
		}
	}
	return h.server.GetValue(ctx, request)
}

func (h *_APIHandler) SetValue(ctx context.Context, requestMessage proto.Message) (proto.Message, error) {
	var request *SetValueRequest
	var ok bool
	if requestMessage != nil {
		request, ok = requestMessage.(*SetValueRequest)
		if !ok {
			return nil, protobuf.CastError(emptyAPI_SetValueRequest, requestMessage)
		}
	}
	return h.server.SetValue(ctx, request)
}

func newAPI_GetValueRequest() proto.Message {
	return &GetValueRequest{}
}

func newAPI_GetValueResponse() proto.Message {
	return &GetValueResponse{}
}

func newAPI_SetValueRequest() proto.Message {
	return &SetValueRequest{}
}

func newAPI_SetValueResponse() proto.Message {
	return &SetValueResponse{}
}

var (
	emptyAPI_GetValueRequest  = &GetValueRequest{}
	emptyAPI_GetValueResponse = &GetValueResponse{}
	emptyAPI_SetValueRequest  = &SetValueRequest{}
	emptyAPI_SetValueResponse = &SetValueResponse{}
)
