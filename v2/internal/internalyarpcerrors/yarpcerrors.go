// Copyright (c) 2018 Uber Technologies, Inc.
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

package internalyarpcerrors

import (
	"fmt"

	"go.uber.org/yarpc/v2/yarpcerrors"
)

// NewWithNamef calls yarpcerrors.Newf and WithName on the resulting Status.
//
// This is put in a separate package so that we can ignore this specific file
// with staticcheck and existing transports can still use this logic, as
// WithName is deprecated but we still want to handle name behavior for
// backwards compatibility.
func NewWithNamef(code yarpcerrors.Code, name string, format string, args ...interface{}) *yarpcerrors.Status {
	return yarpcerrors.Newf(code, format, args...).WithName(name)
}

// AnnotateWithInfo will take an error and add info to it's error message while
// keeping the same status code.
func AnnotateWithInfo(status *yarpcerrors.Status, format string, args ...interface{}) *yarpcerrors.Status {
	return yarpcerrors.Newf(status.Code(), "%s: %s", fmt.Sprintf(format, args...), status.Message())
}
