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

package yarpctest

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc"
	"go.uber.org/yarpc/api/middleware"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/transport/grpc"
	"go.uber.org/yarpc/transport/http"
	"go.uber.org/yarpc/transport/tchannel"
	"go.uber.org/yarpc/x/yarpctest/api"
)

// HTTPRequest creates a new YARPC http request.
func HTTPRequest(options ...api.RequestOption) api.Action {
	return api.ActionFunc(func(t testing.TB) {
		opts := api.NewRequestOpts()
		for _, option := range options {
			option.ApplyRequest(&opts)
		}

		trans := http.NewTransport()
		httpOut := trans.NewSingleOutbound(fmt.Sprintf("http://127.0.0.1:%d/", opts.Port))
		out := middleware.ApplyUnaryOutbound(httpOut, yarpc.UnaryOutboundMiddleware(opts.UnaryMiddleware...))

		require.NoError(t, trans.Start())
		defer func() { assert.NoError(t, trans.Stop()) }()

		require.NoError(t, out.Start())
		defer func() { assert.NoError(t, out.Stop()) }()

		resp, cancel, err := sendRequest(out, opts.GiveRequest, opts.GiveTimeout)
		defer cancel()
		validateError(t, err, opts.WantError)
		if opts.WantError == nil {
			validateResponse(t, resp, opts.WantResponse)
		}
	})
}

// TChannelRequest creates a new tchannel request.
func TChannelRequest(options ...api.RequestOption) api.Action {
	return api.ActionFunc(func(t testing.TB) {
		opts := api.NewRequestOpts()
		for _, option := range options {
			option.ApplyRequest(&opts)
		}

		trans, err := tchannel.NewTransport(tchannel.ServiceName(opts.GiveRequest.Caller))
		require.NoError(t, err)
		tchannelOut := trans.NewSingleOutbound(fmt.Sprintf("127.0.0.1:%d", opts.Port))
		out := middleware.ApplyUnaryOutbound(tchannelOut, yarpc.UnaryOutboundMiddleware(opts.UnaryMiddleware...))

		require.NoError(t, trans.Start())
		defer func() { assert.NoError(t, trans.Stop()) }()

		require.NoError(t, out.Start())
		defer func() { assert.NoError(t, out.Stop()) }()

		resp, cancel, err := sendRequest(out, opts.GiveRequest, opts.GiveTimeout)
		defer cancel()
		validateError(t, err, opts.WantError)
		if opts.WantError == nil {
			validateResponse(t, resp, opts.WantResponse)
		}
	})
}

// GRPCRequest creates a new grpc unary request.
func GRPCRequest(options ...api.RequestOption) api.Action {
	return api.ActionFunc(func(t testing.TB) {
		opts := api.NewRequestOpts()
		for _, option := range options {
			option.ApplyRequest(&opts)
		}

		trans := grpc.NewTransport()
		grpcOut := trans.NewSingleOutbound(fmt.Sprintf("127.0.0.1:%d", opts.Port))
		out := middleware.ApplyUnaryOutbound(grpcOut, yarpc.UnaryOutboundMiddleware(opts.UnaryMiddleware...))

		require.NoError(t, trans.Start())
		defer func() { assert.NoError(t, trans.Stop()) }()

		require.NoError(t, out.Start())
		defer func() { assert.NoError(t, out.Stop()) }()

		resp, cancel, err := sendRequest(out, opts.GiveRequest, opts.GiveTimeout)
		defer cancel()
		validateError(t, err, opts.WantError)
		if opts.WantError == nil {
			validateResponse(t, resp, opts.WantResponse)
		}
	})
}

func sendRequest(out transport.UnaryOutbound, request *transport.Request, timeout time.Duration) (*transport.Response, context.CancelFunc, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	resp, err := out.Call(ctx, request)
	return resp, cancel, err
}

func validateError(t testing.TB, actualErr error, wantError error) {
	if wantError != nil {
		require.Error(t, actualErr)
		require.Contains(t, actualErr.Error(), wantError.Error())
		return
	}
	require.NoError(t, actualErr)
}

func validateResponse(t testing.TB, actualResp *transport.Response, expectedResp *transport.Response) {
	var actualBody []byte
	var expectedBody []byte
	var err error
	if actualResp.Body != nil {
		actualBody, err = ioutil.ReadAll(actualResp.Body)
		require.NoError(t, err)
	}
	if expectedResp.Body != nil {
		expectedBody, err = ioutil.ReadAll(expectedResp.Body)
		require.NoError(t, err)
	}
	assert.Equal(t, string(actualBody), string(expectedBody))
	for k, v := range expectedResp.Headers.Items() {
		actualValue, ok := actualResp.Headers.Get(k)
		require.True(t, ok, "header %q was not set on the response", k)
		require.Equal(t, actualValue, v, "headers did not match for %q", k)
	}
}

// UNARY-SPECIFIC REQUEST OPTIONS

// Body sets the body on a request to the raw representation of the msg field.
func Body(msg string) api.RequestOption {
	return api.RequestOptionFunc(func(opts *api.RequestOpts) {
		opts.GiveRequest.Body = bytes.NewBufferString(msg)
	})
}

// GiveTimeout will set the timeout for the request.
func GiveTimeout(duration time.Duration) api.RequestOption {
	return api.RequestOptionFunc(func(opts *api.RequestOpts) {
		opts.GiveTimeout = duration
	})
}

// UnaryOutboundMiddleware sets unary outbound middleware for a request.
//
// Multiple invocations will append to existing middleware.
func UnaryOutboundMiddleware(mw ...middleware.UnaryOutbound) api.RequestOption {
	return api.RequestOptionFunc(func(opts *api.RequestOpts) {
		opts.UnaryMiddleware = append(opts.UnaryMiddleware, mw...)
	})
}

// WantError creates an assertion on the request response to validate the
// error.
func WantError(errMsg string) api.RequestOption {
	return api.RequestOptionFunc(func(opts *api.RequestOpts) {
		opts.WantError = errors.New(errMsg)
	})
}

// WantRespBody will assert that the response body matches at the end of the
// request.
func WantRespBody(body string) api.RequestOption {
	return api.RequestOptionFunc(func(opts *api.RequestOpts) {
		opts.WantResponse.Body = ioutil.NopCloser(bytes.NewBufferString(body))
	})
}

// GiveAndWantLargeBodyIsEchoed creates an extremely large random byte buffer
// and validates that the body is echoed back to the response.
func GiveAndWantLargeBodyIsEchoed(numOfBytes int) api.RequestOption {
	return api.RequestOptionFunc(func(opts *api.RequestOpts) {
		body := bytes.Repeat([]byte("t"), numOfBytes)
		opts.GiveRequest.Body = bytes.NewReader(body)
		opts.WantResponse.Body = ioutil.NopCloser(bytes.NewReader(body))
	})
}
