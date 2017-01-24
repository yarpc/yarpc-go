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

package json

import (
	"bytes"
	"context"
	"encoding/json"

	"go.uber.org/yarpc"
	encodingapi "go.uber.org/yarpc/api/encoding"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/internal/encoding"
)

// Client makes JSON requests to a single service.
type Client interface {
	// Call performs an outbound JSON request.
	//
	// resBodyOut is a pointer to a value that can be filled with
	// json.Unmarshal.
	//
	// Returns the response or an error if the request failed.
	Call(ctx context.Context, procedure string, reqBody interface{}, resBodyOut interface{}, opts ...yarpc.CallOption) error
	CallOneway(ctx context.Context, procedure string, reqBody interface{}, opts ...yarpc.CallOption) (transport.Ack, error)
}

// New builds a new JSON client.
func New(c transport.ClientConfig) Client {
	return jsonClient{cc: c}
}

func init() {
	yarpc.RegisterClientBuilder(New)
}

type jsonClient struct {
	cc transport.ClientConfig
}

func (c jsonClient) Call(ctx context.Context, procedure string, reqBody interface{}, resBodyOut interface{}, opts ...yarpc.CallOption) error {
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

	encoded, err := json.Marshal(reqBody)
	if err != nil {
		return encoding.RequestBodyEncodeError(&treq, err)
	}

	treq.Body = bytes.NewReader(encoded)
	tres, err := c.cc.GetUnaryOutbound().Call(ctx, &treq)
	if err != nil {
		return err
	}

	if _, err = call.ReadFromResponse(ctx, tres); err != nil {
		return err
	}

	dec := json.NewDecoder(tres.Body)
	if err := dec.Decode(resBodyOut); err != nil {
		return encoding.ResponseBodyDecodeError(&treq, err)
	}

	return tres.Body.Close()
}

func (c jsonClient) CallOneway(ctx context.Context, procedure string, reqBody interface{}, opts ...yarpc.CallOption) (transport.Ack, error) {
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
	if err := json.NewEncoder(&buff).Encode(reqBody); err != nil {
		return nil, encoding.RequestBodyEncodeError(&treq, err)
	}
	treq.Body = &buff

	return c.cc.GetOnewayOutbound().CallOneway(ctx, &treq)
}
