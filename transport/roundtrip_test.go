// Copyright (c) 2018 Uber Technologies, Inc.
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
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uber/tchannel-go"
	"github.com/uber/tchannel-go/testutils"
	"go.uber.org/yarpc/api/middleware"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/api/transport/transporttest"
	"go.uber.org/yarpc/encoding/raw"
	"go.uber.org/yarpc/internal/testtime"
	"go.uber.org/yarpc/transport/grpc"
	"go.uber.org/yarpc/transport/http"
	tch "go.uber.org/yarpc/transport/tchannel"
	"go.uber.org/yarpc/x/yarpctest"
	"go.uber.org/yarpc/x/yarpctest/api"
	"go.uber.org/yarpc/x/yarpctest/types"
	"go.uber.org/yarpc/yarpcerrors"
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
	// Name is the string representation of the transport. eg http, grpc, tchannel
	Name() string
	// Set up an Inbound serving Router r, and call f with an Outbound that
	// knows how to talk to that Inbound.
	WithRouter(r transport.Router, f func(transport.UnaryOutbound))
	WithRouterOneway(r transport.Router, f func(transport.OnewayOutbound))
}

type staticRouter struct {
	Handler       transport.UnaryHandler
	OnewayHandler transport.OnewayHandler
}

func (r staticRouter) Register([]transport.Procedure) {
	panic("cannot register methods on a static router")
}

func (r staticRouter) Procedures() []transport.Procedure {
	return []transport.Procedure{{Name: testProcedure, Service: testService}}
}

func (r staticRouter) Choose(ctx context.Context, req *transport.Request) (transport.HandlerSpec, error) {
	if req.Procedure == testProcedure {
		return transport.NewUnaryHandlerSpec(r.Handler), nil
	}
	return transport.NewOnewayHandlerSpec(r.OnewayHandler), nil
}

// handlerFunc wraps a function into a transport.Router
type unaryHandlerFunc func(context.Context, *transport.Request, transport.ResponseWriter) error

func (f unaryHandlerFunc) Handle(ctx context.Context, r *transport.Request, w transport.ResponseWriter) error {
	return f(ctx, r, w)
}

// onewayHandlerFunc wraps a function into a transport.Router
type onewayHandlerFunc func(context.Context, *transport.Request) error

func (f onewayHandlerFunc) HandleOneway(ctx context.Context, r *transport.Request) error {
	return f(ctx, r)
}

// httpTransport implements a roundTripTransport for HTTP.
type httpTransport struct{ t *testing.T }

func (ht httpTransport) Name() string {
	return "http"
}

func (ht httpTransport) WithRouter(r transport.Router, f func(transport.UnaryOutbound)) {
	httpTransport := http.NewTransport()

	i := httpTransport.NewInbound("127.0.0.1:0")
	i.SetRouter(r)
	require.NoError(ht.t, i.Start(), "failed to start")
	defer i.Stop()

	o := httpTransport.NewSingleOutbound(fmt.Sprintf("http://%s", i.Addr().String()))
	require.NoError(ht.t, o.Start(), "failed to start outbound")
	defer o.Stop()
	f(o)
}

func (ht httpTransport) WithRouterOneway(r transport.Router, f func(transport.OnewayOutbound)) {
	httpTransport := http.NewTransport()

	i := httpTransport.NewInbound("127.0.0.1:0")
	i.SetRouter(r)
	require.NoError(ht.t, i.Start(), "failed to start")
	defer i.Stop()

	o := httpTransport.NewSingleOutbound(fmt.Sprintf("http://%s", i.Addr().String()))
	require.NoError(ht.t, o.Start(), "failed to start outbound")
	defer o.Stop()
	f(o)
}

// tchannelTransport implements a roundTripTransport for TChannel.
type tchannelTransport struct{ t *testing.T }

func (tt tchannelTransport) Name() string {
	return "tchannel"
}

func (tt tchannelTransport) WithRouter(r transport.Router, f func(transport.UnaryOutbound)) {
	serverOpts := testutils.NewOpts().SetServiceName(testService)
	clientOpts := testutils.NewOpts().SetServiceName(testCaller)
	testutils.WithServer(tt.t, serverOpts, func(ch *tchannel.Channel, hostPort string) {
		ix, err := tch.NewChannelTransport(tch.WithChannel(ch))
		require.NoError(tt.t, err)

		i := ix.NewInbound()
		i.SetRouter(r)
		require.NoError(tt.t, ix.Start(), "failed to start inbound transport")
		require.NoError(tt.t, i.Start(), "failed to start inbound")

		defer i.Stop()
		// ^ the server is already listening so this will just set up the
		// handler.

		client := testutils.NewClient(tt.t, clientOpts)
		ox, err := tch.NewChannelTransport(tch.WithChannel(client))
		require.NoError(tt.t, err)

		o := ox.NewSingleOutbound(hostPort)
		require.NoError(tt.t, ox.Start(), "failed to start outbound transport")
		require.NoError(tt.t, o.Start(), "failed to start outbound")
		defer o.Stop()

		f(o)
	})
}

func (tt tchannelTransport) WithRouterOneway(r transport.Router, f func(transport.OnewayOutbound)) {
	panic("tchannel does not support oneway calls")
}

// grpcTransport implements a roundTripTransport for gRPC.
type grpcTransport struct{ t *testing.T }

func (gt grpcTransport) Name() string {
	return "grpc"
}

func (gt grpcTransport) WithRouter(r transport.Router, f func(transport.UnaryOutbound)) {
	grpcTransport := grpc.NewTransport()
	require.NoError(gt.t, grpcTransport.Start(), "failed to start transport")
	defer grpcTransport.Stop()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(gt.t, err)
	i := grpcTransport.NewInbound(listener)
	i.SetRouter(r)
	require.NoError(gt.t, i.Start(), "failed to start inbound")
	defer i.Stop()

	o := grpcTransport.NewSingleOutbound(listener.Addr().String())
	require.NoError(gt.t, o.Start(), "failed to start outbound")
	defer o.Stop()
	f(o)
}

func (gt grpcTransport) WithRouterOneway(r transport.Router, f func(transport.OnewayOutbound)) {
	panic("grpc does not support oneway calls")
}

func TestSimpleRoundTrip(t *testing.T) {
	transports := []roundTripTransport{
		httpTransport{t},
		tchannelTransport{t},
		grpcTransport{t},
	}

	tests := []struct {
		name string

		requestHeaders  transport.Headers
		requestBody     string
		responseHeaders transport.Headers
		responseBody    string
		responseError   error

		wantError func(error)
	}{
		{
			name:            "headers",
			requestHeaders:  transport.NewHeaders().With("token", "1234"),
			requestBody:     "world",
			responseHeaders: transport.NewHeaders().With("status", "ok"),
			responseBody:    "hello, world",
		},
		{
			name:          "internal err",
			requestBody:   "foo",
			responseError: yarpcerrors.Newf(yarpcerrors.CodeInternal, "great sadness"),
			wantError: func(err error) {
				assert.True(t, yarpcerrors.FromError(err).Code() == yarpcerrors.CodeInternal, err.Error())
			},
		},
		{
			name:          "invalid arg",
			requestBody:   "bar",
			responseError: yarpcerrors.Newf(yarpcerrors.CodeInvalidArgument, "missing service name"),
			wantError: func(err error) {
				assert.True(t, yarpcerrors.FromError(err).Code() == yarpcerrors.CodeInvalidArgument, err.Error())
			},
		},
	}

	for _, tt := range tests {
		for _, trans := range transports {
			t.Run(tt.name+"/"+trans.Name(), func(t *testing.T) {
				requestMatcher := transporttest.NewRequestMatcher(t, &transport.Request{
					Caller:    testCaller,
					Service:   testService,
					Transport: trans.Name(),
					Procedure: testProcedure,
					Encoding:  raw.Encoding,
					Headers:   tt.requestHeaders,
					Body:      bytes.NewBufferString(tt.requestBody),
				})

				handler := unaryHandlerFunc(func(_ context.Context, r *transport.Request, w transport.ResponseWriter) error {
					r.Headers.Del("user-agent") // for gRPC
					r.Headers.Del(":authority") // for gRPC
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

				ctx, cancel := context.WithTimeout(context.Background(), 200*testtime.Millisecond)
				defer cancel()

				router := staticRouter{Handler: handler}
				trans.WithRouter(router, func(o transport.UnaryOutbound) {
					res, err := o.Call(ctx, &transport.Request{
						Caller:    testCaller,
						Service:   testService,
						Procedure: testProcedure,
						Encoding:  raw.Encoding,
						Headers:   tt.requestHeaders,
						Body:      bytes.NewBufferString(tt.requestBody),
					})

					if tt.wantError != nil {
						if assert.Error(t, err, "%T: expected error, got %v", trans, res) {
							tt.wantError(err)
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
				Transport: trans.Name(),
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
				testtime.Sleep(5 * time.Second)

				// close the channel, telling the client (which should not be waiting for
				// a response) that the handler finished executing
				close(handlerDone)

				return nil
			})

			router := staticRouter{OnewayHandler: onewayHandler}

			trans.WithRouterOneway(router, func(o transport.OnewayOutbound) {
				ctx, cancel := context.WithTimeout(rootCtx, time.Second)
				defer cancel()
				ack, err := o.CallOneway(ctx, &transport.Request{
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
					assert.Fail(t, "client waited for server handler to finish executing")
				default:
				}

				if assert.NoError(t, err, "%T: oneway call failed for test '%v'", trans, tt.name) {
					assert.NotNil(t, ack)
				}
			})
		})
	}
}

func TestRoundTripMeta(t *testing.T) {
	const (
		id           = "test-id"
		host1, host2 = "test-host-1", "test-host-2"
		env1, env2   = "test-env-1", "test-env-2"
		service      = "test-service"
		caller       = "unknown" /* from x/yarpctest */
	)

	reqMatcher := transporttest.NewRequestMatcher(t, &transport.Request{
		ID:          id,
		Host:        host1,
		Environment: env1,
		Caller:      caller,
		Service:     service,
		Procedure:   testProcedure,
		Encoding:    raw.Encoding,
		Headers:     transport.NewHeaders().With("foo", "bar"),
		Body:        bytes.NewBufferString(""),
	})

	respMatcher := transporttest.NewResponseMatcher(t, &transport.Response{
		ID:          id,
		Host:        host2,
		Environment: env2,
		Service:     service,
		Headers:     transport.NewHeaders().With("fizz", "buzz"),
		Body:        ioutil.NopCloser(bytes.NewBufferString("")),
	})

	// add meta information to transport.Request for all outbound calls
	outboundMW := middleware.UnaryOutboundFunc(func(ctx context.Context, req *transport.Request, next transport.UnaryOutbound) (*transport.Response, error) {
		// add meta
		req.ID = id
		req.Host = host1
		req.Environment = env1

		// issue call
		resp, err := next.Call(ctx, req)
		if err != nil {
			return nil, err
		}

		// validate response
		if !assert.True(t, respMatcher.Matches(resp)) {
			return nil, errors.New("unexpected response")
		}
		return resp, nil
	})

	handler := &types.UnaryHandler{
		Handler: api.UnaryHandlerFunc(func(ctx context.Context, req *transport.Request, resw transport.ResponseWriter) error {
			// validate request
			if !assert.True(t, reqMatcher.Matches(req)) {
				return errors.New("unexpected request")
			}

			// get responseMeta
			responseMetaWriter, ok := resw.(transport.ResponseMetaWriter)
			if !assert.True(t, ok) {
				return errors.New("could not upcast response writer to transport.ResponseMetaWriter")
			}

			// validate responseMeta
			resMeta := responseMetaWriter.ResponseMeta()
			if !assert.NotNil(t, resMeta) {
				return errors.New("responseMeta is nil")
			}
			if !assert.NotEmpty(t, resMeta.ID) {
				return errors.New("ResponseMeta.ID is empty")
			}
			if !assert.Empty(t, resMeta.Host) {
				return errors.New("ResponseMeta.Host is unexpectedly set")
			}
			if !assert.Empty(t, resMeta.Environment) {
				return errors.New("ResponseMeta.Environment is unexpectedly set")
			}
			if !assert.NotEmpty(t, resMeta.Service) {
				return errors.New("ResponseMeta.Service is empty")
			}

			// write additional response meta
			resMeta.Host = host2
			resMeta.Environment = env2
			resMeta.AddHeaders(transport.NewHeaders().With("fizz", "buzz"))

			return nil
		}),
	}

	ports := yarpctest.NewPortProvider(t)

	serviceOpts := []api.ServiceOption{
		yarpctest.Name(service),
		yarpctest.Proc(yarpctest.Name(testProcedure), handler),
	}

	requestOpts := []api.RequestOption{
		yarpctest.Service(service),
		yarpctest.Procedure(testProcedure),
		yarpctest.GiveTimeout(time.Second),
		yarpctest.UnaryOutboundMiddleware(outboundMW),
		yarpctest.WithHeader("foo", "bar"),
	}

	tests := []struct {
		name    string
		service api.Lifecycle
		request api.Action
	}{
		{
			name:    "http",
			service: yarpctest.HTTPService(append(serviceOpts, ports.NamedPort("http"))...),
			request: yarpctest.HTTPRequest(append(requestOpts, ports.NamedPort("http"))...),
		},
		{
			name:    "TChannel",
			service: yarpctest.TChannelService(append(serviceOpts, ports.NamedPort("TChannel"))...),
			request: yarpctest.TChannelRequest(append(requestOpts, ports.NamedPort("TChannel"))...),
		},
		// TODO(apeatsbond): add gRPC test
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.NoError(t, tt.service.Start(t))
			tt.request.Run(t)
			require.NoError(t, tt.service.Stop(t))
		})
	}
}
