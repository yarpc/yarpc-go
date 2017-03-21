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

package protobuf

import (
	"bytes"
	"context"

	"github.com/golang/protobuf/proto"

	apiencoding "go.uber.org/yarpc/api/encoding"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/encoding/x/protobuf/internal"
	"go.uber.org/yarpc/internal/buffer"
	"go.uber.org/yarpc/internal/encoding"
)

type client struct {
	serviceName  string
	clientConfig transport.ClientConfig
}

func newClient(serviceName string, clientConfig transport.ClientConfig) *client {
	return &client{serviceName, clientConfig}
}

func (c *client) Call(ctx context.Context, requestMethodName string, request proto.Message, newResponse func() proto.Message) (proto.Message, error) {
	transportRequest := &transport.Request{
		Caller:    c.clientConfig.Caller(),
		Service:   c.clientConfig.Service(),
		Encoding:  Encoding,
		Procedure: toProcedureName(c.serviceName, requestMethodName),
	}
	if request != nil {
		requestData, err := protoMarshal(request)
		if err != nil {
			return nil, encoding.RequestBodyEncodeError(transportRequest, err)
		}
		if requestData != nil {
			transportRequest.Body = bytes.NewReader(requestData)
		}
	}
	// TODO: call options
	call := apiencoding.NewOutboundCall()
	ctx, err := call.WriteToRequest(ctx, transportRequest)
	if err != nil {
		return nil, err
	}
	transportResponse, err := c.clientConfig.GetUnaryOutbound().Call(ctx, transportRequest)
	if err != nil {
		return nil, err
	}
	// thrift is not checking the error, should be consistent
	defer transportResponse.Body.Close()
	if _, err := call.ReadFromResponse(ctx, transportResponse); err != nil {
		return nil, err
	}
	buf := buffer.Get()
	defer buffer.Put(buf)
	if _, err := buf.ReadFrom(transportResponse.Body); err != nil {
		return nil, err
	}
	responseData := buf.Bytes()
	if responseData == nil {
		return nil, nil
	}
	internalResponse := &internal.Response{}
	if err := proto.Unmarshal(responseData, internalResponse); err != nil {
		return nil, encoding.ResponseBodyDecodeError(transportRequest, err)
	}
	var response proto.Message
	if internalResponse.Payload != nil {
		response = newResponse()
		if err := proto.Unmarshal(internalResponse.Payload, response); err != nil {
			return nil, encoding.ResponseBodyDecodeError(transportRequest, err)
		}
	}
	if internalResponse.Error != nil {
		return response, newApplicationError(internalResponse.Error.Message)
	}
	return response, nil
}
