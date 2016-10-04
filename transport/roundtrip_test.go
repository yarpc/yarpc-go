// Copyright (c) 2016 Uber Technologies, Inc.
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

package transport_test

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"testing"
	"time"

	"golang.org/x/net/context"

	"go.uber.org/yarpc/encoding/raw"
	"go.uber.org/yarpc/internal/errors"
	"go.uber.org/yarpc/transport"
	"go.uber.org/yarpc/transport/http"
	tch "go.uber.org/yarpc/transport/tchannel"
	"go.uber.org/yarpc/transport/transporttest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uber/tchannel-go"
	"github.com/uber/tchannel-go/testutils"
)

// all tests in this file should use these names for callers and services.
const (
	testCaller  = "testService-client"
	testService = "testService"
)

// roundTripTransport provides a function that sets up and tears down an
// Inbound, and provides an Outbound which knows how to call that Inbound.
type roundTripTransport interface {
	// Set up an Inbound serving Handler h, and call f with an Outbound that
	// knows how to talk to that Inbound.
	WithHandler(h transport.Handler, f func(transport.Outbound))
}

// handlerFunc wraps a function into a transport.Handler
type handlerFunc func(context.Context, transport.Options, *transport.Request, transport.ResponseWriter) error

func (f handlerFunc) Handle(ctx context.Context, opts transport.Options, r *transport.Request, w transport.ResponseWriter) error {
	return f(ctx, opts, r, w)
}

// httpTransport implements a roundTripTransport for HTTP.
type httpTransport struct{ t *testing.T }

func (ht httpTransport) WithHandler(h transport.Handler, f func(transport.Outbound)) {
	i := http.NewInbound("127.0.0.1:0")
	require.NoError(ht.t, i.Start(h, transport.NoDeps), "failed to start")
	defer i.Stop()

	addr := fmt.Sprintf("http://%v/", i.Addr().String())
	o := http.NewOutbound(addr)
	require.NoError(ht.t, o.Start(transport.NoDeps), "failed to start outbound")
	defer o.Stop()
	f(o)
}

// tchannelTransport implements a roundTripTransport for TChannel.
type tchannelTransport struct{ t *testing.T }

func (tt tchannelTransport) WithHandler(h transport.Handler, f func(transport.Outbound)) {
	serverOpts := testutils.NewOpts().SetServiceName(testService)
	clientOpts := testutils.NewOpts().SetServiceName(testCaller)
	testutils.WithServer(tt.t, serverOpts, func(ch *tchannel.Channel, hostPort string) {
		i := tch.NewInbound(ch)
		require.NoError(tt.t, i.Start(h, transport.NoDeps), "failed to start")
		defer i.Stop()
		// ^ the server is already listening so this will just set up the
		// handler.

		client := testutils.NewClient(tt.t, clientOpts)
		o := tch.NewOutbound(client, tch.HostPort(hostPort))
		require.NoError(tt.t, o.Start(transport.NoDeps), "failed to start outbound")
		defer o.Stop()

		f(o)
	})
}

func TestSimpleRoundTrip(t *testing.T) {
	transports := []roundTripTransport{
		httpTransport{t},
		tchannelTransport{t},
	}

	tests := []struct {
		requestHeaders  transport.Headers
		requestBody     string
		responseHeaders transport.Headers
		responseBody    string
		responseError   error

		wantError func(error)
	}{
		{
			requestHeaders:  transport.NewHeaders().With("token", "1234"),
			requestBody:     "world",
			responseHeaders: transport.NewHeaders().With("status", "ok"),
			responseBody:    "hello, world",
		},
		{
			requestBody:   "foo",
			responseError: errors.HandlerUnexpectedError(fmt.Errorf("great sadness")),
			wantError: func(err error) {
				assert.True(t, transport.IsUnexpectedError(err), err)
				assert.Equal(t, "UnexpectedError: great sadness", err.Error())
			},
		},
		{
			requestBody:   "bar",
			responseError: errors.HandlerBadRequestError(fmt.Errorf("missing service name")),
			wantError: func(err error) {
				assert.True(t, transport.IsBadRequestError(err))
				assert.Equal(t, "BadRequest: missing service name", err.Error())
			},
		},
		{
			requestBody: "baz",
			responseError: errors.RemoteUnexpectedError(
				`UnexpectedError: error for procedure "foo" of service "bar": great sadness`,
			),
			wantError: func(err error) {
				assert.True(t, transport.IsUnexpectedError(err))
				assert.Equal(t,
					`UnexpectedError: error for procedure "hello" of service "testService": `+
						`UnexpectedError: error for procedure "foo" of service "bar": great sadness`,
					err.Error())
			},
		},
		{
			requestBody: "qux",
			responseError: errors.RemoteBadRequestError(
				`BadRequest: unrecognized procedure "echo" for service "derp"`,
			),
			wantError: func(err error) {
				assert.True(t, transport.IsUnexpectedError(err))
				assert.Equal(t,
					`UnexpectedError: error for procedure "hello" of service "testService": `+
						`BadRequest: unrecognized procedure "echo" for service "derp"`,
					err.Error())
			},
		},
	}

	rootCtx := context.Background()
	for _, tt := range tests {
		for _, trans := range transports {
			requestMatcher := transporttest.NewRequestMatcher(t, &transport.Request{
				Caller:    testCaller,
				Service:   testService,
				Procedure: "hello",
				Encoding:  raw.Encoding,
				Headers:   tt.requestHeaders,
				Body:      bytes.NewReader([]byte(tt.requestBody)),
			})

			handler := handlerFunc(func(_ context.Context, _ transport.Options, r *transport.Request, w transport.ResponseWriter) error {
				assert.True(t, requestMatcher.Matches(r), "request mismatch: received %v", r)

				if tt.responseError != nil {
					return tt.responseError
				}

				if tt.responseHeaders.Len() > 0 {
					w.AddHeaders(tt.responseHeaders)
				}

				_, err := w.Write([]byte(tt.responseBody))
				assert.NoError(t, err, "failed to write response for %v", r)
				return err
			})

			ctx, _ := context.WithTimeout(rootCtx, 200*time.Millisecond)

			trans.WithHandler(handler, func(o transport.Outbound) {
				res, err := o.Call(ctx, &transport.Request{
					Caller:    testCaller,
					Service:   testService,
					Procedure: "hello",
					Encoding:  raw.Encoding,
					Headers:   tt.requestHeaders,
					Body:      bytes.NewReader([]byte(tt.requestBody)),
				})

				if tt.wantError != nil {
					if assert.Error(t, err, "%T: expected error, got %v", trans, res) {
						tt.wantError(err)

						// none of the errors returned by Call can be valid
						// Handler errors.
						_, ok := err.(errors.HandlerError)
						assert.False(t, ok, "%T: %T must not be a HandlerError", trans, err)
					}
				} else {
					responseMatcher := transporttest.NewResponseMatcher(t, &transport.Response{
						Headers: tt.responseHeaders,
						Body:    ioutil.NopCloser(bytes.NewReader([]byte(tt.responseBody))),
					})

					if assert.NoError(t, err, "%T: call failed", trans) {
						assert.True(t, responseMatcher.Matches(res), "%T: response mismatch", trans)
					}
				}
			})
		}
	}
}
