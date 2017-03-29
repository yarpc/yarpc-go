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

package grpc

import (
	"fmt"

	"go.uber.org/yarpc/api/transport"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

type methodHandler struct {
	serviceName string
	methodName  string
	router      transport.Router
}

func newMethodHandler(serviceName string, methodName string, router transport.Router) *methodHandler {
	return &methodHandler{serviceName, methodName, router}
}

func (m *methodHandler) handle(
	server interface{},
	ctx context.Context,
	decodeFunc func(interface{}) error,
	interceptor grpc.UnaryServerInterceptor,
) (interface{}, error) {
	var data []byte
	if err := decodeFunc(&data); err != nil {
		return nil, err
	}
	fmt.Printf("%s\n%s\n%+v\n%TX%sX\n", m.serviceName, m.methodName, ctx, data, string(data))
	return nil, nil
}
