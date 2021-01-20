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

package main_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	tchannel "github.com/uber/tchannel-go"
	tutils "github.com/uber/tchannel-go/testutils"
	"go.uber.org/yarpc/api/transport"
	wc "go.uber.org/yarpc/encoding/thrift/thriftrw-plugin-yarpc/internal/tests/weather/weatherclient"
	ytchannel "go.uber.org/yarpc/transport/tchannel"
	"go.uber.org/yarpc/yarpcerrors"
	"golang.org/x/net/context"
)

func TestSanitization(t *testing.T) {
	copts := tutils.NewOpts().DisableLogVerification()
	copts.DisableRelay = true

	var handlerWasCalled bool
	copts.Handler = tchannel.HandlerFunc(func(ctx context.Context, call *tchannel.InboundCall) {
		headered := ctx.(tchannel.ContextWithHeaders)
		assert.Len(t, headered.Headers(), 0)
		handlerWasCalled = true
		call.Response().SendSystemError(
			tchannel.NewSystemError(tchannel.ErrCodeBadRequest, "infinite sadness"),
		)
	})

	server := tutils.NewTestServer(t, copts)

	client, done := newWeatherClient(t, server.HostPort())
	defer done()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := client.Check(tchannel.WrapWithHeaders(ctx, map[string]string{"key": "value"}))
	require.Error(t, err, "expected Check to fail")
	assert.Equal(t, yarpcerrors.CodeInvalidArgument, yarpcerrors.FromError(err).Code(),
		"error code must match")
	assert.True(t, handlerWasCalled, "newTestServer handler was never called")
}

func newWeatherClient(t *testing.T, hostPort string) (_ wc.Interface, done func()) {
	trans, err := ytchannel.NewTransport(ytchannel.ServiceName(tutils.DefaultClientName))
	require.NoError(t, err)
	require.NoError(t, trans.Start(), "failed to start transport")

	outbound := trans.NewSingleOutbound(hostPort)
	require.NoError(t, outbound.Start(), "failed to start outbound")

	cc := testClientConfig{outbound: outbound}
	return wc.New(cc), func() {
		assert.NoError(t, outbound.Stop(), "failed to stop outbound")
		assert.NoError(t, trans.Stop(), "failed to stop transport")
	}
}

type testClientConfig struct {
	outbound transport.UnaryOutbound
}

func (cc testClientConfig) Caller() string {
	return tutils.DefaultClientName
}

func (cc testClientConfig) Service() string {
	return tutils.DefaultServerName
}

func (cc testClientConfig) GetUnaryOutbound() transport.UnaryOutbound {
	return cc.outbound
}

func (cc testClientConfig) GetOnewayOutbound() transport.OnewayOutbound {
	panic("Not implemented")
}
