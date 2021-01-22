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
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
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
		func(t *testing.T) {
			status, ok := NamedErrorf("foo", "hello %d", 1).(*Status)
			require.True(t, ok)
			require.Equal(t, "code:unknown name:foo message:hello 1", status.Error())
		},
	)
}

func TestIsYARPCError(t *testing.T) {
	testAllErrorConstructors(
		t,
		func(t *testing.T, code Code, errorConstructor func(string, ...interface{}) error) {
			require.True(t, IsYARPCError(errorConstructor("")))
		},
		func(t *testing.T) {
			require.True(t, IsYARPCError(NamedErrorf("", "")))
		},
	)
}

func TestErrorCode(t *testing.T) {
	testAllErrorConstructors(
		t,
		func(t *testing.T, code Code, errorConstructor func(string, ...interface{}) error) {
			require.Equal(t, code, ErrorCode(errorConstructor("")))
		},
		func(t *testing.T) {
			require.Equal(t, CodeUnknown, ErrorCode(NamedErrorf("", "")))
		},
	)
}

func TestErrorName(t *testing.T) {
	testAllErrorConstructors(
		t,
		func(t *testing.T, code Code, errorConstructor func(string, ...interface{}) error) {
			require.Empty(t, ErrorName(errorConstructor("")))
		},
		func(t *testing.T) {
			require.Equal(t, "foo", ErrorName(NamedErrorf("foo", "")))
		},
	)
}

func TestErrorMessage(t *testing.T) {
	testAllErrorConstructors(
		t,
		func(t *testing.T, code Code, errorConstructor func(string, ...interface{}) error) {
			require.Equal(t, "hello 1", ErrorMessage(errorConstructor("hello %d", 1)))
		},
		func(t *testing.T) {
			require.Equal(t, "hello 1", ErrorMessage(NamedErrorf("foo", "hello %d", 1)))
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

func TestNonYARPCErrors(t *testing.T) {
	assert.Equal(t, CodeOK, ErrorCode(nil))
	assert.Equal(t, CodeUnknown, ErrorCode(errors.New("")))
	assert.Equal(t, "", ErrorName(nil))
	assert.Equal(t, "", ErrorName(errors.New("")))
	assert.Equal(t, "", ErrorMessage(nil))
	assert.Equal(t, "", ErrorMessage(errors.New("")))
	assert.Nil(t, FromHeaders(CodeOK, "", ""))
}

func testAllErrorConstructors(
	t *testing.T,
	errorConstructorFunc func(*testing.T, Code, func(string, ...interface{}) error),
	namedFunc func(*testing.T),
) {
	for code, errorConstructor := range _codeToErrorConstructor {
		t.Run(code.String(), func(t *testing.T) {
			errorConstructorFunc(t, code, errorConstructor)
		})
	}
	t.Run("Named", namedFunc)
}

func TestErrUnwrap(t *testing.T) {
	myErr := errors.New("my custom error")
	yErr := AbortedErrorf("wrap my custom err: %w", myErr)

	assert.Equal(t, FromError(yErr).Message(), "wrap my custom err: my custom error", "unexpected message")
	assert.Equal(t, myErr, errors.Unwrap(yErr), "expected original error")
	assert.Equal(t, myErr, errors.Unwrap(FromError(myErr)), "expected original error")
	assert.True(t, errors.Is(yErr, myErr), "expected original error")
}

func TestErrUnwrapIs(t *testing.T) {
	t.Run("FromError", func(t *testing.T) {
		err := FromError(context.DeadlineExceeded)
		assert.True(t, errors.Is(err, context.DeadlineExceeded), "errors be errors, yo")
	})

	t.Run("DeadlineExceededErrorf", func(t *testing.T) {
		err := DeadlineExceededErrorf("Past due: %w", context.DeadlineExceeded)
		assert.True(t, errors.Is(err, context.DeadlineExceeded), "errors be errors, yo")
	})
}

func TestErrUnwrapNewf(t *testing.T) {
	t.Run("no format", func(t *testing.T) {
		err := Newf(CodeAborted, "not going to do it")
		assert.NoError(t, errors.Unwrap(err))
	})

	t.Run("formatted with v verb", func(t *testing.T) {
		origErr := errors.New("something broke")
		err := Newf(CodeAborted, "not going to do it: %v", origErr)
		assert.NoError(t, errors.Unwrap(err)) // %v hides the inner error
	})

	t.Run("wrapped with w verb", func(t *testing.T) {
		origErr := errors.New("something broke")
		err := Newf(CodeAborted, "not going to do it: %w", origErr)
		assert.Equal(t, origErr, errors.Unwrap(err))
	})
}

func TestErrUnwrapNil(t *testing.T) {
	assert.NotPanics(t, func() {
		var err *Status
		errors.Unwrap(err)
	})

	assert.NotPanics(t, func() {
		err := &Status{}
		errors.Unwrap(err)
	})
}

type customYARPCError struct {
	err string
}

func (e customYARPCError) Error() string {
	return e.err
}
func (e customYARPCError) YARPCError() *Status {
	return FromError(DataLossErrorf(e.err))
}

func TestFromError(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		assert.Nil(t, FromError(nil))
	})

	t.Run("unknown err", func(t *testing.T) {
		st := FromError(errors.New("foo"))
		assert.Equal(t, CodeUnknown.String(), st.Code().String(), "unexpected code")
	})

	t.Run("wrapped Status", func(t *testing.T) {
		wrappedErr := fmt.Errorf("wrap 2: %w",
			FailedPreconditionErrorf("wrap 1: %w", // yarpc error
				errors.New("inner")))

		st := FromError(wrappedErr)
		assert.Equal(t, CodeFailedPrecondition.String(), st.Code().String(), "unexpected Code")
		assert.Equal(t, "wrap 1: inner", st.Message())
	})

	t.Run("wrapped Status interface", func(t *testing.T) {
		st := FromError(fmt.Errorf("wrapped: %w", customYARPCError{err: "custom err"}))
		assert.Equal(t, CodeDataLoss.String(), st.Code().String(), "unexpected Code")
		assert.Equal(t, "custom err", st.Message())
	})
}

func TestIsStatus(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		assert.False(t, IsStatus(nil))
	})

	t.Run("unknown err", func(t *testing.T) {
		err := errors.New("foo")
		assert.False(t, IsStatus(err), "unexpected Status")
	})

	t.Run("wrapped Status", func(t *testing.T) {
		err := fmt.Errorf("wrap 2: %w",
			FailedPreconditionErrorf("wrap 1: %w", // yarpc error
				errors.New("inner")))

		assert.True(t, IsStatus(err), "expected YARPC error")
	})

	t.Run("wrapped Status interface", func(t *testing.T) {
		err := fmt.Errorf("wrapped: %w", customYARPCError{err: "custom err"})
		assert.True(t, IsStatus(err))
	})
}

func TestErrorWithFmtVerbs(t *testing.T) {
	err := errors.New(`http://foo%s: invalid URL escape "%s"`)
	assert.EqualError(t, UnknownErrorf(err.Error()), FromError(err).Error())
}

func TestWrapError(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		var we *wrapError
		assert.Empty(t, we.Error())
		assert.NoError(t, errors.Unwrap(we))
	})

	t.Run("empty", func(t *testing.T) {
		we := &wrapError{}
		assert.Empty(t, we.Error())
		assert.NoError(t, errors.Unwrap(we))
	})

	t.Run("full", func(t *testing.T) {
		inner := errors.New("i'm a little error")
		we := &wrapError{err: inner}
		assert.Equal(t, inner.Error(), we.Error())
		assert.Equal(t, inner, errors.Unwrap(we))
	})
}
