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
	"bytes"
	"context"
	"encoding/json"

	yarpc "go.uber.org/yarpc/v2"
	"go.uber.org/yarpc/v2/yarpctransport"
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
}

// New builds a new JSON client.
func New(c yarpctransport.ClientConfig) Client {
	return jsonClient{cc: c}
}

type jsonClient struct {
	cc yarpctransport.ClientConfig
}

func (c jsonClient) Call(ctx context.Context, procedure string, reqBody interface{}, resBodyOut interface{}, opts ...yarpc.CallOption) error {
	call := yarpc.NewOutboundCall(opts...)
	treq := yarpctransport.Request{
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
		return yarpctransport.RequestBodyEncodeError(&treq, err)
	}

	treq.Body = bytes.NewReader(encoded)
	tres, appErr := c.cc.GetUnaryOutbound().Call(ctx, &treq)
	if tres == nil {
		return appErr
	}

	// we want to return the appErr if it exists as this is what
	// the previous behavior was so we deprioritize this error
	var decodeErr error
	if _, err = call.ReadFromResponse(ctx, tres); err != nil {
		decodeErr = err
	}
	if tres.Body != nil {
		if err := json.NewDecoder(tres.Body).Decode(resBodyOut); err != nil && decodeErr == nil {
			decodeErr = yarpctransport.ResponseBodyDecodeError(&treq, err)
		}
		if err := tres.Body.Close(); err != nil && decodeErr == nil {
			decodeErr = err
		}
	}

	if appErr != nil {
		return appErr
	}
	return decodeErr
}
