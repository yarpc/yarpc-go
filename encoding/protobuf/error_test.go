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

	"github.com/gogo/googleapis/google/rpc"
	"github.com/gogo/protobuf/proto"
	"github.com/gogo/protobuf/types"
	"github.com/gogo/status"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc/api/transport/transporttest"
	"go.uber.org/yarpc/yarpcerrors"
	"google.golang.org/grpc/codes"
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
		resw.ApplicationErrorMeta.Details,
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

type yarpcError interface{ YARPCError() *yarpcerrors.Status }

func TestPbErrorToYARPCError(t *testing.T) {
	tests := []struct {
		name             string
		code             yarpcerrors.Code
		message          string
		details          []proto.Message
		expectedGRPCCode codes.Code
	}{
		{
			name:             "pbError without details",
			code:             yarpcerrors.CodeAborted,
			message:          "simple test",
			expectedGRPCCode: codes.Aborted,
		},
		{
			name:             "pbError with single detail",
			code:             yarpcerrors.CodeInternal,
			message:          "internal error",
			expectedGRPCCode: codes.Internal,
			details: []proto.Message{
				&types.StringValue{Value: "test value"},
			},
		},
		{
			name:             "pbError with multiple details",
			code:             yarpcerrors.CodeNotFound,
			message:          "not found error",
			expectedGRPCCode: codes.NotFound,
			details: []proto.Message{
				&types.StringValue{Value: "test value"},
				&types.Int32Value{Value: 45},
				&types.Any{Value: []byte{1, 2, 3, 4, 5}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var errOpts []ErrorOption
			if len(tt.details) > 0 {
				errOpts = append(errOpts, WithErrorDetails(tt.details...))
			}
			pberror := NewError(tt.code, tt.message, errOpts...)
			st := pberror.(yarpcError).YARPCError()
			assert.Equal(t, st.Code(), tt.code)
			assert.Equal(t, st.Message(), tt.message)

			statusPb := rpc.Status{}
			proto.Unmarshal(st.Details(), &statusPb)
			status := status.FromProto(&statusPb)

			assert.Equal(t, tt.expectedGRPCCode, status.Code(), "unexpected grpc status code")
			assert.Equal(t, tt.message, status.Message(), "unexpected grpc status message")
			for i, detail := range tt.details {
				assert.Equal(t, detail, status.Details()[i])
			}
		})
	}
}

func TestPbErrorToYARPCErrorWithNonProtoDetail(t *testing.T) {
	pberr := pberror{
		code:    yarpcerrors.CodeAborted,
		message: "test wrong proto",
		details: []interface{}{pberror{}},
	}
	err := pberr.YARPCError()
	assert.Equal(t, yarpcerrors.CodeUnknown, err.Code())
	assert.Equal(t, "proto error detail is not proto.Message compatible", err.Message())
}
