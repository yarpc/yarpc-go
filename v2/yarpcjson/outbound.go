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

package yarpcjson

import (
	"context"
	"encoding/json"

	yarpc "go.uber.org/yarpc/v2"
	"go.uber.org/yarpc/v2/yarpcencoding"
)

// New builds a new JSON client.
func New(c yarpc.Client) Client {
	return Client{c: c}
}

// Client is a JSON encoding porcelain for a YARPC client.
type Client struct {
	c yarpc.Client
}

// Call performs an outbound JSON request.
//
// resBodyOut is a pointer to a value that can be filled with
// json.Unmarshal.
//
// Returns the response or an error if the request failed.
func (c Client) Call(ctx context.Context, procedure string, reqBody interface{}, resBodyOut interface{}, opts ...yarpc.CallOption) error {
	call := yarpc.NewOutboundCall(opts...)
	req := yarpc.Request{
		Caller:    c.c.Caller,
		Service:   c.c.Service,
		Procedure: procedure,
		Encoding:  Encoding,
	}

	ctx, err := call.WriteToRequest(ctx, &req)
	if err != nil {
		return err
	}

	encoded, err := json.Marshal(reqBody)
	if err != nil {
		return yarpcencoding.RequestBodyEncodeError(&req, err)
	}

	res, resBuf, appErr := c.c.Unary.Call(ctx, &req, yarpc.NewBufferBytes(encoded))
	if res == nil {
		return appErr
	}

	// we want to return the appErr if it exists as this is what
	// the previous behavior was so we deprioritize this error
	var decodeErr error
	if _, err = call.ReadFromResponse(ctx, res); err != nil {
		decodeErr = err
	}
	if resBuf != nil {
		if err := json.NewDecoder(resBuf).Decode(resBodyOut); err != nil && decodeErr == nil {
			decodeErr = yarpcencoding.ResponseBodyDecodeError(&req, err)
		}
	}

	if appErr != nil {
		return appErr
	}
	return decodeErr
}
