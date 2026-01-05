// Copyright (c) 2026 Uber Technologies, Inc.
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

package tchannel

import (
	"context"
	"errors"

	"github.com/uber/tchannel-go"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/yarpcerrors"
)

// GetResponseErrorMeta extracts the TChannel specific error response information
// from an error. Returns nil if no information is available.
//
// This API is experimental and subject to change or break at anytime.
func GetResponseErrorMeta(err error) *ResponseErrorMeta {
	if err == nil {
		return nil
	}

	var meta *ytchanError
	if !errors.As(err, &meta) {
		return nil
	}
	return &ResponseErrorMeta{
		Code: meta.err.Code(),
	}
}

// ResponseErrorMeta exposes TChannel specific information from an error.
//
// This API is experimental and subject to change or break at anytime.
type ResponseErrorMeta struct {
	Code tchannel.SystemErrCode
}

// private error for propagating transparently
type ytchanError struct {
	err tchannel.SystemError
}

func (y *ytchanError) Error() string { return y.err.Message() }

func fromSystemError(err tchannel.SystemError) error {
	code, ok := _tchannelCodeToCode[err.Code()]
	if !ok {
		return yarpcerrors.Newf(yarpcerrors.CodeInternal, "got tchannel.SystemError %v which did not have a matching YARPC code", err)
	}

	// transparently wrap our private error so we can extract it with
	// GetResponseErrorMeta.
	return yarpcerrors.Newf(code, "%w", &ytchanError{err: err})
}

func toYARPCError(req *transport.Request, err error) error {
	if err == nil {
		return err
	}
	if yarpcerrors.IsStatus(err) {
		return err
	}
	if err, ok := err.(tchannel.SystemError); ok {
		return fromSystemError(err)
	}
	if err == context.DeadlineExceeded {
		return yarpcerrors.DeadlineExceededErrorf("deadline exceeded for service: %q, procedure: %q", req.Service, req.Procedure)
	}
	return yarpcerrors.UnknownErrorf("received unknown error calling service: %q, procedure: %q, err: %s", req.Service, req.Procedure, err.Error())
}
