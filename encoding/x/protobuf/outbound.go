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

	"github.com/gogo/protobuf/proto"

	"go.uber.org/yarpc"
	apiencoding "go.uber.org/yarpc/api/encoding"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/encoding/x/protobuf/internal/wirepb"
	"go.uber.org/yarpc/internal/buffer"
	"go.uber.org/yarpc/internal/encoding"
	"go.uber.org/yarpc/internal/procedure"
)

type client struct {
	serviceName  string
	clientConfig transport.ClientConfig
}

func newClient(serviceName string, clientConfig transport.ClientConfig) *client {
	return &client{serviceName, clientConfig}
}

func (c *client) Call(
	ctx context.Context,
	requestMethodName string,
	request proto.Message,
	newResponse func() proto.Message,
	options ...yarpc.CallOption,
) (proto.Message, error) {
	transportRequest := &transport.Request{
		Caller:    c.clientConfig.Caller(),
		Service:   c.clientConfig.Service(),
		Encoding:  Encoding,
		Procedure: procedure.ToName(c.serviceName, requestMethodName),
	}
	if request != nil {
		protoBuffer := getBuffer()
		defer putBuffer(protoBuffer)
		if err := protoBuffer.Marshal(request); err != nil {
			return nil, encoding.RequestBodyEncodeError(transportRequest, err)
		}
		requestData := protoBuffer.Bytes()
		if requestData != nil {
			transportRequest.Body = bytes.NewReader(requestData)
		}
	}
	call := apiencoding.NewOutboundCall(encoding.FromOptions(options)...)
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
	wireResponse := &wirepb.Response{}
	if err := proto.Unmarshal(responseData, wireResponse); err != nil {
		return nil, encoding.ResponseBodyDecodeError(transportRequest, err)
	}
	var response proto.Message
	if wireResponse.Payload != nil {
		response = newResponse()
		if err := proto.Unmarshal(wireResponse.Payload, response); err != nil {
			return nil, encoding.ResponseBodyDecodeError(transportRequest, err)
		}
	}
	if wireResponse.Error != nil {
		return response, newApplicationError(wireResponse.Error.Message)
	}
	return response, nil
}
