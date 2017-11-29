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

package yarpc_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/encoding/raw"
	"go.uber.org/yarpc/internal/testutils"
	pkgerrors "go.uber.org/yarpc/pkg/errors"
	"go.uber.org/zap"
)

func TestStartStopErrors(t *testing.T) {
	wantOutboundError := pkgerrors.NotRunningOutboundError("example-client")
	wantInboundError := pkgerrors.NotRunningInboundError("example")

	procedures := []transport.Procedure{
		raw.Procedure("echo", func(_ context.Context, data []byte) ([]byte, error) {
			return data, nil
		})[0],
		raw.OnewayProcedure("nop", func(context.Context, []byte) error {
			return nil
		})[0],
	}

	dispatcherConfig, err := testutils.NewDispatcherConfig("example")
	require.NoError(t, err)
	serverDispatcher, err := testutils.NewServerDispatcher(procedures, dispatcherConfig, zap.NewNop())
	require.NoError(t, err)
	clientDispatcher, err := testutils.NewClientDispatcher(testutils.TransportTypeHTTP, dispatcherConfig, zap.NewNop())
	require.NoError(t, err)

	client := raw.New(clientDispatcher.ClientConfig("example"))

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_, err = client.Call(ctx, "echo", []byte("hello"))
	require.Equal(t, wantOutboundError, err)
	_, err = client.CallOneway(ctx, "nop", []byte("hello"))
	require.Equal(t, wantOutboundError, err)

	require.NoError(t, serverDispatcher.Start())
	require.NoError(t, clientDispatcher.Start())

	ctx, cancel = context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	response, err := client.Call(ctx, "echo", []byte("hello"))
	require.NoError(t, err)
	require.Equal(t, "hello", string(response))
	_, err = client.CallOneway(ctx, "nop", []byte("hello"))
	require.NoError(t, err)

	require.NoError(t, serverDispatcher.Stop())

	ctx, cancel = context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_, err = client.Call(ctx, "echo", []byte("hello"))
	require.Equal(t, wantInboundError, err)
	// Inbound middleware is run on a goroutine for HTTP
	// Maybe the semantics will change in the future
	//_, err = client.CallOneway(ctx, "nop", []byte("hello"))
	//require.Equal(t, wantInboundError, err)

	require.NoError(t, clientDispatcher.Stop())

	ctx, cancel = context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_, err = client.Call(ctx, "echo", []byte("hello"))
	require.Equal(t, wantOutboundError, err)
	_, err = client.CallOneway(ctx, "nop", []byte("hello"))
	require.Equal(t, wantOutboundError, err)
}
