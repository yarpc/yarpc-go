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

import "fmt"

// HandlerError represents error types that can be returned from a Handler
// that the Inbound implementation MUST handle.
//
// Currently, this includes BadRequestError and UnexpectedError only. Error
// types which know how to convert themselves into BadRequestError or
// UnexpectedError may provide a `AsHandlerError() HandlerError` method.
type HandlerError interface {
	error

	// private function because we want control over which types are valid
	// HandlerErrors.
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

// isHandlerError may be mixed into types which are valid HandlerErrors.
type isHandlerError struct{}

func (isHandlerError) handlerError() {}

//////////////////////////////////////////////////////////////////////////////

// BadRequestError is a failure to process a request because the request was
// invalid.
type BadRequestError struct {
	isHandlerError

	Reason error
}

func (e BadRequestError) Error() string {
	return "BadRequest: " + e.Reason.Error()
}

// AsHandlerError on a BadRequestError returns the error as-is.
func (e BadRequestError) AsHandlerError() HandlerError { return e }

// UnexpectedError is a failure to process a request for an unexpected reason.
type UnexpectedError struct {
	isHandlerError

	Reason error
}

func (e UnexpectedError) Error() string {
	return "UnexpectedError: " + e.Reason.Error()
}

// AsHandlerError on an UnexpectedError returns the error as-is.
func (e UnexpectedError) AsHandlerError() HandlerError { return e }

//////////////////////////////////////////////////////////////////////////////
// Private errors

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
	return BadRequestError{Reason: e}
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
	return UnexpectedError{Reason: e}
}
