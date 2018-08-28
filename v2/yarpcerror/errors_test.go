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

package yarpcerror

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

var (
	_codeToErrorConstructor = map[Code]func(string, ...interface{}) error{
		CodeCancelled:          CancelledErrorf,
		CodeUnknown:            UnknownErrorf,
		CodeInvalidArgument:    InvalidArgumentErrorf,
		CodeDeadlineExceeded:   DeadlineExceededErrorf,
		CodeNotFound:           NotFoundErrorf,
		CodeAlreadyExists:      AlreadyExistsErrorf,
		CodePermissionDenied:   PermissionDeniedErrorf,
		CodeResourceExhausted:  ResourceExhaustedErrorf,
		CodeFailedPrecondition: FailedPreconditionErrorf,
		CodeAborted:            AbortedErrorf,
		CodeOutOfRange:         OutOfRangeErrorf,
		CodeUnimplemented:      UnimplementedErrorf,
		CodeInternal:           InternalErrorf,
		CodeUnavailable:        UnavailableErrorf,
		CodeDataLoss:           DataLossErrorf,
		CodeUnauthenticated:    UnauthenticatedErrorf,
	}
	_codeToIsErrorWithCode = map[Code]func(error) bool{
		CodeCancelled:          IsCancelled,
		CodeUnknown:            IsUnknown,
		CodeInvalidArgument:    IsInvalidArgument,
		CodeDeadlineExceeded:   IsDeadlineExceeded,
		CodeNotFound:           IsNotFound,
		CodeAlreadyExists:      IsAlreadyExists,
		CodePermissionDenied:   IsPermissionDenied,
		CodeResourceExhausted:  IsResourceExhausted,
		CodeFailedPrecondition: IsFailedPrecondition,
		CodeAborted:            IsAborted,
		CodeOutOfRange:         IsOutOfRange,
		CodeUnimplemented:      IsUnimplemented,
		CodeInternal:           IsInternal,
		CodeUnavailable:        IsUnavailable,
		CodeDataLoss:           IsDataLoss,
		CodeUnauthenticated:    IsUnauthenticated,
	}
)

func TestErrorsString(t *testing.T) {
	testAllErrorConstructors(
		t,
		func(t *testing.T, code Code, errorConstructor func(string, ...interface{}) error) {
			status, ok := errorConstructor("hello %d", 1).(*Status)
			require.True(t, ok)
			require.Equal(t, fmt.Sprintf("code:%s message:hello 1", code.String()), status.Error())
		},
	)
}

func TestIsErrorWithCode(t *testing.T) {
	for code, errorConstructor := range _codeToErrorConstructor {
		t.Run(code.String(), func(t *testing.T) {
			isErrorWithCode, ok := _codeToIsErrorWithCode[code]
			require.True(t, ok)
			require.True(t, isErrorWithCode(errorConstructor("")))
		})
	}
}

func testAllErrorConstructors(
	t *testing.T,
	errorConstructorFunc func(*testing.T, Code, func(string, ...interface{}) error),
) {
	for code, errorConstructor := range _codeToErrorConstructor {
		t.Run(code.String(), func(t *testing.T) {
			errorConstructorFunc(t, code, errorConstructor)
		})
	}
}
