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
		protoReq proto.Message,
		protoRes proto.Message,
		opts ...yarpc.CallOption,
	) (proto.Message, error)
}

// StreamClient is a protobuf client with streaming.
type StreamClient interface {
	Client

	CallStream(ctx context.Context, method string, opts ...yarpc.CallOption) (*ClientStream, error)
}

type client struct {
	c            yarpc.Client
	encoding     yarpc.Encoding
	protoService string
}

// NewClient creates a new client.
func NewClient(c yarpc.Client, protoService string, opts ...ClientOption) Client {
	return newClient(c, protoService, opts...)
}

// NewStreamClient creates a new stream client.
func NewStreamClient(c yarpc.Client, protoService string, opts ...ClientOption) StreamClient {
	return newClient(c, protoService, opts...)
}

func newClient(c yarpc.Client, service string, opts ...ClientOption) *client {
	cli := &client{c: c, encoding: Encoding, protoService: service}
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
	ctx, req, err := c.toRequest(ctx, method, call)
	if err != nil {
		return nil, err
	}
	stream, err := c.c.Stream.CallStream(ctx, req)
	if err != nil {
		return nil, err
	}
	return &ClientStream{stream: stream}, nil
}

func (c *client) Call(
	ctx context.Context,
	method string,
	protoReq proto.Message,
	protoRes proto.Message,
	opts ...yarpc.CallOption,
) (proto.Message, error) {
	call := yarpc.NewOutboundCall(opts...)
	ctx, req, err := c.toRequest(ctx, method, call)
	if err != nil {
		return nil, err
	}

	body, cleanup, err := marshal(req.Encoding, protoReq)
	if err != nil {
		return nil, yarpcencoding.RequestBodyEncodeError(req, err)
	}
	defer cleanup()

	reqBuf := &yarpc.Buffer{}
	if _, err := reqBuf.Write(body); err != nil {
		return nil, err
	}

	res, resBuf, appErr := c.c.Unary.Call(ctx, req, reqBuf)
	if res == nil {
		return nil, appErr
	}

	if _, err := call.ReadFromResponse(ctx, res); err != nil {
		return nil, err
	}

	if resBuf != nil {
		if err := unmarshal(req.Encoding, resBuf, protoRes); err != nil {
			return nil, yarpcencoding.ResponseBodyDecodeError(req, err)
		}
	}
	return protoRes, appErr
}

// toRequest maps the outbound call to its corresponding request.
// Note that the procedure name is derived from the proto service's
// fully-qualified name, combined with the specific method we are
// calling.
//
// Given a "Store" service declared in the "keyvalue" package, the derived
// procedure for the "Get" method would be "keyvalue.Store::Get".
func (c *client) toRequest(ctx context.Context, method string, call *yarpc.OutboundCall) (context.Context, *yarpc.Request, error) {
	req := &yarpc.Request{
		Caller:    c.c.Caller,
		Service:   c.c.Service,
		Procedure: yarpcprocedure.ToName(c.protoService, method),
		Encoding:  c.encoding,
	}

	ctx, err := call.WriteToRequest(ctx, req)
	if err != nil {
		return nil, nil, err
	}

	return ctx, req, nil
}
