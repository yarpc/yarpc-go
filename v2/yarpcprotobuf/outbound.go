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

package yarpcprotobuf

import (
	"context"

	"github.com/gogo/protobuf/proto"
	yarpc "go.uber.org/yarpc/v2"
	"go.uber.org/yarpc/v2/yarpcencoding"
	"go.uber.org/yarpc/v2/yarpcprocedure"
)

// Client is a protobuf client.
type Client interface {
	Call(
		ctx context.Context,
		method string,
		request proto.Message,
		create func() proto.Message,
		opts ...yarpc.CallOption,
	) (proto.Message, error)
}

// StreamClient is a protobuf client with streaming.
type StreamClient interface {
	Client

	CallStream(ctx context.Context, method string, opts ...yarpc.CallOption) (*ClientStream, error)
}

type client struct {
	c        yarpc.Client
	encoding yarpc.Encoding
}

// NewClient creates a new client.
func NewClient(c yarpc.Client, opts ...ClientOption) Client {
	return newClient(c, opts...)
}

// NewStreamClient creates a new stream client.
func NewStreamClient(c yarpc.Client, opts ...ClientOption) StreamClient {
	return newClient(c, opts...)
}

func newClient(c yarpc.Client, opts ...ClientOption) *client {
	cli := &client{c: c, encoding: Encoding}
	for _, o := range opts {
		o.apply(cli)
	}
	return cli
}

func (c *client) CallStream(ctx context.Context, method string, opts ...yarpc.CallOption) (*ClientStream, error) {
	call, err := yarpc.NewStreamOutboundCall(opts...)
	if err != nil {
		return nil, err
	}
	ctx, request, err := c.toYARPCRequest(ctx, method, call)
	if err != nil {
		return nil, err
	}
	stream, err := c.c.Stream.CallStream(ctx, request)
	if err != nil {
		return nil, err
	}
	return &ClientStream{stream: stream}, nil
}

func (c *client) Call(
	ctx context.Context,
	method string,
	proto proto.Message,
	create func() proto.Message,
	opts ...yarpc.CallOption,
) (proto.Message, error) {
	call := yarpc.NewOutboundCall(opts...)
	ctx, request, err := c.toYARPCRequest(ctx, method, call)
	if err != nil {
		return nil, err
	}

	body, cleanup, err := marshal(request.Encoding, proto)
	if cleanup != nil {
		defer cleanup()
	}
	if err != nil {
		return nil, yarpcencoding.RequestBodyEncodeError(request, err)
	}
	requestBuf := &yarpc.Buffer{}
	if _, err := requestBuf.Write(body); err != nil {
		return nil, err
	}

	response, responseBuf, appErr := c.c.Unary.Call(ctx, request, requestBuf)
	if response == nil {
		return nil, appErr
	}
	if _, err := call.ReadFromResponse(ctx, response); err != nil {
		return nil, err
	}

	protoResponse := create()
	if responseBuf != nil {
		if err := unmarshal(request.Encoding, responseBuf, protoResponse); err != nil {
			return nil, yarpcencoding.ResponseBodyDecodeError(request, err)
		}
	}
	return protoResponse, appErr
}

func (c *client) toYARPCRequest(ctx context.Context, method string, call *yarpc.OutboundCall) (context.Context, *yarpc.Request, error) {
	request := &yarpc.Request{
		Caller:    c.c.Caller,
		Service:   c.c.Service,
		Procedure: yarpcprocedure.ToName(c.c.Service, method),
		Encoding:  c.encoding,
	}

	ctx, err := call.WriteToRequest(ctx, request)
	if err != nil {
		return nil, nil, err
	}

	return ctx, request, nil
}
