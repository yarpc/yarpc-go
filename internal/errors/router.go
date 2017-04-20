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

package errors

import "fmt"

// UnrecognizedProcedureError indicates that a request could not be handled locally because
// the router contained no handler for the request.
type UnrecognizedProcedureError interface {
	error

	unrecognizedProcedureError()
}

// RouterUnrecognizedProcedureError returns an error indicating that the router
// could find no corresponding handler for the request.
func RouterUnrecognizedProcedureError(service, procedure string) error {
	return unrecognizedProcedureError{
		Service:   service,
		Procedure: procedure,
	}
}

// unrecognizedProcedureError is a failure to process a request because the
// procedure and/or service name was unrecognized.
type unrecognizedProcedureError struct {
	Service   string
	Procedure string
}

func (unrecognizedProcedureError) unrecognizedProcedureError() {}

func (e unrecognizedProcedureError) Error() string {
	return fmt.Sprintf(`unrecognized procedure %q for service %q`, e.Procedure, e.Service)
}

// AsHandlerError for unrecognizedProcedureError.
func (e unrecognizedProcedureError) AsHandlerError() HandlerError {
	return HandlerBadRequestError(e)
}
