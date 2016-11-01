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
	"context"
	"fmt"
	"io/ioutil"
	"testing"
	"time"

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

	testProcedure       = "hello"
	testProcedureOneway = "hello-oneway"
)

// roundTripTransport provides a function that sets up and tears down an
// Inbound, and provides an Outbound which knows how to call that Inbound.
type roundTripTransport interface {
	// Set up an Inbound serving Registry r, and call f with an Outbound that
	// knows how to talk to that Inbound.
	WithRegistry(r transport.Registry, f func(transport.UnaryOutbound))
	WithRegistryOneway(r transport.Registry, f func(transport.OnewayOutbound))
}

type staticRegistry struct {
	Handler       transport.UnaryHandler
	OnewayHandler transport.OnewayHandler
}

func (r staticRegistry) Register([]transport.Registrant) {
	panic("cannot register methods on a static registry")
}

func (r staticRegistry) ServiceProcedures() []transport.ServiceProcedure {
	return []transport.ServiceProcedure{{Service: testService, Procedure: testProcedure}}
}

func (r staticRegistry) GetHandlerSpec(service string, procedure string) (transport.HandlerSpec, error) {
	if procedure == testProcedure {
		return transport.HandlerSpec{Type: transport.Unary, UnaryHandler: r.Handler}, nil
	} else {
		return transport.HandlerSpec{Type: transport.Oneway, OnewayHandler: r.OnewayHandler}, nil
	}
}

// handlerFunc wraps a function into a transport.Registry
type unaryHandlerFunc func(context.Context, *transport.Request, transport.ResponseWriter) error

func (f unaryHandlerFunc) HandleUnary(ctx context.Context, r *transport.Request, w transport.ResponseWriter) error {
	return f(ctx, r, w)
}

// onewayHandlerFunc wraps a function into a transport.Registry
type onewayHandlerFunc func(context.Context, *transport.Request) error

func (f onewayHandlerFunc) HandleOneway(ctx context.Context, r *transport.Request) error {
	return f(ctx, r)
}

// httpTransport implements a roundTripTransport for HTTP.
type httpTransport struct{ t *testing.T }

func (ht httpTransport) WithRegistry(r transport.Registry, f func(transport.UnaryOutbound)) {
	i := http.NewInbound("127.0.0.1:0")
	require.NoError(ht.t, i.Start(transport.ServiceDetail{Name: testService, Registry: r}, transport.NoDeps), "failed to start")
	defer i.Stop()

	addr := fmt.Sprintf("http://%v/", i.Addr().String())
	o := http.NewOutbound(addr)
	require.NoError(ht.t, o.Start(transport.NoDeps), "failed to start outbound")
	defer o.Stop()
	f(o)
}

func (ht httpTransport) WithRegistryOneway(r transport.Registry, f func(transport.OnewayOutbound)) {
	i := http.NewInbound("127.0.0.1:0")
	require.NoError(ht.t, i.Start(transport.ServiceDetail{Name: testService, Registry: r}, transport.NoDeps), "failed to start")
	defer i.Stop()

	addr := fmt.Sprintf("http://%v/", i.Addr().String())
	o := http.NewOnewayOutbound(addr)
	require.NoError(ht.t, o.Start(transport.NoDeps), "failed to start outbound")
	defer o.Stop()
	f(o)
}

// tchannelTransport implements a roundTripTransport for TChannel.
type tchannelTransport struct{ t *testing.T }

func (tt tchannelTransport) WithRegistry(r transport.Registry, f func(transport.UnaryOutbound)) {
	serverOpts := testutils.NewOpts().SetServiceName(testService)
	clientOpts := testutils.NewOpts().SetServiceName(testCaller)
	testutils.WithServer(tt.t, serverOpts, func(ch *tchannel.Channel, hostPort string) {
		i := tch.NewInbound(ch)
		require.NoError(tt.t, i.Start(transport.ServiceDetail{Name: testService, Registry: r}, transport.NoDeps), "failed to start")

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

func (tt tchannelTransport) WithRegistryOneway(r transport.Registry, f func(transport.OnewayOutbound)) {
	panic("tchannel does not support oneway calls")
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
				Procedure: testProcedure,
				Encoding:  raw.Encoding,
				Headers:   tt.requestHeaders,
				Body:      bytes.NewReader([]byte(tt.requestBody)),
			})

			handler := unaryHandlerFunc(func(_ context.Context, r *transport.Request, w transport.ResponseWriter) error {
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

			ctx, cancel := context.WithTimeout(rootCtx, 200*time.Millisecond)
			defer cancel()

			registry := staticRegistry{Handler: handler}
			trans.WithRegistry(registry, func(o transport.UnaryOutbound) {
				res, err := o.CallUnary(ctx, &transport.Request{
					Caller:    testCaller,
					Service:   testService,
					Procedure: testProcedure,
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

func TestSimpleRoundTripOneway(t *testing.T) {
	trans := httpTransport{t}

	tests := []struct {
		name           string
		requestHeaders transport.Headers
		requestBody    string
	}{
		{
			name:           "hello world",
			requestHeaders: transport.NewHeaders().With("foo", "bar"),
			requestBody:    "hello world",
		},
		{
			name:           "empty",
			requestHeaders: transport.NewHeaders(),
			requestBody:    "",
		},
	}

	rootCtx := context.Background()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			requestMatcher := transporttest.NewRequestMatcher(t, &transport.Request{
				Caller:    testCaller,
				Service:   testService,
				Procedure: testProcedureOneway,
				Encoding:  raw.Encoding,
				Headers:   tt.requestHeaders,
				Body:      bytes.NewReader([]byte(tt.requestBody)),
			})

			handlerDone := make(chan struct{})

			onewayHandler := onewayHandlerFunc(func(_ context.Context, r *transport.Request) error {
				assert.True(t, requestMatcher.Matches(r), "request mismatch: received %v", r)

				// Pretend to work: this delay should not slow down tests since it is a
				// server-side operation
				time.Sleep(5 * time.Second)

				// fill the channel, telling the client (which should not be waiting for
				// a response) that the handler finished executing
				handlerDone <- struct{}{}

				return nil
			})

			registry := staticRegistry{OnewayHandler: onewayHandler}

			trans.WithRegistryOneway(registry, func(o transport.OnewayOutbound) {
				ack, err := o.CallOneway(rootCtx, &transport.Request{
					Caller:    testCaller,
					Service:   testService,
					Procedure: testProcedureOneway,
					Encoding:  raw.Encoding,
					Headers:   tt.requestHeaders,
					Body:      bytes.NewReader([]byte(tt.requestBody)),
				})

				select {
				case <-handlerDone:
					// if the server filled the channel, it means we waited for the server
					// to complete the request
					assert.Fail(t, "server handler executed before client")
				default:
				}

				if assert.NoError(t, err, "%T: oneway call failed for test '%v'", trans, tt.name) {
					assert.NotNil(t, ack)
				}
			})
		})
	}
}
