// Copyright (c) 2024 Uber Technologies, Inc.
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

package grpc

import (
	"context"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc/yarpcerrors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestInvalidStreamContext(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	tran := NewTransport()
	i := tran.NewInbound(listener)

	h := handler{i: i}

	_, err = h.getBasicTransportRequest(context.Background(), "serv/proc")

	require.Contains(t, err.Error(), "cannot get metadata from ctx:")
	require.Contains(t, err.Error(), "code:internal")
}

func TestInvalidStreamMethod(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	tran := NewTransport()
	i := tran.NewInbound(listener)

	h := handler{i: i}
	ctx := metadata.NewIncomingContext(context.Background(), metadata.MD{})

	_, err = h.getBasicTransportRequest(ctx, "invalidMethod!")

	require.Contains(t, err.Error(), errInvalidGRPCMethod.Error())
}

func TestInvalidStreamRequest(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	tran := NewTransport()
	i := tran.NewInbound(listener)

	h := handler{i: i}
	ctx := metadata.NewIncomingContext(context.Background(), metadata.MD{})

	_, err = h.getBasicTransportRequest(ctx, "service/proc")

	require.Contains(t, err.Error(), "code:invalid-argument")
	require.Contains(t, err.Error(), "missing service name, caller name, encoding")
}

func TestInvalidStreamEmptyHeader(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	tran := NewTransport()
	i := tran.NewInbound(listener)

	h := handler{i: i}
	md := metadata.MD{
		CallerHeader:   []string{},
		ServiceHeader:  []string{"test"},
		EncodingHeader: []string{"raw"},
	}
	ctx := metadata.NewIncomingContext(context.Background(), md)

	_, err = h.getBasicTransportRequest(ctx, "service/proc")

	require.Contains(t, err.Error(), "code:invalid-argument")
	require.Contains(t, err.Error(), "missing caller name")
}

func TestInvalidStreamMultipleHeaders(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	tran := NewTransport()
	i := tran.NewInbound(listener)

	h := handler{i: i}
	md := metadata.MD{
		CallerHeader: []string{"caller1", "caller2"},
	}
	ctx := metadata.NewIncomingContext(context.Background(), md)

	_, err = h.getBasicTransportRequest(ctx, "service/proc")

	require.Contains(t, err.Error(), "code:invalid-argument")
	require.Contains(t, err.Error(), "header has more than one value: rpc-caller:[caller1 caller2]")
}

func TestToGRPCError(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		assert.Nil(t, toGRPCError(nil))
	})

	t.Run("gRPC status", func(t *testing.T) {
		grpcSt := status.New(codes.InvalidArgument, "foo").Err()
		assert.Equal(t, grpcSt, toGRPCError(grpcSt), "expected same error given")
	})

	t.Run("yarpcerror", func(t *testing.T) {
		msg := "foo"
		yErr := yarpcerrors.FailedPreconditionErrorf(msg)

		grpcSt, ok := status.FromError(toGRPCError(yErr))
		require.True(t, ok, "expected gRPC error")

		assert.Equal(t, codes.FailedPrecondition, grpcSt.Code(), "code")
		assert.Equal(t, msg, grpcSt.Message(), "message mismatch")
	})
}
