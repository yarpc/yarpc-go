// Copyright (c) 2026 Uber Technologies, Inc.
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
	"io"
	"strings"
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

		sendRequestAndValidateResp(t, out, opts)
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

		sendRequestAndValidateResp(t, out, opts)
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

		sendRequestAndValidateResp(t, out, opts)
	})
}

func sendRequestAndValidateResp(t testing.TB, out transport.UnaryOutbound, opts api.RequestOpts) {
	f := func(i int) bool {
		resp, cancel, err := sendRequest(out, opts.GiveRequest, opts.GiveTimeout)
		defer cancel()

		if i == opts.RetryCount {
			validateError(t, err, opts.WantError)
			if opts.WantError == nil {
				validateResponse(t, resp, opts.WantResponse)
			}
			return true
		}

		if err != nil || matchResponse(resp, opts.WantResponse) != nil {
			return false
		}

		return true
	}

	for i := 0; i < opts.RetryCount+1; i++ {
		if ok := f(i); ok {
			return
		}
		time.Sleep(opts.RetryInterval)
	}
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
	require.NoError(t, matchResponse(actualResp, expectedResp), "response mismatch")
}

func matchResponse(actualResp *transport.Response, expectedResp *transport.Response) error {
	var actualBody []byte
	var expectedBody []byte
	var err error
	if actualResp.Body != nil {
		actualBody, err = io.ReadAll(actualResp.Body)
		if err != nil {
			return fmt.Errorf("failed to read response body")
		}
	}
	if expectedResp.Body != nil {
		expectedBody, err = io.ReadAll(expectedResp.Body)
		if err != nil {
			return fmt.Errorf("failed to read response body")
		}
	}
	if string(actualBody) != string(expectedBody) {
		return fmt.Errorf("response body mismatch, expect %s, got %s",
			expectedBody, actualBody)
	}
	for k, v := range expectedResp.Headers.Items() {
		actualValue, ok := actualResp.Headers.Get(k)
		if !ok {
			return fmt.Errorf("headler %q was not set on the response", k)
		}
		if actualValue != v {
			return fmt.Errorf("headers mismatch for %q, expected %v, got %v",
				k, v, actualValue)
		}
	}
	return nil
}

// UNARY-SPECIFIC REQUEST OPTIONS

// Body sets the body on a request to the raw representation of the msg field.
func Body(msg string) api.RequestOption {
	return api.RequestOptionFunc(func(opts *api.RequestOpts) {
		opts.GiveRequest.Body = strings.NewReader(msg)
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
		opts.WantResponse.Body = io.NopCloser(strings.NewReader(body))
	})
}

// GiveAndWantLargeBodyIsEchoed creates an extremely large random byte buffer
// and validates that the body is echoed back to the response.
func GiveAndWantLargeBodyIsEchoed(numOfBytes int) api.RequestOption {
	return api.RequestOptionFunc(func(opts *api.RequestOpts) {
		body := bytes.Repeat([]byte("t"), numOfBytes)
		opts.GiveRequest.Body = bytes.NewReader(body)
		opts.WantResponse.Body = io.NopCloser(bytes.NewReader(body))
	})
}

// Retry retries the request for a given times, until the request succeeds
// and the response matches.
func Retry(count int, interval time.Duration) api.RequestOption {
	return api.RequestOptionFunc(func(opts *api.RequestOpts) {
		opts.RetryCount = count
		opts.RetryInterval = interval
	})
}
