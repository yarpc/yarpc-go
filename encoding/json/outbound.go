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

package json

import (
	"bytes"
	"context"
	"encoding/json"

	"go.uber.org/yarpc"
	"go.uber.org/yarpc/internal/encoding"
	"go.uber.org/yarpc/internal/meta"
	"go.uber.org/yarpc/transport"
)

// Client makes JSON requests to a single service.
type Client interface {
	// Call performs an outbound JSON request.
	//
	// resBodyOut is a pointer to a value that can be filled with
	// json.Unmarshal.
	//
	// Returns the response or an error if the request failed.
	CallUnary(ctx context.Context, reqMeta yarpc.CallReqMeta, reqBody interface{}, resBodyOut interface{}) (yarpc.CallResMeta, error)
	CallOneway(ctx context.Context, reqMeta yarpc.CallReqMeta, reqBody interface{}, resBodyOut interface{}) (transport.Ack, error)
}

// New builds a new JSON client.
func New(c transport.Channel) Client {
	return jsonClient{ch: c}
}

func init() {
	yarpc.RegisterClientBuilder(New)
}

type jsonClient struct {
	ch transport.Channel
}

func (c jsonClient) CallUnary(ctx context.Context, reqMeta yarpc.CallReqMeta, reqBody interface{}, resBodyOut interface{}) (yarpc.CallResMeta, error) {
	treq := transport.Request{
		Caller:   c.ch.Caller(),
		Service:  c.ch.Service(),
		Encoding: Encoding,
	}
	meta.ToTransportRequest(reqMeta, &treq)

	encoded, err := json.Marshal(reqBody)
	if err != nil {
		return nil, encoding.RequestBodyEncodeError(&treq, err)
	}

	treq.Body = bytes.NewReader(encoded)
	tres, err := c.ch.GetUnaryOutbound().CallUnary(ctx, &treq)

	if err != nil {
		return nil, err
	}

	dec := json.NewDecoder(tres.Body)
	if err := dec.Decode(resBodyOut); err != nil {
		return nil, encoding.ResponseBodyDecodeError(&treq, err)
	}

	if err := tres.Body.Close(); err != nil {
		return nil, err
	}

	return meta.FromTransportResponse(tres), nil
}

func (c jsonClient) CallOneway(ctx context.Context, reqMeta yarpc.CallReqMeta, reqBody interface{}, resBodyOut interface{}) (transport.Ack, error) {
	treq := transport.Request{
		Caller:   c.ch.Caller(),
		Service:  c.ch.Service(),
		Encoding: Encoding,
	}
	meta.ToTransportRequest(reqMeta, &treq)

	encoded, err := json.Marshal(reqBody)
	if err != nil {
		return nil, encoding.RequestBodyEncodeError(&treq, err)
	}

	treq.Body = bytes.NewReader(encoded)
	ack, err := c.ch.GetOnewayOutbound().CallOneway(ctx, &treq)

	if err != nil {
		return nil, err
	}

	return ack, nil
}
