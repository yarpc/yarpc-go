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
	"fmt"
	"strconv"
	"time"

	"go.uber.org/yarpc/transport"

	"context"
)

// Validator helps validate requests.
//
// 	v := Validator{Request: request}
// 	v.ParseTTL(ttlstring)
// 	request, err := v.Validate()
type Validator struct {
	Request  *transport.Request
	earlyErr error
	lateErr  error
}

// Validate is a shortcut for the case where a request needs to be validated
// without changing the TTL.
func Validate(ctx context.Context, req *transport.Request) (*transport.Request, error) {
	v := Validator{Request: req}
	return v.Validate(ctx)
}

// ParseTTL takes a context parses the given TTL, clamping the context to that TTL
// and as a side-effect, tracking any errors encountered while attempting to
// parse and validate that TTL.
func (v *Validator) ParseTTL(ctx context.Context, ttl string) (context.Context, func()) {
	if ttl == "" {
		// The TTL is missing so set it to 0 and let Validate() fail with the
		// correct error message.
		return ctx, func() {}
	}

	ttlms, err := strconv.Atoi(ttl)
	if err != nil {
		v.earlyErr = invalidTTLError{
			Service:   v.Request.Service,
			Procedure: v.Request.Procedure,
			TTL:       ttl,
		}
		return ctx, func() {}
	}
	// negative TTLs are invalid
	if ttlms < 0 {
		v.lateErr = invalidTTLError{
			Service:   v.Request.Service,
			Procedure: v.Request.Procedure,
			TTL:       fmt.Sprint(ttlms),
		}
		return ctx, func() {}
	}

	return context.WithTimeout(ctx, time.Duration(ttlms)*time.Millisecond)
}

// Validate checks that the request inside this validator is valid and returns
// either the validated request or an error.
func (v *Validator) Validate(ctx context.Context) (*transport.Request, error) {
	// already failed
	if v.earlyErr != nil {
		return nil, v.earlyErr
	}

	_, hasDeadline := ctx.Deadline()

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
	if !hasDeadline && v.lateErr == nil {
		missingParams = append(missingParams, "TTL")
	}
	if v.Request.Encoding == "" {
		missingParams = append(missingParams, "encoding")
	}
	if len(missingParams) > 0 {
		return nil, missingParametersError{Parameters: missingParams}
	}

	if v.lateErr != nil {
		return nil, v.lateErr
	}

	return v.Request, nil
}
