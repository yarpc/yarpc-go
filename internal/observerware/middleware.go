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

// Unary is middleware for unary RPCs.
type Unary struct {
	logger  *zap.Logger
	extract ContextExtractor
}

// NewUnary constructs a Unary.
func NewUnary(logger *zap.Logger, extract ContextExtractor) *Unary {
	return &Unary{
		logger:  logger.With(zap.String("rpcType", "unary")),
		extract: extract,
	}
}

// Handle implements middleware.UnaryInbound.
func (m *Unary) Handle(ctx context.Context, req *transport.Request, w transport.ResponseWriter, h transport.UnaryHandler) error {
	start := _timeNow()
	err := h.Handle(ctx, req, w)
	elapsed := _timeNow().Sub(start)

	if ce := m.logger.Check(zap.DebugLevel, "Handled inbound request."); ce != nil {
		ce.Write(
			m.extract(ctx),
			zap.Object("request", req),
			zap.Duration("latency", elapsed),
			zap.Bool("successful", err == nil),
			zap.Error(err),
		)
	}
	return err
}

// Call implements middleware.UnaryOutbound.
func (m *Unary) Call(ctx context.Context, req *transport.Request, out transport.UnaryOutbound) (*transport.Response, error) {
	start := _timeNow()
	res, err := out.Call(ctx, req)
	elapsed := _timeNow().Sub(start)

	if ce := m.logger.Check(zap.DebugLevel, "Made outbound call."); ce != nil {
		ce.Write(
			m.extract(ctx),
			zap.Object("request", req),
			zap.Duration("latency", elapsed),
			zap.Bool("successful", err == nil),
			zap.Error(err),
		)
	}
	return res, err
}
