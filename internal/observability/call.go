// Copyright (c) 2019 Uber Technologies, Inc.
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

const (
	_error = "error"

	_successfulInbound  = "Handled inbound request."
	_successfulOutbound = "Made outbound call."
	_errorInbound       = "Error handling inbound request."
	_errorOutbound      = "Error making outbound call."

	_successStreamOpen  = "Successfully created stream"
	_successStreamClose = "Successfully closed stream"
	_errorStreamOpen    = "Error creating stream"
	_errorStreamClose   = "Error closing stream"
)

// A call represents a single RPC along an edge.
//
// To prevent allocating on the heap on the request path, it's a value instead
// of a pointer.
type call struct {
	edge    *edge
	extract ContextExtractor
	fields  [5]zapcore.Field

	started   time.Time
	ctx       context.Context
	req       *transport.Request
	rpcType   transport.Type
	direction directionName

	levels *levels
}

type levels struct {
	success, failure, applicationError zapcore.Level
}

func (c call) End(err error) {
	c.EndWithAppError(err, false)
}

func (c call) EndWithAppError(err error, isApplicationError bool) {
	elapsed := _timeNow().Sub(c.started)
	c.endLogs(elapsed, err, isApplicationError)
	c.endStats(elapsed, err, isApplicationError)
}

func (c call) endLogs(elapsed time.Duration, err error, isApplicationError bool) {
	var ce *zapcore.CheckedEntry
	if err == nil && !isApplicationError {
		msg := _successfulInbound
		if c.direction != _directionInbound {
			msg = _successfulOutbound
		}
		ce = c.edge.logger.Check(c.levels.success, msg)
	} else {
		msg := _errorInbound
		if c.direction != _directionInbound {
			msg = _errorOutbound
		}

		lvl := c.levels.failure
		if isApplicationError {
			lvl = c.levels.applicationError
		}
		ce = c.edge.logger.Check(lvl, msg)
	}

	if ce == nil {
		return
	}

	fields := c.fields[:0]
	fields = append(fields, zap.String("rpcType", c.rpcType.String()))
	fields = append(fields, zap.Duration("latency", elapsed))
	fields = append(fields, zap.Bool("successful", err == nil && !isApplicationError))
	fields = append(fields, c.extract(c.ctx))
	if isApplicationError {
		fields = append(fields, zap.String(_error, "application_error"))
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

	isStatus := yarpcerrors.IsStatus(err)

	// If the error is not a yarpcerrors.Status, we cannot determine a code or
	// fault. Assume that these application errors are the caller's fault and emit
	// a generic error tag.
	if isApplicationError && !isStatus {
		c.edge.callerErrLatencies.Observe(elapsed)
		if counter, err := c.edge.callerFailures.Get(_error, "application_error"); err == nil {
			counter.Inc()
		}
		return
	}

	if !isStatus {
		c.edge.serverErrLatencies.Observe(elapsed)
		if counter, err := c.edge.serverFailures.Get(_error, "unknown_internal_yarpc"); err == nil {
			counter.Inc()
		}
		return
	}

	// Emit finer grained metrics since the error is a yarpcerrors.Status.
	errCode := yarpcerrors.FromError(err).Code()
	switch errCode {
	case yarpcerrors.CodeCancelled,
		yarpcerrors.CodeInvalidArgument,
		yarpcerrors.CodeNotFound,
		yarpcerrors.CodeAlreadyExists,
		yarpcerrors.CodePermissionDenied,
		yarpcerrors.CodeFailedPrecondition,
		yarpcerrors.CodeAborted,
		yarpcerrors.CodeOutOfRange,
		yarpcerrors.CodeUnimplemented,
		yarpcerrors.CodeUnauthenticated:
		c.edge.callerErrLatencies.Observe(elapsed)
		if counter, err := c.edge.callerFailures.Get(_error, errCode.String()); err == nil {
			counter.Inc()
		}
		return
	case yarpcerrors.CodeUnknown,
		yarpcerrors.CodeDeadlineExceeded,
		yarpcerrors.CodeResourceExhausted,
		yarpcerrors.CodeInternal,
		yarpcerrors.CodeUnavailable,
		yarpcerrors.CodeDataLoss:
		c.edge.serverErrLatencies.Observe(elapsed)
		if counter, err := c.edge.serverFailures.Get(_error, errCode.String()); err == nil {
			counter.Inc()
		}
		return
	}
	// If this code is executed we've hit an error code outside the usual error
	// code range, so we'll just log the string representation of that code.
	c.edge.serverErrLatencies.Observe(elapsed)
	if counter, err := c.edge.serverFailures.Get(_error, errCode.String()); err == nil {
		counter.Inc()
	}
}

// EndStreamHandshake should be invoked immediately after successfully creating
// a stream.
func (c call) EndStreamHandshake() {
	c.EndStreamHandshakeWithError(nil)
}

// EndStreamHandshakeWithError should be invoked immediately after attempting to
// create a stream.
func (c call) EndStreamHandshakeWithError(err error) {
	c.logStreamEvent(err, _successStreamOpen, _errorStreamOpen)
}

// EndStream should be invoked immediately after a stream closes.
func (c call) EndStream(err error) {
	elapsed := _timeNow().Sub(c.started)
	c.logStreamEvent(err, _successStreamClose, _errorStreamClose, zap.Duration("duration", elapsed))
}

// logStreamEvent is a generic logging function useful for logging stream
// events.
func (c call) logStreamEvent(err error, succMsg, errMsg string, extraFields ...zap.Field) {
	var ce *zapcore.CheckedEntry
	if err != nil {
		ce = c.edge.logger.Check(c.levels.failure, errMsg)
	} else {
		ce = c.edge.logger.Check(c.levels.success, succMsg)
	}

	fields := []zap.Field{
		zap.String("rpcType", c.rpcType.String()),
		zap.Bool("successful", err == nil),
		c.extract(c.ctx),
		zap.Error(err), // no-op if err == nil
	}
	fields = append(fields, extraFields...)

	ce.Write(fields...)
}
