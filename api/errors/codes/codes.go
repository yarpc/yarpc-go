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

import (
	"fmt"
	"strconv"
	"strings"
)

const (
	// None means no error; returned on success
	//
	// HTTP Mapping: 200 OK
	// Google API Mapping: Code.OK
	None Code = 0

	// Cancelled means the operation was cancelled, typically by the caller.
	//
	// HTTP Mapping: 499 Client Closed Request
	// Google API Mapping: Code.CANCELLED
	Cancelled Code = 1

	// Unknown means an unknown error. For example, this error may be returned when
	// a `Status` value received from another address space belongs to
	// an error space that is not known in this address space. Also
	// errors raised by APIs that do not return enough error information
	// may be converted to this error.
	//
	// HTTP Mapping: 500 Internal Server Error
	// Google API Mapping: Code.UNKNOWN
	Unknown Code = 2

	// InvalidArgument means the client specified an invalid argument. Note that this differs
	// from `FailedPrecondition`. `InvalidArgument` indicates arguments
	// that are problematic regardless of the state of the system
	// (e.g., a malformed file name).
	//
	// HTTP Mapping: 400 Bad Request
	// Google API Mapping: Code.INVALID_ARGUMENT
	InvalidArgument Code = 3

	// DeadlineExceeded means the deadline expired before the operation could
	// complete. For operations that change the state of the system, this error
	// may be returned even if the operation has completed successfully. For example,
	// a successful response from a server could have been delayed long
	// enough for the deadline to expire.
	//
	// HTTP Mapping: 504 Gateway Timeout
	// Google API Mapping: Code.DEADLINE_EXCEEDED
	DeadlineExceeded Code = 4

	// NotFound means some requested entity (e.g., file or directory) was not found.
	// For privacy reasons, this code *may* be returned when the client
	// does not have the access rights to the entity, though such usage is
	// discouraged.
	//
	// HTTP Mapping: 404 Not Found
	// Google API Mapping: Code.NOT_FOUND
	NotFound Code = 5

	// AlreadyExists means the entity that a client attempted to create
	// (e.g., file or directory) already exists.
	//
	// HTTP Mapping: 409 Conflict
	// Google API Mapping: Code.ALREADY_EXISTS
	AlreadyExists Code = 6

	// PermissionDenied means the caller does not have permission to execute
	// the specified operation. `PermissionDenied` must not be used for rejections
	// caused by exhausting some resource (use `ResourceExhausted`
	// instead for those errors). `PermissionDenied` must not be
	// used if the caller can not be identified (use `Unauthenticated`
	// instead for those errors).
	//
	// HTTP Mapping: 403 Forbidden
	// Google API Mapping: Code.PERMISSION_DENIED
	PermissionDenied Code = 7

	// ResourceExhausted means some resource has been exhausted, perhaps a per-user
	// quota, or perhaps the entire file system is out of space.
	//
	// HTTP Mapping: 429 Too Many Requests
	// Google API Mapping: Code.RESOURCE_EXHAUSTED
	ResourceExhausted Code = 8

	// FailedPrecondition means the operation was rejected because the system is not
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
	//
	// HTTP Mapping: 400 Bad Request
	// Google API Mapping: Code.FAILED_PRECONDITION
	FailedPrecondition Code = 9

	// Aborted means the operation was aborted, typically due to a concurrency issue
	// such as a sequencer check failure or transaction abort.
	//
	// See the guidelines above for deciding between `FailedPrecondition`,
	// `Aborted`, and `Unavailable`.
	//
	// HTTP Mapping: 409 Conflict
	// Google API Mapping: Code.ABORTED
	Aborted Code = 10

	// OutOfRange means the operation was attempted past the valid range.
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
	//
	// HTTP Mapping: 400 Bad Request
	// Google API Mapping: Code.OUT_OF_RANGE
	OutOfRange Code = 11

	// Unimplemented means the operation is not implemented or is not
	// supported/enabled in this service.
	//
	// HTTP Mapping: 501 Not Implemented
	// Google API Mapping: Code.UNIMPLEMENTED
	Unimplemented Code = 12

	// Internal means an internal error. This means that some invariants expected
	// by the underlying system have been broken. This error code is reserved
	// for serious errors.
	//
	// HTTP Mapping: 500 Internal Server Error
	// Google API Mapping: Code.INTERNAL
	Internal Code = 13

	// Unavailable means the service is currently unavailable. This is most likely a
	// transient condition, which can be corrected by retrying with
	// a backoff.
	//
	// See the guidelines above for deciding between `FailedPrecondition`,
	// `Aborted`, and `Unavailable`.
	//
	// HTTP Mapping: 503 Service Unavailable
	// Google API Mapping: Code.UNAVAILABLE
	Unavailable Code = 14

	// DataLoss means unrecoverable data loss or corruption.
	//
	// HTTP Mapping: 500 Internal Server Error
	// Google API Mapping: Code.DATA_LOSS
	DataLoss Code = 15

	// Unauthenticated means the request does not have valid authentication
	// credentials for the operation.
	//
	// HTTP Mapping: 401 Unauthorized
	// Google API Mapping: Code.UNAUTHENTICATED
	Unauthenticated Code = 16

	// Application means there was an application error. This will typically be
	// accompianed by a user-defined error name across the wire.
	//
	// HTTP Mapping: 500 Internal Server Error
	// Google API Mapping: Code.UNKNOWN
	Application Code = 17
)

var (
	codeToString = map[Code]string{
		None:               "none",
		Cancelled:          "cancelled",
		Unknown:            "unknown",
		InvalidArgument:    "invalid-argument",
		DeadlineExceeded:   "deadline-exceeded",
		NotFound:           "not-found",
		AlreadyExists:      "already-exists",
		PermissionDenied:   "permission-denied",
		ResourceExhausted:  "resource-exhausted",
		FailedPrecondition: "failed-precondition",
		Aborted:            "aborted",
		OutOfRange:         "out-of-range",
		Unimplemented:      "unimplemented",
		Internal:           "internal",
		Unavailable:        "unavailable",
		DataLoss:           "data-loss",
		Unauthenticated:    "unauthenticated",
		Application:        "application",
	}
	stringToCode = map[string]Code{
		"none":                None,
		"cancelled":           Cancelled,
		"unknown":             Unknown,
		"invalid-argument":    InvalidArgument,
		"deadline-exceeded":   DeadlineExceeded,
		"not-found":           NotFound,
		"already-exists":      AlreadyExists,
		"permission-denied":   PermissionDenied,
		"resource-exhausted":  ResourceExhausted,
		"failed-precondition": FailedPrecondition,
		"aborted":             Aborted,
		"out-of-range":        OutOfRange,
		"unimplemented":       Unimplemented,
		"internal":            Internal,
		"unavailable":         Unavailable,
		"data-loss":           DataLoss,
		"unauthenticated":     Unauthenticated,
		"application":         Application,
	}
	codeToHTTPStatusCode = map[Code]int{
		None:               200,
		Cancelled:          499,
		Unknown:            500,
		InvalidArgument:    400,
		DeadlineExceeded:   504,
		NotFound:           404,
		AlreadyExists:      409,
		PermissionDenied:   403,
		ResourceExhausted:  429,
		FailedPrecondition: 400,
		Aborted:            409,
		OutOfRange:         400,
		Unimplemented:      501,
		Internal:           500,
		Unavailable:        503,
		DataLoss:           500,
		Unauthenticated:    401,
		Application:        500,
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
	s, ok := codeToString[c]
	if ok {
		return s
	}
	return strconv.Itoa(int(c))
}

// MarshalText implements encoding.TextMarshaler.
func (c Code) MarshalText() ([]byte, error) {
	s, ok := codeToString[c]
	if ok {
		return []byte(s), nil
	}
	return nil, fmt.Errorf("unknown code: %d", int(c))
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (c *Code) UnmarshalText(text []byte) error {
	i, ok := stringToCode[strings.ToLower(string(text))]
	if !ok {
		return fmt.Errorf("unknown code string: %s", string(text))
	}
	*c = i
	return nil
}

// MarshalJSON implements json.Marshaler.
func (c Code) MarshalJSON() ([]byte, error) {
	s, ok := codeToString[c]
	if ok {
		return []byte(`"` + s + `"`), nil
	}
	return nil, fmt.Errorf("unknown code: %d", int(c))
}

// UnmarshalJSON implements json.Unmarshaler.
func (c *Code) UnmarshalJSON(text []byte) error {
	s := string(text)
	if len(s) < 3 || s[0] != '"' || s[len(s)-1] != '"' {
		return fmt.Errorf("invalid code string: %s", s)
	}
	i, ok := stringToCode[strings.ToLower(s[1:len(s)-1])]
	if !ok {
		return fmt.Errorf("unknown code string: %s", s)
	}
	*c = i
	return nil
}

// HTTPStatusCode returns the HTTP status code for the given Code.
func (c Code) HTTPStatusCode() (int, error) {
	s, ok := codeToHTTPStatusCode[c]
	if !ok {
		return 0, fmt.Errorf("unknown code: %d", int(c))
	}
	return s, nil
}
