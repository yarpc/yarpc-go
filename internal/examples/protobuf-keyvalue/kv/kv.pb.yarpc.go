// Code generated by protoc-gen-yarpc-go

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
// source: kv.proto
// DO NOT EDIT!

package kv

import (
	"context"

	"github.com/golang/protobuf/proto"
	"go.uber.org/yarpc"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/encoding/protobuf"
)

// APIClient is the client-side interface for the API service.
type APIClient interface {
	GetValue(context.Context, yarpc.CallReqMeta, *GetValueRequest) (*GetValueResponse, yarpc.CallResMeta, error)
	SetValue(context.Context, yarpc.CallReqMeta, *SetValueRequest) (*SetValueResponse, yarpc.CallResMeta, error)
}

// NewAPIClient builds a new client for the API service.
func NewAPIClient(clientConfig transport.ClientConfig, opts ...protobuf.ClientOption) APIClient {
	return &_APICaller{
		protobuf.NewClient(
			"API",
			clientConfig,
			opts...,
		),
	}
}

// APIServer is the server-side interface for the API service.
type APIServer interface {
	GetValue(context.Context, yarpc.ReqMeta, *GetValueRequest) (*GetValueResponse, yarpc.ResMeta, error)
	SetValue(context.Context, yarpc.ReqMeta, *SetValueRequest) (*SetValueResponse, yarpc.ResMeta, error)
}

// BuildAPIProcedures prepares an implementation of the API service for registration.
func BuildAPIProcedures(server APIServer, opts ...protobuf.RegisterOption) []transport.Procedure {
	handler := &_APIHandler{server}
	return protobuf.BuildProcedures(
		"API",
		map[string]protobuf.UnaryHandler{
			"GetValue": protobuf.NewUnaryHandler(handler.GetValue, newAPI_GetValueRequest),
			"SetValue": protobuf.NewUnaryHandler(handler.SetValue, newAPI_SetValueRequest),
		},
		opts...,
	)
}

// ***** all code below is private *****

type _APICaller struct {
	client protobuf.Client
}

func (c *_APICaller) GetValue(ctx context.Context, reqMeta yarpc.CallReqMeta, request *GetValueRequest) (*GetValueResponse, yarpc.CallResMeta, error) {
	resMessage, resMeta, err := c.client.Call(ctx, reqMeta, "GetValue", request, newAPI_GetValueResponse)
	if resMessage == nil {
		return nil, resMeta, err
	}
	response, ok := resMessage.(*GetValueResponse)
	if !ok {
		return nil, resMeta, protobuf.ClientResponseCastError("API", "GetValue", emptyAPI_GetValueResponse, resMessage)
	}
	return response, resMeta, err
}

func (c *_APICaller) SetValue(ctx context.Context, reqMeta yarpc.CallReqMeta, request *SetValueRequest) (*SetValueResponse, yarpc.CallResMeta, error) {
	resMessage, resMeta, err := c.client.Call(ctx, reqMeta, "SetValue", request, newAPI_SetValueResponse)
	if resMessage == nil {
		return nil, resMeta, err
	}
	response, ok := resMessage.(*SetValueResponse)
	if !ok {
		return nil, resMeta, protobuf.ClientResponseCastError("API", "SetValue", emptyAPI_SetValueResponse, resMessage)
	}
	return response, resMeta, err
}

type _APIHandler struct {
	server APIServer
}

func (h *_APIHandler) GetValue(ctx context.Context, reqMeta yarpc.ReqMeta, reqMessage proto.Message) (proto.Message, error, yarpc.ResMeta, error) {
	var request *GetValueRequest
	var ok bool
	if reqMessage != nil {
		request, ok = reqMessage.(*GetValueRequest)
		if !ok {
			return nil, nil, nil, protobuf.ServerRequestCastError("API", "GetValue", emptyAPI_GetValueRequest, reqMessage)
		}
	}
	response, resMeta, err := h.server.GetValue(ctx, reqMeta, request)
	return response, err, resMeta, nil
}

func (h *_APIHandler) SetValue(ctx context.Context, reqMeta yarpc.ReqMeta, reqMessage proto.Message) (proto.Message, error, yarpc.ResMeta, error) {
	var request *SetValueRequest
	var ok bool
	if reqMessage != nil {
		request, ok = reqMessage.(*SetValueRequest)
		if !ok {
			return nil, nil, nil, protobuf.ServerRequestCastError("API", "SetValue", emptyAPI_SetValueRequest, reqMessage)
		}
	}
	response, resMeta, err := h.server.SetValue(ctx, reqMeta, request)
	return response, err, resMeta, nil
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
