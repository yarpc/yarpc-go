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

package protobuf

import (
	"errors"
	"fmt"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/gogo/protobuf/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc/api/transport/transporttest"
	"go.uber.org/yarpc/yarpcerrors"
)

func TestNewOK(t *testing.T) {
	err := NewError(yarpcerrors.CodeOK, "okay")
	assert.Nil(t, err)

	assert.Equal(t, yarpcerrors.FromError(err).Code(), yarpcerrors.CodeOK)
	assert.Equal(t, yarpcerrors.FromError(err).Message(), "")
}

func TestNew(t *testing.T) {
	err := NewError(yarpcerrors.CodeNotFound, "unfounded accusation")
	assert.Equal(t, yarpcerrors.FromError(err).Code(), yarpcerrors.CodeNotFound)
	assert.Equal(t, yarpcerrors.FromError(err).Message(), "unfounded accusation")
	assert.Contains(t, err.Error(), "unfounded accusation")
}

func TestForeignError(t *testing.T) {
	err := errors.New("to err is go")
	assert.Equal(t, yarpcerrors.FromError(err).Code(), yarpcerrors.CodeUnknown)
	assert.Equal(t, yarpcerrors.FromError(err).Message(), "to err is go")
}

func TestConvertToYARPCErrorWithWrappedError(t *testing.T) {
	errDetail := &types.BytesValue{Value: []byte("err detail bytes")}

	pbErr := NewError(
		yarpcerrors.CodeAborted,
		"aborted",
		WithErrorDetails(errDetail))

	wrappedErr := fmt.Errorf("wrapped err 2: %w", fmt.Errorf("wrapped err 1: %w", pbErr))

	err := convertToYARPCError(Encoding, wrappedErr, &codec{}, nil /* resw */)
	require.True(t, yarpcerrors.IsStatus(err), "unexpected error")
	assert.Equal(t, yarpcerrors.FromError(err).Code(), yarpcerrors.CodeAborted, "unexpected err code")
	assert.Equal(t, yarpcerrors.FromError(err).Message(), "aborted", "unexpected error message")

	gotDetails := yarpcerrors.FromError(err).Details()
	assert.NotEmpty(t, gotDetails, "no details marshaled")
}

func TestConvertToYARPCErrorApplicationErrorMeta(t *testing.T) {
	errDetails := []proto.Message{
		&types.StringValue{Value: "detail message"},
		&types.Int32Value{Value: 42},
		&types.BytesValue{Value: []byte("detail bytes")},
	}

	pbErr := NewError(
		yarpcerrors.CodeAborted,
		"aborted",
		WithErrorDetails(errDetails...))

	resw := &transporttest.FakeResponseWriter{}
	err := convertToYARPCError(Encoding, pbErr, &codec{}, resw)
	require.Error(t, err)

	require.NotNil(t, resw.ApplicationErrorMeta)
	assert.Equal(t, "StringValue", resw.ApplicationErrorMeta.Name, "expected first error detail name")
	assert.Equal(t,
		"[]{ StringValue{value:\"detail message\" } , Int32Value{value:42 } , BytesValue{value:\"detail bytes\" } }",
		resw.ApplicationErrorMeta.Message,
		"unexpected string of error details")
	assert.Nil(t, resw.ApplicationErrorMeta.Code, "code should nil")
}

func TestMessageNameWithoutPackage(t *testing.T) {
	tests := []struct {
		name string
		give string
		want string
	}{
		{
			name: "fqn",
			give: "uber.foo.bar.baz.MessageName",
			want: "MessageName",
		},
		{
			name: "not fully qualified",
			give: "MyMessage",
			want: "MyMessage",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, messageNameWithoutPackage(tt.give), "unexpected trim")
		})
	}
}
