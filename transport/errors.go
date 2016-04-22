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
func AsHandlerError(service, procedure string, err error) error {
	if err == nil {
		return err
	}

	switch e := err.(type) {
	case BadRequestError, UnexpectedError:
		return err
	case serverEncodingError:
		if e.IsResponse {
			// Error encoding the response
			return UnexpectedError{Reason: err}
		}
		return BadRequestError{Reason: err}
	default:
		return UnexpectedError{
			Reason: ProcedureFailedError{
				Service:   service,
				Procedure: procedure,
				Reason:    err,
			},
		}
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
	return "UnexpectedError: " + e.Reason.Error()
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

	if len(ps) == 2 {
		s += fmt.Sprintf("%s and %s", ps[0], ps[1])
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

// ProcedureFailedError is a failure to execute a procedure due to an
// unexpected error.
type ProcedureFailedError struct {
	Service   string
	Procedure string
	Reason    error
}

func (e ProcedureFailedError) Error() string {
	return fmt.Sprintf(`error for procedure %q of service %q: %v`,
		e.Procedure, e.Service, e.Reason)
}

//////////////////////////////////////////////////////////////////////////////
// {Request, Response} {Body, Headers} {Encoding, Decoding} errors

type serverEncodingError struct {
	Encoding  Encoding
	Caller    string
	Service   string
	Procedure string
	Reason    error

	// These parameters control whether the error is for a request or a response,
	// and whether it's for a header or body.

	IsResponse bool
	IsHeader   bool
}

func (e serverEncodingError) Error() string {
	parts := []string{"failed to"}
	if e.IsResponse {
		parts = append(parts, fmt.Sprintf("encode %q response", string(e.Encoding)))
	} else {
		parts = append(parts, fmt.Sprintf("decode %q request", string(e.Encoding)))
	}
	if e.IsHeader {
		parts = append(parts, "headers")
	} else {
		parts = append(parts, "body")
	}
	parts = append(parts,
		fmt.Sprintf("for procedure %q of service %q from caller %q: %v",
			e.Procedure, e.Service, e.Caller, e.Reason))
	return strings.Join(parts, " ")
}

func newServerEncodingError(req *Request, err error) serverEncodingError {
	return serverEncodingError{
		Encoding:  req.Encoding,
		Caller:    req.Caller,
		Service:   req.Service,
		Procedure: req.Procedure,
		Reason:    err,
	}
}

// RequestBodyDecodeError builds an error that represents a failure to decode
// the request body.
func RequestBodyDecodeError(req *Request, err error) error {
	return newServerEncodingError(req, err)
}

// ResponseBodyEncodeError builds an error that represents a failure to encode
// the response body.
func ResponseBodyEncodeError(req *Request, err error) error {
	e := newServerEncodingError(req, err)
	e.IsResponse = true
	return e
}

// RequestHeadersDecodeError builds an error that represents a failure to
// decode the request headers.
func RequestHeadersDecodeError(req *Request, err error) error {
	e := newServerEncodingError(req, err)
	e.IsHeader = true
	return e
}

// ResponseHeadersEncodeError builds an error that represents a failure to
// encode the response headers.
func ResponseHeadersEncodeError(req *Request, err error) error {
	e := newServerEncodingError(req, err)
	e.IsResponse = true
	e.IsHeader = true
	return e
}

type clientEncodingError struct {
	Encoding  Encoding
	Service   string
	Procedure string
	Reason    error

	// These parameters control whether the error is for a request or a response,
	// and whether it's for a header or body.

	IsResponse bool
	IsHeader   bool
}

func (e clientEncodingError) Error() string {
	parts := []string{"failed to"}
	if e.IsResponse {
		parts = append(parts, fmt.Sprintf("decode %q response", string(e.Encoding)))
	} else {
		parts = append(parts, fmt.Sprintf("encode %q request", string(e.Encoding)))
	}
	if e.IsHeader {
		parts = append(parts, "headers")
	} else {
		parts = append(parts, "body")
	}
	parts = append(parts,
		fmt.Sprintf("for procedure %q of service %q: %v",
			e.Procedure, e.Service, e.Reason))
	return strings.Join(parts, " ")
}

func newClientEncodingError(req *Request, err error) clientEncodingError {
	return clientEncodingError{
		Encoding:  req.Encoding,
		Service:   req.Service,
		Procedure: req.Procedure,
		Reason:    err,
	}
}

// RequestBodyEncodeError builds an error that represents a failure to encode
// the request body.
func RequestBodyEncodeError(req *Request, err error) error {
	return newClientEncodingError(req, err)
}

// ResponseBodyDecodeError builds an error that represents a failure to decode
// the response body.
func ResponseBodyDecodeError(req *Request, err error) error {
	e := newClientEncodingError(req, err)
	e.IsResponse = true
	return e
}

// RequestHeadersEncodeError builds an error that represents a failure to
// encode the request headers.
func RequestHeadersEncodeError(req *Request, err error) error {
	e := newClientEncodingError(req, err)
	e.IsHeader = true
	return e
}

// ResponseHeadersDecodeError builds an error that represents a failure to
// decode the response headers.
func ResponseHeadersDecodeError(req *Request, err error) error {
	e := newClientEncodingError(req, err)
	e.IsHeader = true
	e.IsResponse = true
	return e
}
