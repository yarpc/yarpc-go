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
	"context"

	"go.uber.org/yarpc/api/transport"

	"github.com/golang/protobuf/proto"
)

// Client is a protobuf client.
//
// Users should use the generated protobuf Client instead of calling this directly.
type Client interface {
	Call(ctx context.Context, requestMethodName string, request proto.Message, newResponse func() proto.Message) (proto.Message, error)
}

// NewClient creates a new client.
func NewClient(serviceName string, clientConfig transport.ClientConfig) Client {
	return &client{serviceName, clientConfig}
}

type client struct {
	serviceName  string
	clientConfig transport.ClientConfig
}

func (c *client) Call(ctx context.Context, requestMethodName string, request proto.Message, newResponse func() proto.Message) (proto.Message, error) {
	return nil, nil
}
