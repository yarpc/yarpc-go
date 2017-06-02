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

package codes

import "strconv"

const (
	// Not an error; returned on success
	//
	// HTTP Mapping: 200 OK
	// Google API Mapping: Code.OK
	None Code = 0
	// The operation was cancelled, typically by the caller.
	//
	// HTTP Mapping: 499 Client Closed Request
	// Google API Mapping: Code.CANCELLED
	Cancelled Code = 1
	// Unknown error. For example, this error may be returned when
	// a `Status` value received from another address space belongs to
	// an error space that is not known in this address space. Also
	// errors raised by APIs that do not return enough error information
	// may be converted to this error.
	//
	// HTTP Mapping: 500 Internal Server Error
	// Google API Mapping: Code.UNKNOWN
	Unknown Code = 2
	// The client specified an invalid argument. Note that this differs
	// from `FailedPrecondition`. `InvalidArgument` indicates arguments
	// that are problematic regardless of the state of the system
	// (e.g., a malformed file name).
	//
	// HTTP Mapping: 400 Bad Request
	// Google API Mapping: Code.INVALID_ARGUMENT
	InvalidArgument Code = 3
	// The deadline expired before the operation could complete. For operations
	// that change the state of the system, this error may be returned
	// even if the operation has completed successfully. For example, a
	// successful response from a server could have been delayed long
	// enough for the deadline to expire.
	//
	// HTTP Mapping: 504 Gateway Timeout
	// Google API Mapping: Code.DEADLINE_EXCEEDED
	DeadlineExceeded Code = 4
	// Some requested entity (e.g., file or directory) was not found.
	// For privacy reasons, this code *may* be returned when the client
	// does not have the access rights to the entity, though such usage is
	// discouraged.
	//
	// HTTP Mapping: 404 Not Found
	// Google API Mapping: Code.NOT_FOUND
	NotFound Code = 5
	// The entity that a client attempted to create (e.g., file or directory)
	// already exists.
	//
	// HTTP Mapping: 409 Conflict
	// Google API Mapping: Code.ALREADY_EXISTS
	AlreadyExists Code = 6
	// The caller does not have permission to execute the specified
	// operation. `PermissionDenied` must not be used for rejections
	// caused by exhausting some resource (use `ResourceExhausted`
	// instead for those errors). `PermissionDenied` must not be
	// used if the caller can not be identified (use `Unauthenticated`
	// instead for those errors).
	//
	// HTTP Mapping: 403 Forbidden
	// Google API Mapping: Code.PERMISSION_DENIED
	PermissionDenied Code = 7
	// Some resource has been exhausted, perhaps a per-user quota, or
	// perhaps the entire file system is out of space.
	//
	// HTTP Mapping: 429 Too Many Requests
	// Google API Mapping: Code.RESOURCE_EXHAUSTED
	ResourceExhausted Code = 8
	// The operation was rejected because the system is not in a state
	// required for the operation's execution. For example, the directory
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
	//
	// HTTP Mapping: 400 Bad Request
	// Google API Mapping: Code.FAILED_PRECONDITION
	FailedPrecondition Code = 9
	// The operation was aborted, typically due to a concurrency issue such as
	// a sequencer check failure or transaction abort.
	//
	// See the guidelines above for deciding between `FailedPrecondition`,
	// `Aborted`, and `Unavailable`.
	//
	// HTTP Mapping: 409 Conflict
	// Google API Mapping: Code.ABORTED
	Aborted Code = 10
	// The operation was attempted past the valid range. E.g., seeking or
	// reading past end-of-file.
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
	//
	// HTTP Mapping: 400 Bad Request
	// Google API Mapping: Code.OUT_OF_RANGE
	OutOfRange Code = 11
	// The operation is not implemented or is not supported/enabled in this
	// service.
	//
	// HTTP Mapping: 501 Not Implemented
	// Google API Mapping: Code.UNIMPLEMENTED
	Unimplemented Code = 12
	// Internal errors. This means that some invariants expected by the
	// underlying system have been broken. This error code is reserved
	// for serious errors.
	//
	// HTTP Mapping: 500 Internal Server Error
	// Google API Mapping: Code.INTERNAL
	Internal Code = 13
	// The service is currently unavailable. This is most likely a
	// transient condition, which can be corrected by retrying with
	// a backoff.
	//
	// See the guidelines above for deciding between `FailedPrecondition`,
	// `Aborted`, and `Unavailable`.
	//
	// HTTP Mapping: 503 Service Unavailable
	// Google API Mapping: Code.UNAVAILABLE
	Unavailable Code = 14
	// Unrecoverable data loss or corruption.
	//
	// HTTP Mapping: 500 Internal Server Error
	// Google API Mapping: Code.DATA_LOSS
	DataLoss Code = 15
	// The request does not have valid authentication credentials for the
	// operation.
	//
	// HTTP Mapping: 401 Unauthorized
	// Google API Mapping: Code.UNAUTHENTICATED
	Unauthenticated Code = 16
	// Application error. This will typically be accompianed by a user-defined
	// error name across the wire.
	//
	// HTTP Mapping: 500 Internal Server Error
	// Google API Mapping: Code.UNKNOWN
	Application Code = 17
)

var (
	codeToString = map[int]string{
		0:  "none",
		1:  "cancelled",
		2:  "unknown",
		3:  "invalid-argument",
		4:  "deadline-exceeded",
		5:  "not-found",
		6:  "already-exists",
		7:  "permission-denied",
		8:  "resource-exhausted",
		9:  "failed-precondition",
		10: "aborted",
		11: "out-of-range",
		12: "unimplemented",
		13: "internal",
		14: "unavailable",
		15: "data-loss",
		16: "unauthenticated",
		17: "application",
	}
	stringToCode = map[string]int{
		"none":                0,
		"cancelled":           1,
		"unknown":             2,
		"invalid-argument":    3,
		"deadline-exceeded":   4,
		"not-found":           5,
		"already-exists":      6,
		"permission-denied":   7,
		"resource-exhausted":  8,
		"failed-precondition": 9,
		"aborted":             10,
		"out-of-range":        11,
		"unimplemented":       12,
		"internal":            13,
		"unavailable":         14,
		"data-loss":           15,
		"unauthenticated":     16,
		"application":         17,
	}
)

// Code represents the type of error for an RPC call.
//
// Sometimes multiple error codes may apply. Services should return
// the most specific error code that applies. For example, prefer
// `OutOfRange` over `FailedPrecondition` if both codes apply.
// Similarly prefer `NotFound` or `AlreadyExists` over `FailedPrecondition`.
type Code int

// String returns the the string representation of the Code.
func (c Code) String() string {
	s, ok := codeToString[int(c)]
	if ok {
		return s
	}
	return strconv.Itoa(int(c))
}
