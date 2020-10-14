// Copyright (c) 2020 Uber Technologies, Inc.
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
	"fmt"
	"time"

	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/yarpcerrors"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	_error = "error"

	_errorNameMetricsKey = "error_name"
	_notSet              = "__not_set__"

	_errorNameLogKey    = "errorName"
	_errorCodeLogKey    = "errorCode"
	_errorDetailsLogKey = "errorDetails"

	_successfulInbound  = "Handled inbound request."
	_successfulOutbound = "Made outbound call."
	_errorInbound       = "Error handling inbound request."
	_errorOutbound      = "Error making outbound call."

	_successStreamOpen  = "Successfully created stream"
	_successStreamClose = "Successfully closed stream"
	_errorStreamOpen    = "Error creating stream"
	_errorStreamClose   = "Error closing stream"

	_dropped           = "dropped"
	_droppedAppErrLog  = "dropped application error due to context timeout or cancelation"
	_droppedErrLogFmt  = "dropped error due to context timeout or cancelation: %v"
	_droppedSuccessLog = "dropped handler success due to context timeout or cancelation"
)

// A call represents a single RPC along an edge.
//
// To prevent allocating on the heap on the request path, it's a value instead
// of a pointer.
type call struct {
	edge    *edge
	extract ContextExtractor
	fields  [9]zapcore.Field

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

func (c call) End(err error, requestSize int) {
	c.endWithAppError(err, false /* isApplicationError */, nil /* applicationErrorMeta */, requestSize, 0)
}

func (c call) EndCallWithAppError(
	err error,
	isApplicationError bool,
	applicationErrorMeta *transport.ApplicationErrorMeta,
) {
	c.endWithAppError(err, isApplicationError, applicationErrorMeta, 0, 0)
}

func (c call) EndHandleWithAppError(err error,
	isApplicationError bool,
	applicationErrorMeta *transport.ApplicationErrorMeta,
	ctxOverrideErr error,
	requestSize int,
	responseSize int) {
	if ctxOverrideErr == nil {
		c.endWithAppError(err, isApplicationError, applicationErrorMeta, requestSize, responseSize)
		return
	}

	// We'll override the user's response with the appropriate context error. Also, log
	// the dropped response.
	var droppedField zap.Field
	if isApplicationError && err == nil { // Thrift exceptions
		droppedField = zap.String(_dropped, _droppedAppErrLog)
	} else if err != nil { // other errors
		droppedField = zap.String(_dropped, fmt.Sprintf(_droppedErrLogFmt, err))
	} else {
		droppedField = zap.String(_dropped, _droppedSuccessLog)
	}

	c.endWithAppError(ctxOverrideErr,
		false, /* application error */
		nil,   /* application failure */
		requestSize,
		responseSize,
		droppedField,
	)
}

func (c call) endWithAppError(
	err error,
	isApplicationError bool,
	applicationErrorMeta *transport.ApplicationErrorMeta,
	requestSize int,
	responseSize int,
	extraLogFields ...zap.Field) {
	elapsed := _timeNow().Sub(c.started)
	c.endLogs(elapsed, err, isApplicationError, applicationErrorMeta, extraLogFields...)
	c.endStats(elapsed, err, isApplicationError, applicationErrorMeta, requestSize, responseSize)
}

// EndWithPanic ends the call with additional panic metrics
func (c call) EndWithPanic(err error) {
	c.edge.panics.Inc()
	c.endWithAppError(err, true /* isApplicationError */, nil, 0, 0 /* applicationErrorMeta */)
}

func (c call) endLogs(
	elapsed time.Duration,
	err error,
	isApplicationError bool,
	applicationErrorMeta *transport.ApplicationErrorMeta,
	extraLogFields ...zap.Field) {
	appErrBitWithNoError := isApplicationError && err == nil // ie Thrift exception

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

		// For logging purposes, application errors are
		//  - Thrift exceptions (appErrBitWithNoError == true)
		//  - `yarpcerror`s with error details (ie created with `encoding/protobuf.NewError`)
		//
		// This will be the least surprising behavior for users migrating from
		// Thrift exceptions to Protobuf error details.
		//
		// Unfortunately, all errors returned from a Protobuf handler are marked as
		// an application error on the 'transport.ResponseWriter'. Therefore, we
		// distinguish an application error from a regular error by inspecting if an
		// error detail was set.
		//
		// https://github.com/yarpc/yarpc-go/pull/1912
		hasErrDetails := len(yarpcerrors.FromError(err).Details()) > 0
		if appErrBitWithNoError || (isApplicationError && hasErrDetails) {
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

	if appErrBitWithNoError { // Thrift exception
		fields = append(fields, zap.String(_error, "application_error"))
		if applicationErrorMeta != nil {
			if applicationErrorMeta.Code != nil {
				fields = append(fields, zap.String(_errorCodeLogKey, applicationErrorMeta.Code.String()))
			}
			if applicationErrorMeta.Name != "" {
				fields = append(fields, zap.String(_errorNameLogKey, applicationErrorMeta.Name))
			}
			if applicationErrorMeta.Details != "" {
				fields = append(fields, zap.String(_errorDetailsLogKey, applicationErrorMeta.Details))
			}
		}

	} else if isApplicationError { // Protobuf error
		fields = append(fields, zap.Error(err))
		fields = append(fields, zap.String(_errorCodeLogKey, yarpcerrors.FromError(err).Code().String()))
		if applicationErrorMeta != nil {
			// ignore transport.ApplicationErrorMeta#Code, since we should get this
			// directly from the error
			if applicationErrorMeta.Name != "" {
				fields = append(fields, zap.String(_errorNameLogKey, applicationErrorMeta.Name))
			}
			if applicationErrorMeta.Details != "" {
				fields = append(fields, zap.String(_errorDetailsLogKey, applicationErrorMeta.Details))
			}
		}

	} else if err != nil { // unknown error
		fields = append(fields, zap.Error(err))
		fields = append(fields, zap.String(_errorCodeLogKey, yarpcerrors.FromError(err).Code().String()))
	}

	fields = append(fields, extraLogFields...)
	ce.Write(fields...)
}

func (c call) endStats(
	elapsed time.Duration,
	err error,
	isApplicationError bool,
	applicationErrorMeta *transport.ApplicationErrorMeta,
	requestSize int,
	responseSize int,
) {
	c.edge.calls.Inc()

	if c.direction == _directionInbound {
		if deadlineTime, ok := c.ctx.Deadline(); ok {
			c.edge.ttls.Observe(deadlineTime.Sub(c.started))
		}

		if requestSize > 0 {
			c.edge.requestPayloadSizes.IncBucket(int64(requestSize))
		}
	}

	if err == nil && !isApplicationError {
		c.edge.successes.Inc()
		c.edge.latencies.Observe(elapsed)

		if c.direction == _directionInbound && responseSize > 0 {
			c.edge.responsePayloadSizes.IncBucket(int64(responseSize))
		}
		return
	}

	appErrorName := _notSet
	if applicationErrorMeta != nil && applicationErrorMeta.Name != "" {
		appErrorName = applicationErrorMeta.Name
	}

	if yarpcerrors.IsStatus(err) {
		status := yarpcerrors.FromError(err)
		errCode := status.Code()
		c.endStatsFromFault(elapsed, errCode, appErrorName)
		return
	}

	if isApplicationError {
		if applicationErrorMeta != nil && applicationErrorMeta.Code != nil {
			c.endStatsFromFault(elapsed, *applicationErrorMeta.Code, appErrorName)
			return
		}

		// It is an application error but not a Status and no YARPC Code is found.
		// Assume it's a caller's fault and emit generic error data.
		c.edge.callerErrLatencies.Observe(elapsed)

		if counter, err := c.edge.callerFailures.Get(
			_error, "application_error",
			_errorNameMetricsKey, appErrorName,
		); err == nil {
			counter.Inc()
		}
		return
	}

	c.edge.serverErrLatencies.Observe(elapsed)
	if counter, err := c.edge.serverFailures.Get(
		_error, "unknown_internal_yarpc",
		_errorNameMetricsKey, _notSet,
	); err == nil {
		counter.Inc()
	}
}

// Emits stats based on a caller or server failure, inferred by a YARPC Code.
func (c call) endStatsFromFault(elapsed time.Duration, code yarpcerrors.Code, applicationErrorName string) {
	switch faultFromCode(code) {
	case clientFault:
		c.edge.callerErrLatencies.Observe(elapsed)
		if counter, err := c.edge.callerFailures.Get(
			_error, code.String(),
			_errorNameMetricsKey, applicationErrorName,
		); err == nil {
			counter.Inc()
		}

	case serverFault:
		c.edge.serverErrLatencies.Observe(elapsed)
		if counter, err := c.edge.serverFailures.Get(
			_error, code.String(),
			_errorNameMetricsKey, applicationErrorName,
		); err == nil {
			counter.Inc()
		}
		if c.direction == _directionInbound && code == yarpcerrors.CodeDeadlineExceeded {
			if deadlineTime, ok := c.ctx.Deadline(); ok {
				c.edge.timeoutTtls.Observe(deadlineTime.Sub(c.started))
			}
		}

	default:
		// If this code is executed we've hit an error code outside the usual error
		// code range, so we'll just log the string representation of that code.
		c.edge.serverErrLatencies.Observe(elapsed)
		if counter, err := c.edge.serverFailures.Get(
			_error, code.String(),
			_errorNameMetricsKey, applicationErrorName,
		); err == nil {
			counter.Inc()
		}
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
	c.logStreamEvent(err, err == nil, _successStreamOpen, _errorStreamOpen)

	c.edge.calls.Inc()
	if err == nil {
		c.edge.successes.Inc()
		c.edge.streaming.streamsActive.Inc()
		return
	}

	c.emitStreamError(err)
}

// EndStream should be invoked immediately after a stream closes.
func (c call) EndStream(err error) {
	elapsed := _timeNow().Sub(c.started)
	c.logStreamEvent(err, err == nil, _successStreamClose, _errorStreamClose, zap.Duration("duration", elapsed))

	c.edge.streaming.streamsActive.Dec()
	c.edge.streaming.streamDurations.Observe(elapsed)
	c.emitStreamError(err)
}

// EndStreamWithPanic ends the stream call with additional panic metrics
func (c call) EndStreamWithPanic(err error) {
	c.edge.panics.Inc()
	c.EndStream(err)
}

// This function resembles EndStats for unary calls. However, we do not special
// case application errors and it does not measure failure latencies as those
// measurements are irrelevant for streams.
func (c call) emitStreamError(err error) {
	if err == nil {
		return
	}

	if !yarpcerrors.IsStatus(err) {
		if counter, err := c.edge.serverFailures.Get(
			_error, "unknown_internal_yarpc",
			_errorNameMetricsKey, _notSet,
		); err == nil {
			counter.Inc()
		}
		return
	}

	// Emit finer grained metrics since the error is a yarpcerrors.Status.
	errCode := yarpcerrors.FromError(err).Code()

	switch faultFromCode(yarpcerrors.FromError(err).Code()) {
	case clientFault:
		if counter, err2 := c.edge.callerFailures.Get(
			_error, errCode.String(),
			_errorNameMetricsKey, _notSet,
		); err2 != nil {
			c.edge.logger.DPanic("could not retrieve caller failures counter", zap.Error(err2))
		} else {
			counter.Inc()
		}

	case serverFault:
		if counter, err2 := c.edge.serverFailures.Get(
			_error, errCode.String(),
			_errorNameMetricsKey, _notSet,
		); err2 != nil {
			c.edge.logger.DPanic("could not retrieve server failures counter", zap.Error(err2))
		} else {
			counter.Inc()
		}

	default:
		// If this code is executed we've hit an error code outside the usual error
		// code range, so we'll just log the string representation of that code.
		if counter, err2 := c.edge.serverFailures.Get(
			_error, errCode.String(),
			_errorNameMetricsKey, _notSet,
		); err2 != nil {
			c.edge.logger.DPanic("could not retrieve server failures counter", zap.Error(err2))
		} else {
			counter.Inc()
		}
	}
}

// logStreamEvent is a generic logging function useful for logging stream
// events.
func (c call) logStreamEvent(err error, success bool, succMsg, errMsg string, extraFields ...zap.Field) {
	var ce *zapcore.CheckedEntry
	// Success is usually only when the error is nil.
	// The only exception is when emitting an error from ReceiveMessage, which
	// returns EOF when the stream closes normally.
	if success {
		ce = c.edge.logger.Check(c.levels.success, succMsg)
	} else {
		ce = c.edge.logger.Check(c.levels.failure, errMsg)
	}

	fields := []zap.Field{
		zap.String("rpcType", c.rpcType.String()),
		zap.Bool("successful", success),
		c.extract(c.ctx),
		zap.Error(err), // no-op if err == nil
	}
	fields = append(fields, extraFields...)

	ce.Write(fields...)
}

// inteded for metric tags, this returns the yarpcerrors.Status error code name
// or "unknown_internal_yarpc"
func errToMetricString(err error) string {
	if yarpcerrors.IsStatus(err) {
		return yarpcerrors.FromError(err).Code().String()
	}
	return "unknown_internal_yarpc"
}
