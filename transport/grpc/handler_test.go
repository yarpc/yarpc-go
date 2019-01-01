// Copyright (c) 2019 Uber Technologies, Inc.
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
	"fmt"
	"net"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"
)

func TestInvalidStreamContext(t *testing.T) {
	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:0"))
	require.NoError(t, err)

	tran := NewTransport()
	i := tran.NewInbound(listener)

	h := handler{i: i}

	_, err = h.getBasicTransportRequest(context.Background(), "serv/proc")

	require.Contains(t, err.Error(), "cannot get metadata from ctx:")
	require.Contains(t, err.Error(), "code:internal")
}

func TestInvalidStreamMethod(t *testing.T) {
	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:0"))
	require.NoError(t, err)

	tran := NewTransport()
	i := tran.NewInbound(listener)

	h := handler{i: i}
	ctx := metadata.NewIncomingContext(context.Background(), metadata.MD{})

	_, err = h.getBasicTransportRequest(ctx, "invalidMethod!")

	require.Contains(t, err.Error(), errInvalidGRPCMethod.Error())
}

func TestInvalidStreamRequest(t *testing.T) {
	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:0"))
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
	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:0"))
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
	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:0"))
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
	require.Contains(t, err.Error(), "header has more than one value: rpc-caller")
}
