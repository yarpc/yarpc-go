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

package raw

import (
	"bytes"
	"context"
	"io/ioutil"

	"go.uber.org/yarpc"
	encodingapi "go.uber.org/yarpc/api/encoding"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/pkg/encoding"
)

// Client makes Raw requests to a single service.
type Client interface {
	// Call performs a unary outbound Raw request.
	Call(ctx context.Context, procedure string, body []byte, opts ...yarpc.CallOption) ([]byte, error)
}

// New builds a new Raw client.
func New(c transport.ClientConfig) Client {
	return rawClient{cc: c}
}

func init() {
	yarpc.RegisterClientBuilder(New)
}

type rawClient struct {
	cc transport.ClientConfig
}

func (c rawClient) Call(ctx context.Context, procedure string, body []byte, opts ...yarpc.CallOption) ([]byte, error) {
	call := encodingapi.NewOutboundCall(encoding.FromOptions(opts)...)
	treq := transport.Request{
		Caller:    c.cc.Caller(),
		Service:   c.cc.Service(),
		Procedure: procedure,
		Encoding:  Encoding,
		Body:      bytes.NewReader(body),
	}

	ctx, err := call.WriteToRequest(ctx, &treq)
	if err != nil {
		return nil, err
	}

	tres, appErr := c.cc.GetUnaryOutbound().Call(ctx, &treq)
	if tres == nil {
		return nil, appErr
	}
	if tres.Body != nil {
		defer tres.Body.Close()
	}

	if _, err = call.ReadFromResponse(ctx, tres); err != nil {
		return nil, err
	}

	// we want to return the appErr if it exists as this is what
	// the previous behavior was so we deprioritize this error
	var resBody []byte
	var readErr error
	if tres.Body != nil {
		resBody, readErr = ioutil.ReadAll(tres.Body)
	}
	if appErr != nil {
		return resBody, appErr
	}
	return resBody, readErr
}
