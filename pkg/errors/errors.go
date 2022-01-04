// Copyright (c) 2022 Uber Technologies, Inc.
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

// Package errors contains helper functions for working with YARPC errors
// for encoding and transport implementations.
package errors

import "go.uber.org/yarpc/yarpcerrors"

// WrapHandlerError is a convenience function to help wrap errors returned
// from a handler.
//
// If err is nil, WrapHandlerError returns nil.
// If err is a YARPC error, WrapHandlerError returns err with no changes.
// If err is not a YARPC error, WrapHandlerError returns a new YARPC error
// with code yarpcerrors.CodeUnknown and message err.Error(), along with
// service and procedure information.
func WrapHandlerError(err error, service string, procedure string) error {
	if err == nil {
		return nil
	}
	if yarpcerrors.IsStatus(err) {
		return err
	}
	return yarpcerrors.Newf(yarpcerrors.CodeUnknown, "error for service %q and procedure %q: %s", service, procedure, err.Error())
}
