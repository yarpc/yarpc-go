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

package yarpcerrors

// Fault is a type of error.
type Fault int

const (
	// UnknownFault indicates that the fault type is unknown.
	UnknownFault Fault = iota
	// ClientFault indicates that the fault is a client fault.
	ClientFault
	// ServerFault indicates that the fault is a server fault.
	ServerFault
)

// GetFaultTypeFromError determines whether the error is a client, server or indeterminate fault based on a YARPC Code.
func GetFaultTypeFromError(err error) Fault {
	return GetFaultTypeFromCode(FromError(err).Code())
}

// GetFaultTypeFromCode determines whether the status code is a client, server or indeterminate fault based on a YARPC Code.
func GetFaultTypeFromCode(code Code) Fault {
	switch code {
	case CodeCancelled,
		CodeInvalidArgument,
		CodeNotFound,
		CodeAlreadyExists,
		CodePermissionDenied,
		CodeFailedPrecondition,
		CodeAborted,
		CodeOutOfRange,
		CodeUnauthenticated,
		CodeUnimplemented,
		CodeResourceExhausted:
		return ClientFault

	case CodeUnknown,
		CodeDeadlineExceeded,
		CodeInternal,
		CodeUnavailable,
		CodeDataLoss:
		return ServerFault
	}

	return UnknownFault
}
