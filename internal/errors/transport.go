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

import (
	stderrors "errors"
	"fmt"

	"go.uber.org/yarpc/api/errors"
	"go.uber.org/yarpc/yarpcproto"
)

// ErrNoRouter indicates that Start was called without first calling
// SetRouter for an inbound transport.
var ErrNoRouter = stderrors.New("no router configured for transport inbound")

// AsHandlerError converts an error into a HandlerError, leaving it unchanged
// if it is already one.
//
// Deprecated.
func AsHandlerError(service, procedure string, err error) error {
	if err == nil {
		return nil
	}
	if errors.Type(err) == yarpcproto.ERROR_TYPE_INTERNAL {
		return err
	}
	return errors.Internal("service: %s procedure: %s message: %s", service, procedure, err.Error())
}

// UnsupportedTypeError is a failure to process a request because the RPC type
// is unknown to the transport
type UnsupportedTypeError struct {
	Transport string
	Type      string
}

func (e UnsupportedTypeError) Error() string {
	return fmt.Sprintf(`unsupported RPC type %q for transport %q`, e.Type, e.Transport)
}
