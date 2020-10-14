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

package tchannel_test

import (
	"bytes"
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uber/tchannel-go"
	"go.uber.org/yarpc/api/transport"
	ytchannel "go.uber.org/yarpc/transport/tchannel"
	"go.uber.org/yarpc/yarpcerrors"
)

func TestResponseErrorMetaIntegration(t *testing.T) {
	const networkErrMsg = "a network error!"

	// use vanilla TChannel server to force return a system error
	tchHandler := func(ctx context.Context, call *tchannel.InboundCall) {
		networkErr := tchannel.NewSystemError(tchannel.ErrCodeNetwork, networkErrMsg)
		require.NoError(t, call.Response().SendSystemError(networkErr), "failed to send system error")
	}
	server, err := tchannel.NewChannel("test", &tchannel.ChannelOptions{
		Handler: tchannel.HandlerFunc(tchHandler),
	})
	require.NoError(t, err, "could not create TChannel channel")

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err, "failed to create listener")
	server.Serve(listener)
	defer server.Close()

	// init client
	clientTransport, err := ytchannel.NewTransport(ytchannel.ServiceName("foo"))
	require.NoError(t, err, "failed to create TChannel client transport")
	require.NoError(t, clientTransport.Start(), "could not start client transport")
	defer func() { assert.NoError(t, clientTransport.Stop(), "did not cleanly shutdown client transport") }()

	client := clientTransport.NewSingleOutbound(listener.Addr().String())
	require.NoError(t, client.Start(), "could not start outbound")
	defer func() { assert.NoError(t, client.Stop(), "did not cleanly shutdown outbound") }()

	// call server
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_, err = client.Call(ctx, &transport.Request{
		Service: "foo",
		Body:    bytes.NewReader([]byte("bar")),
	})
	require.Error(t, err, "expected call failure")

	// ensure this is still a `yarpcerrors.Status` error
	require.True(t, yarpcerrors.IsStatus(err), "expected YARPC status error")
	require.Equal(t, networkErrMsg, yarpcerrors.FromError(err).Message(), "unexpected 'yarpcerrors' error message")
	require.Equal(t, networkErrMsg, errors.Unwrap(err).Error(), "unexpected unwrapped error message")
	assert.IsType(t, &yarpcerrors.Status{}, err, "expected YARPC status type") // ensure type switching still works

	// verify response meta
	meta := ytchannel.GetResponseErrorMeta(err)
	require.NotNil(t, meta, "unable to retrieve response meta")
	assert.Equal(t, tchannel.ErrCodeNetwork.String(), meta.Code.String(), "unexpected response code")
}
