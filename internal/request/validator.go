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

package request

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"go.uber.org/yarpc/api/transport"
)

// Validator helps validate requests.
//
//	v := Validator{Request: request}
//	v.Validate()
//	...
//	v.ParseTTL(ttlstring)
//	request, err := v.ValidateUnary()
type Validator struct {
	Request *transport.Request
	errTTL  error
}

// Validate is a shortcut for the case where a request needs to be validated
// without changing the TTL. This should be used to validate all request types
func Validate(ctx context.Context, req *transport.Request) (*transport.Request, error) {
	v := Validator{Request: req}
	return v.Validate(ctx)
}

// ValidateUnary validates a unary request. This should be used after a successful Validate()
func ValidateUnary(ctx context.Context, req *transport.Request) (*transport.Request, error) {
	v := Validator{Request: req}
	return v.ValidateUnary(ctx)
}

// ValidateOneway validates a oneway request. This should be used after a successful Validate()
func ValidateOneway(ctx context.Context, req *transport.Request) (*transport.Request, error) {
	v := Validator{Request: req}
	return v.ValidateOneway(ctx)
}

// ParseTTL takes a context parses the given TTL, clamping the context to that TTL
// and as a side-effect, tracking any errors encountered while attempting to
// parse and validate that TTL. Should only be used for unary requests
func (v *Validator) ParseTTL(ctx context.Context, ttl string) (context.Context, func()) {
	if ttl == "" {
		// The TTL is missing so set it to 0 and let Validate() fail with the
		// correct error message.
		return ctx, func() {}
	}

	ttlms, err := strconv.Atoi(ttl)
	if err != nil {
		v.errTTL = invalidTTLError{
			Service:   v.Request.Service,
			Procedure: v.Request.Procedure,
			TTL:       ttl,
		}
		return ctx, func() {}
	}
	// negative TTLs are invalid
	if ttlms < 0 {
		v.errTTL = invalidTTLError{
			Service:   v.Request.Service,
			Procedure: v.Request.Procedure,
			TTL:       fmt.Sprint(ttlms),
		}
		return ctx, func() {}
	}

	return context.WithTimeout(ctx, time.Duration(ttlms)*time.Millisecond)
}

// Validate checks that the request inside this validator is valid and returns
// either the validated request or an error. This should be used to check all
// requests, prior to the RPC type specifc validation.
func (v *Validator) Validate(ctx context.Context) (*transport.Request, error) {
	// check missing params
	var missingParams []string
	if v.Request.Service == "" {
		missingParams = append(missingParams, "service name")
	}
	if v.Request.Procedure == "" {
		missingParams = append(missingParams, "procedure")
	}
	if v.Request.Caller == "" {
		missingParams = append(missingParams, "caller name")
	}
	if v.Request.Encoding == "" {
		missingParams = append(missingParams, "encoding")
	}
	if len(missingParams) > 0 {
		return nil, missingParametersError{Parameters: missingParams}
	}

	return v.Request, nil
}

// ValidateUnary validates a unary request. This should be used after a successful v.Validate()
func (v *Validator) ValidateUnary(ctx context.Context) (*transport.Request, error) {
	if v.errTTL != nil {
		return nil, v.errTTL
	}

	_, hasDeadline := ctx.Deadline()

	if !hasDeadline {
		return nil, missingParametersError{Parameters: []string{"TTL"}}
	}

	return v.Request, nil
}

// ValidateOneway validates a oneway request. This should be used after a successful Validate()
func (v *Validator) ValidateOneway(ctx context.Context) (*transport.Request, error) {
	// Currently, no extra checks for oneway requests are required
	return v.Request, nil
}
