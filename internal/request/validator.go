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

	"github.com/yarpc/yarpc-go/transport"
)

// Validator helps validate requests.
//
// 	v := Validator{Request: request}
// 	v.ParseTTL(ttlstring)
// 	request, err := v.Validate()
type Validator struct {
	Request *transport.Request
	err     error
}

// Validate is a shortcut for the case where a request needs to be validated
// without changing the TTL.
func Validate(req *transport.Request) (*transport.Request, error) {
	v := Validator{Request: req}
	return v.Validate()
}

// ParseTTL parses the given TTL and updates the request's TTL value.
func (v *Validator) ParseTTL(ttl string) {
	if ttl == "" {
		// The TTL is missing so set it to 0 and let Validate() fail with the
		// correct error message.
		v.Request.TTL = 0
		return
	}

	ttlms, err := strconv.Atoi(ttl)
	if err != nil {
		v.err = invalidTTLError{
			Service:   v.Request.Service,
			Procedure: v.Request.Procedure,
			TTL:       ttl,
		}
		return
	}

	v.Request.TTL = time.Duration(ttlms) * time.Millisecond
}

// Validate checks that the request inside this validator is valid and returns
// either the validated request or an error.
func (v *Validator) Validate() (*transport.Request, error) {
	// already failed
	if v.err != nil {
		return nil, v.err
	}

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
	if v.Request.TTL == 0 {
		missingParams = append(missingParams, "TTL")
	}
	if len(missingParams) > 0 {
		return nil, missingParametersError{Parameters: missingParams}
	}

	// negative TTLs are invalid
	if v.Request.TTL < 0 {
		return nil, invalidTTLError{
			Service:   v.Request.Service,
			Procedure: v.Request.Procedure,
			TTL:       fmt.Sprint(int64(v.Request.TTL / time.Millisecond)),
		}
	}

	return v.Request, nil
}
