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

package thrift

import (
	"github.com/yarpc/yarpc-go/transport"

	"github.com/thriftrw/thriftrw-go/protocol"
	"golang.org/x/net/context"
)

// Handler represents a Thrift request handler.
type Handler interface {
	Handle(ctx context.Context, req *Request) (*Response, error)
}

// HandlerFunc is a convenience type alias for functions that implement that act as Handlers.
type HandlerFunc func(context.Context, *Request) (*Response, error)

// Handle forwards the request to the underlying function.
func (f HandlerFunc) Handle(ctx context.Context, req *Request) (*Response, error) {
	return f(ctx, req)
}

// Service represents a Thrift service implementation.
type Service interface {
	// Name of the Thrift service.
	Name() string

	// Protocol to use for requests and responses of this service.
	Protocol() protocol.Protocol

	// Map of method name to Handler for all methods of this service.
	Handlers() map[string]Handler
}

// Register registers the handlers for the methods of the given service with the
// given Registry.
func Register(registry transport.Registry, service Service) {
	name := service.Name()
	proto := service.Protocol()
	for method, h := range service.Handlers() {
		handler := thriftHandler{Handler: h, Protocol: proto}
		registry.Register(procedureName(name, method), handler)
	}
}
