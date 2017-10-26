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
	"go.uber.org/yarpc/pkg/encoding"
	"go.uber.org/yarpc/pkg/errors"
	"go.uber.org/yarpc/pkg/procedure"
	"go.uber.org/yarpc/yarpcerrors"
)

type client struct {
	serviceName  string
	clientConfig transport.ClientConfig
	encoding     transport.Encoding
}

func newClient(serviceName string, clientConfig transport.ClientConfig, options ...ClientOption) *client {
	client := &client{
		serviceName:  serviceName,
		clientConfig: clientConfig,
		encoding:     Encoding,
	}
	for _, option := range options {
		option.apply(client)
	}
	return client
}

func (c *client) Call(
	ctx context.Context,
	requestMethodName string,
	request proto.Message,
	newResponse func() proto.Message,
	options ...yarpc.CallOption,
) (proto.Message, error) {
	ctx, call, transportRequest, cleanup, err := c.buildTransportRequest(ctx, requestMethodName, request, options)
	if cleanup != nil {
		defer cleanup()
	}
	if err != nil {
		return nil, err
	}
	transportResponse, appErr := c.clientConfig.GetUnaryOutbound().Call(ctx, transportRequest)
	if transportResponse == nil {
		return nil, appErr
	}
	if transportResponse.Body != nil {
		// thrift is not checking the error, should be consistent
		defer transportResponse.Body.Close()
	}
	if _, err := call.ReadFromResponse(ctx, transportResponse); err != nil {
		return nil, err
	}
	var response proto.Message
	if transportResponse.Body != nil {
		response = newResponse()
		if err := unmarshal(transportRequest.Encoding, transportResponse.Body, response); err != nil {
			return nil, errors.ResponseBodyDecodeError(transportRequest, err)
		}
	}
	return response, appErr
}

func (c *client) CallOneway(
	ctx context.Context,
	requestMethodName string,
	request proto.Message,
	options ...yarpc.CallOption,
) (transport.Ack, error) {
	ctx, _, transportRequest, cleanup, err := c.buildTransportRequest(ctx, requestMethodName, request, options)
	if cleanup != nil {
		defer cleanup()
	}
	if err != nil {
		return nil, err
	}
	return c.clientConfig.GetOnewayOutbound().CallOneway(ctx, transportRequest)
}

func (c *client) buildTransportRequest(ctx context.Context, requestMethodName string, request proto.Message, options []yarpc.CallOption) (context.Context, *apiencoding.OutboundCall, *transport.Request, func(), error) {
	transportRequest := &transport.Request{
		Caller:    c.clientConfig.Caller(),
		Service:   c.clientConfig.Service(),
		Procedure: procedure.ToName(c.serviceName, requestMethodName),
		Encoding:  c.encoding,
	}
	call := apiencoding.NewOutboundCall(encoding.FromOptions(options)...)
	ctx, err := call.WriteToRequest(ctx, transportRequest)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	if transportRequest.Encoding != Encoding && transportRequest.Encoding != JSONEncoding {
		return nil, nil, nil, nil, yarpcerrors.Newf(yarpcerrors.CodeInternal, "can only use encodings %q or %q, but %q was specified", Encoding, JSONEncoding, transportRequest.Encoding)
	}
	if request != nil {
		requestData, cleanup, err := marshal(transportRequest.Encoding, request)
		if err != nil {
			return nil, nil, nil, cleanup, errors.RequestBodyEncodeError(transportRequest, err)
		}
		if requestData != nil {
			transportRequest.Body = bytes.NewReader(requestData)
		}
		return ctx, call, transportRequest, cleanup, nil
	}
	return ctx, call, transportRequest, nil, nil
}
