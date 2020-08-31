// Copyright (c) 2020 Uber Technologies, Inc.
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

package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/thriftrw/compile"
)

func TestGetYARPCErrorCode(t *testing.T) {
	const exName = "MyException"

	t.Run("success", func(t *testing.T) {
		spec := &compile.StructSpec{
			Name:        exName,
			Annotations: map[string]string{_errorCodeAnnotationKey: "ABORTED"},
		}
		assert.NotPanics(t, func() {
			want := "yarpcerrors.CodeAborted"
			got := getYARPCErrorCode(spec)
			assert.Equal(t, want, got)
		}, "unexpected panic")
	})

	t.Run("panic fail", func(t *testing.T) {
		spec := &compile.StructSpec{
			Name:        exName,
			Annotations: map[string]string{_errorCodeAnnotationKey: "foo"},
		}
		assert.PanicsWithValue(t,
			"invalid rpc.code annotation for \"MyException\": \"foo\"\nAvailable codes: CANCELLED,UNKNOWN,INVALID_ARGUMENT,DEADLINE_EXCEEDED,NOT_FOUND,ALREADY_EXISTS,PERMISSION_DENIED,RESOURCE_EXHAUSTED,FAILED_PRECONDITION,ABORTED,OUT_OF_RANGE,UNIMPLEMENTED,INTERNAL,UNAVAILABLE,DATA_LOSS,UNAUTHENTICATED",
			func() { getYARPCErrorCode(spec) },
			"unexpected panic")
	})
}
