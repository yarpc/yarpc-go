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

package gob

import (
	"bytes"
	"context"
	"encoding/gob"

	"go.uber.org/yarpc"
	encodingapi "go.uber.org/yarpc/api/encoding"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/internal/encoding"
)

// Client makes gob requests to a single service.
type Client interface {
	// Call performs an outbound gob request.
	//
	// resBodyOut is a pointer to a value that can be filled with
	// gob.Decode
	//
	// Returns the response or an error if the request failed.
	Call(ctx context.Context, procedure string, reqBody interface{}, resBodyOut interface{}, opts ...yarpc.CallOption) error
	CallOneway(ctx context.Context, procedure string, reqBody interface{}, opts ...yarpc.CallOption) (transport.Ack, error)
}

// New builds a new gob client.
func New(c transport.ClientConfig) Client {
	return gobClient{cc: c}
}

func init() {
	yarpc.RegisterClientBuilder(New)
}

type gobClient struct {
	cc transport.ClientConfig
}

func (c gobClient) Call(ctx context.Context, procedure string, reqBody interface{}, resBodyOut interface{}, opts ...yarpc.CallOption) error {
	call := encodingapi.NewOutboundCall(encoding.FromOptions(opts)...)
	treq := transport.Request{
		Caller:    c.cc.Caller(),
		Service:   c.cc.Service(),
		Procedure: procedure,
		Encoding:  Encoding,
	}

	ctx, err := call.WriteToRequest(ctx, &treq)
	if err != nil {
		return err
	}

	var buff bytes.Buffer
	if err := gob.NewEncoder(&buff).Encode(reqBody); err != nil {
		return encoding.RequestBodyEncodeError(&treq, err)
	}

	treq.Body = &buff
	tres, err := c.cc.GetUnaryOutbound().Call(ctx, &treq)
	if err != nil {
		return err
	}

	if _, err = call.ReadFromResponse(ctx, tres); err != nil {
		return err
	}

	if err := gob.NewDecoder(tres.Body).Decode(resBodyOut); err != nil {
		return encoding.ResponseBodyDecodeError(&treq, err)
	}

	return tres.Body.Close()
}

func (c gobClient) CallOneway(ctx context.Context, procedure string, reqBody interface{}, opts ...yarpc.CallOption) (transport.Ack, error) {
	call := encodingapi.NewOutboundCall(encoding.FromOptions(opts)...)
	treq := transport.Request{
		Caller:    c.cc.Caller(),
		Service:   c.cc.Service(),
		Procedure: procedure,
		Encoding:  Encoding,
	}

	ctx, err := call.WriteToRequest(ctx, &treq)
	if err != nil {
		return nil, err
	}

	var buff bytes.Buffer
	if err := gob.NewEncoder(&buff).Encode(reqBody); err != nil {
		return nil, encoding.RequestBodyEncodeError(&treq, err)
	}
	treq.Body = &buff

	return c.cc.GetOnewayOutbound().CallOneway(ctx, &treq)
}
