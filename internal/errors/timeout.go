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
	"time"

	"go.uber.org/yarpc/api/errors"
)

// HandlerTimeoutError constructs an instance of a TimeoutError representing
// a timeout that occurred during the handler execution, with the caller,
// service, procedure and duration waited.
//
// Deprecated: use errors.DeadlineExceeded instead.
func HandlerTimeoutError(caller string, service string, procedure string, duration time.Duration) error {
	return errors.DeadlineExceeded("caller", caller, "service", service, "procedure", procedure, "duration", duration.String())
}

// RemoteTimeoutError represents a TimeoutError from a remote handler.
//
// Deprecated: use errors.DeadlineExceeded instead.
func RemoteTimeoutError(message string) error {
	return errors.DeadlineExceeded("message", message)
}

// ClientTimeoutError constructs an instance of a TimeoutError representing
// a timeout that occurred while the client was waiting during a request to a
// remote handler. It includes the service, procedure and duration waited.
//
// Deprecated: use errors.DeadlineExceeded instead.
func ClientTimeoutError(service string, procedure string, duration time.Duration) error {
	return errors.DeadlineExceeded("service", service, "procedure", procedure, "duration", duration.String())
}
