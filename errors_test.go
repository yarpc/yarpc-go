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
)

func TestErrorsString(t *testing.T) {
	for code, errorConstructor := range _codeToErrorConstructor {
		t.Run(code.String(), func(t *testing.T) {
			yarpcError, ok := errorConstructor("hello %d", 1).(*yarpcError)
			require.True(t, ok)
			require.Equal(t, fmt.Sprintf("code:%s message:hello 1", code.String()), yarpcError.Error())
		})
	}
	t.Run("Named", func(t *testing.T) {
		yarpcError, ok := NamedErrorf("foo", "hello %d", 1).(*yarpcError)
		require.True(t, ok)
		require.Equal(t, "code:unknown name:foo message:hello 1", yarpcError.Error())
	})
}
