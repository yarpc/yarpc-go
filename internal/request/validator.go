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

	"go.uber.org/yarpc/api/transport"
)

// Validator helps validate requests.
//
//	v := Validator{Request: request}
//	v.ValidateCommon(ctx)
//	...
//	err := v.ValidateUnary(ctx)
type Validator struct {
	Request *transport.Request
}

// ValidateUnary validates a unary request.
func ValidateUnary(ctx context.Context, req *transport.Request) error {
	v := Validator{Request: req}
	if err := v.ValidateCommon(ctx); err != nil {
		return err
	}
	return v.ValidateUnary(ctx)
}

// ValidateOneway validates a oneway request.
func ValidateOneway(ctx context.Context, req *transport.Request) error {
	v := Validator{Request: req}
	if err := v.ValidateCommon(ctx); err != nil {
		return err
	}
	return v.ValidateOneway(ctx)
}

// ValidateCommon checks validity of the common attributes of the request.
// This should be used to check ALL requests prior to calling
// RPC-type-specific validators.
func (v *Validator) ValidateCommon(ctx context.Context) error {
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
		return missingParametersError{Parameters: missingParams}
	}

	return nil
}

// ValidateUnary validates a unary request. This should be used after a
// successful v.ValidateCommon()
func (v *Validator) ValidateUnary(ctx context.Context) error {
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		return missingParametersError{Parameters: []string{"TTL"}}
	}

	return nil
}

// ValidateOneway validates a oneway request. This should be used after a
// successful ValidateCommon()
func (v *Validator) ValidateOneway(ctx context.Context) error {
	// Currently, no extra checks for oneway requests are required
	return nil
}
