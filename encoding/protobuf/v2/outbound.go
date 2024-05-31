// Copyright (c) 2024 Uber Technologies, Inc.
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

package v2

import (
	"bytes"
	"context"

	"go.uber.org/yarpc"
	apiencoding "go.uber.org/yarpc/api/encoding"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/pkg/encoding"
	"go.uber.org/yarpc/pkg/errors"
	"go.uber.org/yarpc/pkg/procedure"
	"go.uber.org/yarpc/yarpcerrors"
	"google.golang.org/protobuf/proto"
)

type client struct {
	serviceName    string
	outboundConfig *transport.OutboundConfig
	encoding       transport.Encoding
	codec          *codec
}

func newClient(serviceName string, clientConfig transport.ClientConfig, anyResolver AnyResolver, options ...ClientOption) *client {
	outboundConfig := toOutboundConfig(clientConfig)
	client := &client{
		serviceName:    serviceName,
		outboundConfig: outboundConfig,
		encoding:       Encoding,
		codec:          newCodec(anyResolver),
	}
	for _, option := range options {
		option.apply(client)
	}
	return client
}

func toOutboundConfig(cc transport.ClientConfig) *transport.OutboundConfig {
	if outboundConfig, ok := cc.(*transport.OutboundConfig); ok {
		return outboundConfig
	}
	// If the config is not an *OutboundConfig we assume the only Outbound is
	// unary and create our own outbound config.
	// If there is no unary outbound, this function will panic, but, we're kinda
	// stuck with that. (and why the hell are you passing a oneway-only client
	// config to protobuf anyway?).
	return &transport.OutboundConfig{
		CallerName: cc.Caller(),
		Outbounds: transport.Outbounds{
			ServiceName: cc.Service(),
			Unary:       cc.GetUnaryOutbound(),
		},
	}
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
	unaryOutbound := c.outboundConfig.Outbounds.Unary
	if unaryOutbound == nil {
		return nil, yarpcerrors.InternalErrorf("no unary outbounds for OutboundConfig %s", c.outboundConfig.CallerName)
	}
	transportResponse, appErr := unaryOutbound.Call(ctx, transportRequest)
	appErr = convertFromYARPCError(transportRequest.Encoding, appErr, c.codec)
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
		if err := unmarshal(transportRequest.Encoding, transportResponse.Body, response, c.codec); err != nil {
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
	onewayOutbound := c.outboundConfig.Outbounds.Oneway
	if onewayOutbound == nil {
		return nil, yarpcerrors.InternalErrorf("no oneway outbounds for OutboundConfig %s", c.outboundConfig.CallerName)
	}
	ack, err := onewayOutbound.CallOneway(ctx, transportRequest)
	return ack, convertFromYARPCError(transportRequest.Encoding, err, c.codec)
}

func (c *client) buildTransportRequest(ctx context.Context, requestMethodName string, request proto.Message, options []yarpc.CallOption) (context.Context, *apiencoding.OutboundCall, *transport.Request, func(), error) {
	transportRequest := &transport.Request{
		Caller:    c.outboundConfig.CallerName,
		Service:   c.outboundConfig.Outbounds.ServiceName,
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
		requestData, cleanup, err := marshal(transportRequest.Encoding, request, c.codec)
		if err != nil {
			return nil, nil, nil, cleanup, errors.RequestBodyEncodeError(transportRequest, err)
		}
		if requestData != nil {
			transportRequest.Body = bytes.NewReader(requestData)
			transportRequest.BodySize = len(requestData)
		}
		return ctx, call, transportRequest, cleanup, nil
	}
	return ctx, call, transportRequest, nil, nil
}

func (c *client) CallStream(
	ctx context.Context,
	requestMethodName string,
	opts ...yarpc.CallOption,
) (*ClientStream, error) {
	streamRequest := &transport.StreamRequest{
		Meta: &transport.RequestMeta{
			Caller:    c.outboundConfig.CallerName,
			Service:   c.outboundConfig.Outbounds.ServiceName,
			Procedure: procedure.ToName(c.serviceName, requestMethodName),
			Encoding:  c.encoding,
		},
	}
	call, err := apiencoding.NewStreamOutboundCall(encoding.FromOptions(opts)...)
	if err != nil {
		return nil, err
	}
	ctx, err = call.WriteToRequestMeta(ctx, streamRequest.Meta)
	if err != nil {
		return nil, err
	}
	if streamRequest.Meta.Encoding != Encoding && streamRequest.Meta.Encoding != JSONEncoding {
		return nil, yarpcerrors.InternalErrorf("can only use encodings %q or %q, but %q was specified", Encoding, JSONEncoding, streamRequest.Meta.Encoding)
	}
	streamOutbound := c.outboundConfig.Outbounds.Stream
	if streamOutbound == nil {
		return nil, yarpcerrors.InternalErrorf("no stream outbounds for OutboundConfig %s", c.outboundConfig.CallerName)
	}
	stream, err := streamOutbound.CallStream(ctx, streamRequest)
	if err != nil {
		return nil, convertFromYARPCError(streamRequest.Meta.Encoding, err, c.codec)
	}
	return &ClientStream{stream: stream, codec: c.codec}, nil
}
