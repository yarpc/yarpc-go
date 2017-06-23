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

package yarpc

import "go.uber.org/yarpc/api/transport"

const (
	// CodeOK means no error; returned on success
	CodeOK = transport.CodeOK

	// CodeCancelled means the operation was cancelled, typically by the caller.
	CodeCancelled = transport.CodeCancelled

	// CodeUnknown means an unknown error. Errors raised by APIs
	// that do not return enough error information
	// may be converted to this error.
	CodeUnknown = transport.CodeUnknown

	// CodeInvalidArgument means the client specified an invalid argument.
	// Note that this differs from `FailedPrecondition`. `InvalidArgument`
	// indicates arguments that are problematic regardless of the state of
	// the system (e.g., a malformed file name).
	CodeInvalidArgument = transport.CodeInvalidArgument

	// CodeDeadlineExceeded means the deadline expired before the operation could
	// complete. For operations that change the state of the system, this error
	// may be returned even if the operation has completed successfully. For example,
	// a successful response from a server could have been delayed long
	// enough for the deadline to expire.
	CodeDeadlineExceeded = transport.CodeDeadlineExceeded

	// CodeNotFound means some requested entity (e.g., file or directory) was not found.
	// For privacy reasons, this code *may* be returned when the client
	// does not have the access rights to the entity, though such usage is
	// discouraged.
	CodeNotFound = transport.CodeNotFound

	// CodeAlreadyExists means the entity that a client attempted to create
	// (e.g., file or directory) already exists.
	CodeAlreadyExists = transport.CodeAlreadyExists

	// CodePermissionDenied means the caller does not have permission to execute
	// the specified operation. `PermissionDenied` must not be used for rejections
	// caused by exhausting some resource (use `ResourceExhausted`
	// instead for those errors). `PermissionDenied` must not be
	// used if the caller can not be identified (use `Unauthenticated`
	// instead for those errors).
	CodePermissionDenied = transport.CodePermissionDenied

	// CodeResourceExhausted means some resource has been exhausted, perhaps a per-user
	// quota, or perhaps the entire file system is out of space.
	CodeResourceExhausted = transport.CodeResourceExhausted

	// CodeFailedPrecondition means the operation was rejected because the system is not
	// in a state required for the operation's execution. For example, the directory
	// to be deleted is non-empty, an rmdir operation is applied to
	// a non-directory, etc.
	//
	// Service implementors can use the following guidelines to decide
	// between `FailedPrecondition`, `Aborted`, and `Unavailable`:
	//  (a) Use `Unavailable` if the client can retry just the failing call.
	//  (b) Use `Aborted` if the client should retry at a higher level
	//      (e.g., restarting a read-modify-write sequence).
	//  (c) Use `FailedPrecondition` if the client should not retry until
	//      the system state has been explicitly fixed. E.g., if an "rmdir"
	//      fails because the directory is non-empty, `FailedPrecondition`
	//      should be returned since the client should not retry unless
	//      the files are deleted from the directory.
	CodeFailedPrecondition = transport.CodeFailedPrecondition

	// CodeAborted means the operation was aborted, typically due to a concurrency issue
	// such as a sequencer check failure or transaction abort.
	//
	// See the guidelines above for deciding between `FailedPrecondition`,
	// `Aborted`, and `Unavailable`.
	CodeAborted = transport.CodeAborted

	// CodeOutOfRange means the operation was attempted past the valid range.
	// E.g., seeking or reading past end-of-file.
	//
	// Unlike `InvalidArgument`, this error indicates a problem that may
	// be fixed if the system state changes. For example, a 32-bit file
	// system will generate `InvalidArgument` if asked to read at an
	// offset that is not in the range [0,2^32-1], but it will generate
	// `OutOfRange` if asked to read from an offset past the current
	// file size.
	//
	// There is a fair bit of overlap between `FailedPrecondition` and
	// `OutOfRange`.  We recommend using `OutOfRange` (the more specific
	// error) when it applies so that callers who are iterating through
	// a space can easily look for an `OutOfRange` error to detect when
	// they are done.
	CodeOutOfRange = transport.CodeOutOfRange

	// CodeUnimplemented means the operation is not implemented or is not
	// supported/enabled in this service.
	CodeUnimplemented = transport.CodeUnimplemented

	// CodeInternal means an internal error. This means that some invariants expected
	// by the underlying system have been broken. This error code is reserved
	// for serious errors.
	CodeInternal = transport.CodeInternal

	// CodeUnavailable means the service is currently unavailable. This is most likely a
	// transient condition, which can be corrected by retrying with a backoff.
	//
	// See the guidelines above for deciding between `FailedPrecondition`,
	// `Aborted`, and `Unavailable`.
	CodeUnavailable = transport.CodeUnavailable

	// CodeDataLoss means unrecoverable data loss or corruption.
	CodeDataLoss = transport.CodeDataLoss

	// CodeUnauthenticated means the request does not have valid authentication
	// credentials for the operation.
	CodeUnauthenticated = transport.CodeUnauthenticated
)
