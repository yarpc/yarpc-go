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
	"go.uber.org/yarpc/yarpcerrors"
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

	errCode := yarpcerrors.ErrorCode(err)
	switch errCode {
	case yarpcerrors.CodeInvalidArgument,
		yarpcerrors.CodeNotFound,
		yarpcerrors.CodeAlreadyExists,
		yarpcerrors.CodePermissionDenied,
		yarpcerrors.CodeOutOfRange,
		yarpcerrors.CodeUnauthenticated:
		c.edge.callerErrLatencies.Observe(elapsed)
		if counter, err := c.edge.callerFailures.Get(errCode.String()); err == nil {
			counter.Inc()
		}
		return
	case yarpcerrors.CodeCancelled,
		yarpcerrors.CodeUnknown,
		yarpcerrors.CodeDeadlineExceeded,
		yarpcerrors.CodeResourceExhausted,
		yarpcerrors.CodeFailedPrecondition,
		yarpcerrors.CodeAborted,
		yarpcerrors.CodeUnimplemented,
		yarpcerrors.CodeInternal,
		yarpcerrors.CodeUnavailable,
		yarpcerrors.CodeDataLoss:
		c.edge.serverErrLatencies.Observe(elapsed)
		if counter, err := c.edge.serverFailures.Get(errCode.String()); err == nil {
			counter.Inc()
		}
		return
	case yarpcerrors.CodeOK:
		// If we got "CodeOK" it really means that this is not a yarpcError, in
		// which case this is another level of "unknown" error.
		c.edge.serverErrLatencies.Observe(elapsed)
		if counter, err := c.edge.serverFailures.Get("unknown_strange"); err == nil {
			counter.Inc()
		}
		return
	}
	// If this code is executed we've hit an error code outside the usual error
	// code range, so we'll just log the string representation of that code.
	c.edge.serverErrLatencies.Observe(elapsed)
	if counter, err := c.edge.serverFailures.Get(errCode.String()); err == nil {
		counter.Inc()
	}
}
