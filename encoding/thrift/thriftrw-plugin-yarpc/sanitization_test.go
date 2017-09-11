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

package main_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	tchannel "github.com/uber/tchannel-go"
	"github.com/uber/tchannel-go/raw"
	tutils "github.com/uber/tchannel-go/testutils"
	"go.uber.org/yarpc/api/transport"
	wc "go.uber.org/yarpc/encoding/thrift/thriftrw-plugin-yarpc/internal/tests/weather/weatherclient"
	ttrans "go.uber.org/yarpc/transport/tchannel"
)

func TestSanitization(t *testing.T) {
	server := newTestServer(t)
	clientConf := newTestClientConfig(server.HostPort(), t)
	badCtx, cancel := newBadContext()
	defer cancel()

	client := wc.New(clientConf)
	_, err := client.Check(badCtx)
	assert.NoError(t, err)
}

func newTestServer(t *testing.T) *tutils.TestServer {
	copts := tutils.NewOpts().DisableLogVerification()
	server := tutils.NewTestServer(t, copts)
	server.RegisterFunc("check", func(ctx context.Context, args *raw.Args) (*raw.Res, error) {
		headered := tchannel.Wrap(ctx)
		assert.Len(t, headered.Headers(), 0)
		return nil, nil
	})
	return server
}

func newBadContext() (context.Context, func()) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	badCtx := tchannel.WrapWithHeaders(ctx, map[string]string{"key": "value"})
	return badCtx, cancel
}

func newTestClientConfig(hostPort string, t *testing.T) transport.ClientConfig {
	trans, err := ttrans.NewTransport()
	assert.NoError(t, err)
	outbound := trans.NewSingleOutbound(hostPort)
	return testClientConfig{
		outbound: outbound,
	}
}

type testClientConfig struct {
	outbound transport.Outbound
}

func (cc testClientConfig) Caller() string {
	return "testcaller"
}

func (cc testClientConfig) Service() string {
	return "testservice"
}

func (cc testClientConfig) GetUnaryOutbound() transport.UnaryOutbound {
	return cc.outbound.(transport.UnaryOutbound)
}

func (cc testClientConfig) GetOnewayOutbound() transport.OnewayOutbound {
	return nil
}
