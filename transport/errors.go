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
	"strings"
)

// AsHandlerError converts an error into a BadRequestError or UnexpectedError,
// leaving it unchanged if it's already one of the two.
func AsHandlerError(err error) error {
	if err == nil {
		return err
	}

	switch err.(type) {
	case BadRequestError, UnexpectedError:
		return err
	default:
		return UnexpectedError{Reason: err}
	}
}

// BadRequestError is a failure to process a request because the request was
// invalid.
type BadRequestError struct {
	Reason error
}

func (e BadRequestError) Error() string {
	return "BadRequest: " + e.Reason.Error()
	// TODO were we planning on dropping these prefixes?
}

// UnexpectedError is a failure to process a request for an unexpected reason.
type UnexpectedError struct {
	Reason error
}

func (e UnexpectedError) Error() string {
	return "Unexpected: " + e.Reason.Error()
	// TODO were we planning on dropping these prefixes?
}

// MissingParametersError is a failure to process a request because it was
// missing required parameters.
type MissingParametersError struct {
	// Names of the missing parameters.
	//
	// Precondition: len(Parameters) > 0
	Parameters []string
}

func (e MissingParametersError) Error() string {
	s := "missing "
	ps := e.Parameters
	if len(ps) == 1 {
		s += ps[0]
		return s
	}

	s += strings.Join(ps[:len(ps)-1], ", ")
	s += fmt.Sprintf(", and %s", ps[len(ps)-1])
	return s
}

// InvalidTTLError is a failure to process a request because the TTL was in an
// invalid format.
type InvalidTTLError struct {
	Service   string
	Procedure string
	TTL       string
}

func (e InvalidTTLError) Error() string {
	return fmt.Sprintf(
		`invalid TTL %q for procedure %q of service %q: must be positive integer`,
		e.TTL, e.Procedure, e.Service,
	)
}

// UnrecognizedProcedureError is a failure to process a request because the
// procedure and/or service name was unrecognized.
type UnrecognizedProcedureError struct {
	Service   string
	Procedure string
}

func (e UnrecognizedProcedureError) Error() string {
	return fmt.Sprintf(`unrecognized procedure %q for service %q`, e.Procedure, e.Service)
}
