// Copyright (c) 2021 Uber Technologies, Inc.
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

import (
	"fmt"
	"strconv"
	"strings"
)

const (
	// CodeOK means no error; returned on success
	CodeOK Code = 0

	// CodeCancelled means the operation was cancelled, typically by the caller.
	CodeCancelled Code = 1

	// CodeUnknown means an unknown error. Errors raised by APIs
	// that do not return enough error information
	// may be converted to this error.
	CodeUnknown Code = 2

	// CodeInvalidArgument means the client specified an invalid argument.
	// Note that this differs from `FailedPrecondition`. `InvalidArgument`
	// indicates arguments that are problematic regardless of the state of
	// the system (e.g., a malformed file name).
	CodeInvalidArgument Code = 3

	// CodeDeadlineExceeded means the deadline expired before the operation could
	// complete. For operations that change the state of the system, this error
	// may be returned even if the operation has completed successfully. For example,
	// a successful response from a server could have been delayed long
	// enough for the deadline to expire.
	CodeDeadlineExceeded Code = 4

	// CodeNotFound means some requested entity (e.g., file or directory) was not found.
	// For privacy reasons, this code *may* be returned when the client
	// does not have the access rights to the entity, though such usage is
	// discouraged.
	CodeNotFound Code = 5

	// CodeAlreadyExists means the entity that a client attempted to create
	// (e.g., file or directory) already exists.
	CodeAlreadyExists Code = 6

	// CodePermissionDenied means the caller does not have permission to execute
	// the specified operation. `PermissionDenied` must not be used for rejections
	// caused by exhausting some resource (use `ResourceExhausted`
	// instead for those errors). `PermissionDenied` must not be
	// used if the caller can not be identified (use `Unauthenticated`
	// instead for those errors).
	CodePermissionDenied Code = 7

	// CodeResourceExhausted means some resource has been exhausted, perhaps a per-user
	// quota, or perhaps the entire file system is out of space.
	CodeResourceExhausted Code = 8

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
	CodeFailedPrecondition Code = 9

	// CodeAborted means the operation was aborted, typically due to a concurrency issue
	// such as a sequencer check failure or transaction abort.
	//
	// See the guidelines above for deciding between `FailedPrecondition`,
	// `Aborted`, and `Unavailable`.
	CodeAborted Code = 10

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
	CodeOutOfRange Code = 11

	// CodeUnimplemented means the operation is not implemented or is not
	// supported/enabled in this service.
	CodeUnimplemented Code = 12

	// CodeInternal means an internal error. This means that some invariants expected
	// by the underlying system have been broken. This error code is reserved
	// for serious errors.
	CodeInternal Code = 13

	// CodeUnavailable means the service is currently unavailable. This is most likely a
	// transient condition, which can be corrected by retrying with a backoff.
	//
	// See the guidelines above for deciding between `FailedPrecondition`,
	// `Aborted`, and `Unavailable`.
	CodeUnavailable Code = 14

	// CodeDataLoss means unrecoverable data loss or corruption.
	CodeDataLoss Code = 15

	// CodeUnauthenticated means the request does not have valid authentication
	// credentials for the operation.
	CodeUnauthenticated Code = 16
)

var (
	_codeToString = map[Code]string{
		CodeOK:                 "ok",
		CodeCancelled:          "cancelled",
		CodeUnknown:            "unknown",
		CodeInvalidArgument:    "invalid-argument",
		CodeDeadlineExceeded:   "deadline-exceeded",
		CodeNotFound:           "not-found",
		CodeAlreadyExists:      "already-exists",
		CodePermissionDenied:   "permission-denied",
		CodeResourceExhausted:  "resource-exhausted",
		CodeFailedPrecondition: "failed-precondition",
		CodeAborted:            "aborted",
		CodeOutOfRange:         "out-of-range",
		CodeUnimplemented:      "unimplemented",
		CodeInternal:           "internal",
		CodeUnavailable:        "unavailable",
		CodeDataLoss:           "data-loss",
		CodeUnauthenticated:    "unauthenticated",
	}
	_stringToCode = map[string]Code{
		"ok":                  CodeOK,
		"cancelled":           CodeCancelled,
		"unknown":             CodeUnknown,
		"invalid-argument":    CodeInvalidArgument,
		"deadline-exceeded":   CodeDeadlineExceeded,
		"not-found":           CodeNotFound,
		"already-exists":      CodeAlreadyExists,
		"permission-denied":   CodePermissionDenied,
		"resource-exhausted":  CodeResourceExhausted,
		"failed-precondition": CodeFailedPrecondition,
		"aborted":             CodeAborted,
		"out-of-range":        CodeOutOfRange,
		"unimplemented":       CodeUnimplemented,
		"internal":            CodeInternal,
		"unavailable":         CodeUnavailable,
		"data-loss":           CodeDataLoss,
		"unauthenticated":     CodeUnauthenticated,
	}
)

// Code represents the type of error for an RPC call.
//
// Sometimes multiple error codes may apply. Services should return
// the most specific error code that applies. For example, prefer
// `OutOfRange` over `FailedPrecondition` if both codes apply.
// Similarly prefer `NotFound` or `AlreadyExists` over `FailedPrecondition`.
//
// These codes are meant to match gRPC status codes.
// https://godoc.org/google.golang.org/grpc/codes#Code
type Code int

// String returns the the string representation of the Code.
func (c Code) String() string {
	s, ok := _codeToString[c]
	if ok {
		return s
	}
	return strconv.Itoa(int(c))
}

// MarshalText implements encoding.TextMarshaler.
func (c Code) MarshalText() ([]byte, error) {
	s, ok := _codeToString[c]
	if ok {
		return []byte(s), nil
	}
	return nil, fmt.Errorf("unknown code: %d", int(c))
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (c *Code) UnmarshalText(text []byte) error {
	i, ok := _stringToCode[strings.ToLower(string(text))]
	if !ok {
		return fmt.Errorf("unknown code string: %s", string(text))
	}
	*c = i
	return nil
}

// MarshalJSON implements json.Marshaler.
func (c Code) MarshalJSON() ([]byte, error) {
	s, ok := _codeToString[c]
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
	i, ok := _stringToCode[strings.ToLower(s[1:len(s)-1])]
	if !ok {
		return fmt.Errorf("unknown code string: %s", s)
	}
	*c = i
	return nil
}
