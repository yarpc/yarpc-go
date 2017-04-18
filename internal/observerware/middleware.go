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

package observerware

import (
	"context"
	"time"

	"go.uber.org/yarpc/api/transport"

	"go.uber.org/zap"
)

// For tests.
var _timeNow = time.Now

// Middleware is logging middleware for all RPC types.
type Middleware struct {
	logger  *zap.Logger
	extract ContextExtractor
}

// New constructs a Middleware.
func New(logger *zap.Logger, extract ContextExtractor) *Middleware {
	return &Middleware{
		logger:  logger,
		extract: extract,
	}
}

// Handle implements middleware.UnaryInbound.
func (m *Middleware) Handle(ctx context.Context, req *transport.Request, w transport.ResponseWriter, h transport.UnaryHandler) error {
	start := _timeNow()
	err := h.Handle(ctx, req, w)
	m.log(ctx, "Handled inbound request.", "unary", req, _timeNow().Sub(start), err)
	return err
}

// Call implements middleware.UnaryOutbound.
func (m *Middleware) Call(ctx context.Context, req *transport.Request, out transport.UnaryOutbound) (*transport.Response, error) {
	start := _timeNow()
	res, err := out.Call(ctx, req)
	m.log(ctx, "Made outbound call.", "unary", req, _timeNow().Sub(start), err)
	return res, err
}

// HandleOneway implements middleware.OnewayInbound.
func (m *Middleware) HandleOneway(ctx context.Context, req *transport.Request, h transport.OnewayHandler) error {
	start := _timeNow()
	err := h.HandleOneway(ctx, req)
	m.log(ctx, "Handled inbound request.", "oneway", req, _timeNow().Sub(start), err)
	return err
}

// CallOneway implements middleware.OnewayOutbound.
func (m *Middleware) CallOneway(ctx context.Context, req *transport.Request, out transport.OnewayOutbound) (transport.Ack, error) {
	start := _timeNow()
	ack, err := out.CallOneway(ctx, req)
	m.log(ctx, "Made outbound call.", "oneway", req, _timeNow().Sub(start), err)
	return ack, err
}

func (m *Middleware) log(ctx context.Context, msg, rpcType string, req *transport.Request, elapsed time.Duration, err error) {
	if ce := m.logger.Check(zap.DebugLevel, msg); ce != nil {
		ce.Write(
			zap.String("rpcType", rpcType),
			zap.Object("request", req),
			zap.Duration("latency", elapsed),
			zap.Bool("successful", err == nil),
			zap.Error(err),
			m.extract(ctx),
		)
	}
}
