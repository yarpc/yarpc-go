// Copyright (c) 2024 Uber Technologies, Inc.
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

package tracinginterceptor

import (
	"context"

	"github.com/opentracing/opentracing-go"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/internal/interceptor"
)

var (
	_ interceptor.UnaryInbound   = (*Interceptor)(nil)
	_ interceptor.UnaryOutbound  = (*Interceptor)(nil)
	_ interceptor.OnewayInbound  = (*Interceptor)(nil)
	_ interceptor.OnewayOutbound = (*Interceptor)(nil)
	_ interceptor.StreamInbound  = (*Interceptor)(nil)
	_ interceptor.StreamOutbound = (*Interceptor)(nil)
)

// Params defines the parameters for creating the Interceptor
type Params struct {
	// Tracer is used to propagate context and to generate spans
	Tracer opentracing.Tracer
	// Transport is the name of the transport, it decides the propagation format and propagation carrier
	Transport string
}

// Interceptor is the tracing interceptor for all RPC types.
// It handles both tracing observability and context propagation using OpenTracing APIs.
type Interceptor struct {
}

// New constructs a tracing interceptor with the provided parameter.
func New(p Params) *Interceptor {
	return &Interceptor{}
}

// Handle implements interceptor.UnaryInbound
func (m *Interceptor) Handle(ctx context.Context, req *transport.Request, resw transport.ResponseWriter, h transport.UnaryHandler) error {
	// TODO: implement
	panic("implement me")
}

// Call implements interceptor.UnaryOutbound
func (m *Interceptor) Call(ctx context.Context, req *transport.Request, out transport.UnaryOutbound) (*transport.Response, error) {
	// TODO: implement
	panic("implement me")
}

// HandleOneway implements interceptor.OnewayInbound
func (m *Interceptor) HandleOneway(ctx context.Context, req *transport.Request, h transport.OnewayHandler) error {
	// TODO: implement
	panic("implement me")
}

// CallOneway implements interceptor.OnewayOutbound
func (m *Interceptor) CallOneway(ctx context.Context, request *transport.Request, out transport.OnewayOutbound) (transport.Ack, error) {
	// TODO: implement
	panic("implement me")
}

// HandleStream implements interceptor.StreamInbound
func (m *Interceptor) HandleStream(s *transport.ServerStream, h transport.StreamHandler) error {
	// TODO: implement
	panic("implement me")
}

// CallStream implements interceptor.StreamOutbound
func (m *Interceptor) CallStream(ctx context.Context, req *transport.StreamRequest, out transport.StreamOutbound) (*transport.ClientStream, error) {
	// TODO: implement
	panic("implement me")
}
