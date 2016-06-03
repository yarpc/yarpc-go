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

package transport

import (
	"fmt"
	"time"
)

// The general hierarchy we have is:
//
// 	BadRequestError                 HandlerError
// 	 |                                        |
// 	 +--------> localBadRequestError <--------+
// 	 |                                        |
// 	 +--------> remoteBadRequestError         |
// 	                                          |
// 	UnexpectedError                           |
// 	 |                                        |
// 	 +--------> localUnexpectedError <--------+
// 	 |
// 	 +--------> remoteUnexpectedError
//
// Only the local versions of the error types are HandlerErrors. If a Handler
// returns one of the remote versions, they will be wrapped in a new
// UnexpectedError.

// HandlerError represents error types that can be returned from a Handler
// that the Inbound implementation MUST handle.
//
// Currently, this includes BadRequestError and UnexpectedError only. Error
// types which know how to convert themselves into BadRequestError or
// UnexpectedError may provide a `AsHandlerError() HandlerError` method.
type HandlerError interface {
	error

	handlerError()
}

type asHandlerError interface {
	AsHandlerError() HandlerError
}

// AsHandlerError converts an error into a HandlerError, leaving it unchanged
// if it is already one.
func AsHandlerError(service, procedure string, err error) error {
	if err == nil {
		return err
	}

	switch e := err.(type) {
	case HandlerError:
		return e
	case asHandlerError:
		return e.AsHandlerError()
	default:
		return procedureFailedError{
			Service:   service,
			Procedure: procedure,
			Reason:    err,
		}.AsHandlerError()
	}
}

//////////////////////////////////////////////////////////////////////////////

// BadRequestError is a failure to process a request because the request was
// invalid.
type BadRequestError interface {
	error

	badRequestError()
}

type localBadRequestError struct {
	Reason error
}

var _ BadRequestError = localBadRequestError{}
var _ HandlerError = localBadRequestError{}

// LocalBadRequestError wraps the given error into a BadRequestError.
//
// It represents a local failure while processing an invalid request.
func LocalBadRequestError(err error) HandlerError {
	return localBadRequestError{Reason: err}
}

func (localBadRequestError) handlerError()    {}
func (localBadRequestError) badRequestError() {}

func (e localBadRequestError) Error() string {
	return "BadRequest: " + e.Reason.Error()
}

type remoteBadRequestError string

var _ BadRequestError = remoteBadRequestError("")

// RemoteBadRequestError builds a new BadRequestError with the given message.
//
// It represents a BadRequest failure from a remote service.
func RemoteBadRequestError(message string) BadRequestError {
	return remoteBadRequestError(message)
}

func (remoteBadRequestError) badRequestError() {}

func (e remoteBadRequestError) Error() string {
	return string(e)
}

//////////////////////////////////////////////////////////////////////////////

// UnexpectedError is a server failure due to unhandled errors. This can be
// caused if the remote server panics while processing the request or fails to
// handle any other errors.
type UnexpectedError interface {
	error

	unexpectedError()
}

type localUnexpectedError struct {
	Reason error
}

var _ UnexpectedError = localUnexpectedError{}
var _ HandlerError = localUnexpectedError{}

// LocalUnexpectedError wraps the given error into an UnexpectedError.
//
// It represens a local failure while processing a request.
func LocalUnexpectedError(err error) HandlerError {
	return localUnexpectedError{Reason: err}
}

func (localUnexpectedError) handlerError()    {}
func (localUnexpectedError) unexpectedError() {}

func (e localUnexpectedError) Error() string {
	return "UnexpectedError: " + e.Reason.Error()
}

type remoteUnexpectedError string

var _ UnexpectedError = remoteUnexpectedError("")

// RemoteUnexpectedError builds a new UnexpectedError with the given message.
//
// It represents an UnexpectedError from a remote service.
func RemoteUnexpectedError(message string) UnexpectedError {
	return remoteUnexpectedError(message)
}

func (remoteUnexpectedError) unexpectedError() {}

func (e remoteUnexpectedError) Error() string {
	return string(e)
}

//////////////////////////////////////////////////////////////////////////////

// unrecognizedProcedureError is a failure to process a request because the
// procedure and/or service name was unrecognized.
type unrecognizedProcedureError struct {
	Service   string
	Procedure string
}

func (e unrecognizedProcedureError) Error() string {
	return fmt.Sprintf(`unrecognized procedure %q for service %q`, e.Procedure, e.Service)
}

func (e unrecognizedProcedureError) AsHandlerError() HandlerError {
	return LocalBadRequestError(e)
}

// procedureFailedError is a failure to execute a procedure due to an
// unexpected error.
type procedureFailedError struct {
	Service   string
	Procedure string
	Reason    error
}

func (e procedureFailedError) Error() string {
	return fmt.Sprintf(`error for procedure %q of service %q: %v`,
		e.Procedure, e.Service, e.Reason)
}

func (e procedureFailedError) AsHandlerError() HandlerError {
	return LocalUnexpectedError(e)
}

//////////////////////////////////////////////////////////////////////////////

// TimeoutError indicates that an error occurred due to a context deadline over
// the course of a request over any transport.
type TimeoutError interface {
	error

	timeoutError()
}

type timeoutError struct {
	Service   string
	Procedure string
	Duration  time.Duration
}

// NewTimeoutError constructs an instance of a TimeoutError for the given
// servier, procedure, and duration waited.
func NewTimeoutError(Service string, Procedure string, Duration time.Duration) TimeoutError {
	return timeoutError{
		Service:   Service,
		Procedure: Procedure,
		Duration:  Duration,
	}
}

func (e timeoutError) timeoutError() {}

func (e timeoutError) Error() string {
	return fmt.Sprintf(`timeout for procedure %q of service %q after %v`,
		e.Procedure, e.Service, e.Duration)
}
