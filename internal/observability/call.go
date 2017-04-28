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

package observability

import (
	"context"
	"time"

	"go.uber.org/yarpc/api/transport"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// A call represents a single RPC along an edge.
//
// To prevent allocating on the heap on the request path, it's a value instead
// of a pointer.
type call struct {
	edge    *edge
	extract ContextExtractor
	fields  [5]zapcore.Field

	started time.Time
	ctx     context.Context
	req     *transport.Request
	rpcType transport.Type
	inbound bool
}

func (c call) End(err error, isApplicationError bool) {
	elapsed := _timeNow().Sub(c.started)
	c.endLogs(elapsed, err, isApplicationError)
	c.endStats(elapsed, err, isApplicationError)
}

func (c call) endLogs(elapsed time.Duration, err error, isApplicationError bool) {
	msg := "Handled inbound request."
	if !c.inbound {
		msg = "Made outbound call."
	}
	ce := c.edge.logger.Check(zap.DebugLevel, msg)
	if ce == nil {
		return
	}
	fields := c.fields[:0]
	fields = append(fields, zap.String("rpcType", c.rpcType.String()))
	fields = append(fields, zap.Duration("latency", elapsed))
	fields = append(fields, zap.Bool("successful", err == nil && !isApplicationError))
	fields = append(fields, c.extract(c.ctx))
	if isApplicationError {
		fields = append(fields, zap.String("error", "application_error"))
	} else {
		fields = append(fields, zap.Error(err))
	}
	ce.Write(fields...)
}

func (c call) endStats(elapsed time.Duration, err error, isApplicationError bool) {
	// TODO: We need a much better way to distinguish between caller and server
	// errors. See T855583.
	c.edge.calls.Inc()
	if err == nil && !isApplicationError {
		c.edge.successes.Inc()
		c.edge.latencies.Observe(elapsed)
		return
	}
	// For now, assume that all application errors are the caller's fault.
	if isApplicationError {
		c.edge.callerErrLatencies.Observe(elapsed)
		if counter, err := c.edge.callerFailures.Get("application_error"); err == nil {
			counter.Inc()
		}
		return
	}
	// Bad request errors are the caller's fault.
	if transport.IsBadRequestError(err) {
		c.edge.callerErrLatencies.Observe(elapsed)
		if counter, err := c.edge.callerFailures.Get("bad_request"); err == nil {
			counter.Inc()
		}
		return
	}
	// For now, assume that all other errors are the server's fault.
	c.edge.serverErrLatencies.Observe(elapsed)
	if transport.IsUnexpectedError(err) {
		if counter, err := c.edge.serverFailures.Get("unexpected"); err == nil {
			counter.Inc()
		}
		return
	}
	// We've encountered an error that isn't otherwise categorized.
	if counter, err := c.edge.serverFailures.Get("unknown"); err == nil {
		counter.Inc()
	}
}
