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

package yarpctchannel

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	tchannel "github.com/uber/tchannel-go"
	"github.com/uber/tchannel-go/testutils"
	yarpc "go.uber.org/yarpc/v2"
	"go.uber.org/yarpc/v2/internal/internaltesttime"
	"go.uber.org/yarpc/v2/yarpcerror"
)

func TestOutboundHeaders(t *testing.T) {
	tests := map[string]struct {
		headers     yarpc.Headers
		wantHeaders []byte
	}{
		"original header": {
			headers: yarpc.NewHeaders().With("contextfoo", "bar"),
			wantHeaders: []byte{
				0x00, 0x01,
				0x00, 0x0A, 'c', 'o', 'n', 't', 'e', 'x', 't', 'f', 'o', 'o',
				0x00, 0x03, 'b', 'a', 'r',
			},
		},
		"canonicalized header": {
			headers: yarpc.NewHeaders().With("Foo", "bar"),
			wantHeaders: []byte{
				0x00, 0x01,
				0x00, 0x03, 'f', 'o', 'o',
				0x00, 0x03, 'b', 'a', 'r',
			},
		},
	}

	for msg, tt := range tests {
		t.Run(msg, func(t *testing.T) {
			ctx := context.Background()
			ctx, cancel := context.WithTimeout(ctx, time.Second)
			defer cancel()

			server := testutils.NewServer(t, nil)
			defer server.Close()
			addr := server.PeerInfo().HostPort

			server.GetSubChannel("service").SetHandler(tchannel.HandlerFunc(
				func(ctx context.Context, call *tchannel.InboundCall) {
					headers, body, err := readArgs(call)
					if assert.NoError(t, err, "failed to read request") {
						assert.Equal(t, tt.wantHeaders, headers, "headers did not match")
						assert.Equal(t, []byte("world"), body)
					}

					err = writeArgs(call.Response(), []byte{0x00, 0x00}, []byte("bye!"))
					assert.NoError(t, err, "failed to write response")
				}))

			dialer := &Dialer{
				Caller: "caller",
			}
			require.NoError(t, dialer.Start(ctx))
			defer dialer.Stop(ctx)

			outbound := &Outbound{
				Dialer: dialer,
				Addr:   addr,
			}

			req := &yarpc.Request{
				Caller:    "caller",
				Service:   "service",
				Encoding:  yarpc.Encoding("raw"),
				Procedure: "hello",
				Headers:   tt.headers,
			}
			reqBody := yarpc.NewBufferString("world")
			_, _, err := outbound.Call(ctx, req, reqBody)
			assert.NoError(t, err, "call failed")
		})
	}
}

func TestCallSuccess(t *testing.T) {
	const (
		headerKey = "foo-BAR-BaZ"
		headerVal = "FooBarBaz"
	)
	tests := map[string]struct {
		msg                   string
		outbound              *Outbound
		headerVal             []byte
		withServiceRespHeader bool
	}{
		"exactCaseHeader options on": {
			outbound: &Outbound{HeaderCase: OriginalHeaderCase},
			headerVal: []byte{
				0x00, 0x01,
				0x00, 0x0b, 'f', 'o', 'o', '-', 'B', 'A', 'R', '-', 'B', 'a', 'Z',
				0x00, 0x09, 'F', 'o', 'o', 'B', 'a', 'r', 'B', 'a', 'z',
			},
		},
		"exactCaseHeader options off": {
			outbound: &Outbound{},
			headerVal: []byte{
				0x00, 0x01,
				0x00, 0x0b, 'f', 'o', 'o', '-', 'b', 'a', 'r', '-', 'b', 'a', 'z',
				0x00, 0x09, 'F', 'o', 'o', 'B', 'a', 'r', 'B', 'a', 'z',
			},
		},
		"exactCaseHeader options off with service response header": {
			outbound: &Outbound{},
			headerVal: []byte{
				0x00, 0x01,
				0x00, 0x0b, 'f', 'o', 'o', '-', 'b', 'a', 'r', '-', 'b', 'a', 'z',
				0x00, 0x09, 'F', 'o', 'o', 'B', 'a', 'r', 'B', 'a', 'z',
			},
			withServiceRespHeader: true,
		},
	}

	for msg, tt := range tests {
		t.Run(msg, func(t *testing.T) {
			server := testutils.NewServer(t, nil)
			defer server.Close()
			serverAddr := server.PeerInfo().HostPort

			server.GetSubChannel("service").SetHandler(tchannel.HandlerFunc(
				func(ctx context.Context, call *tchannel.InboundCall) {
					assert.Equal(t, "caller", call.CallerName())
					assert.Equal(t, "service", call.ServiceName())
					assert.Equal(t, tchannel.Raw, call.Format())
					assert.Equal(t, "hello", call.MethodString())
					headers, body, err := readArgs(call)
					if assert.NoError(t, err, "failed to read request") {
						assert.Equal(t, tt.headerVal, headers)
						assert.Equal(t, []byte("world"), body)
					}

					dl, ok := ctx.Deadline()
					assert.True(t, ok, "deadline expected")
					assert.WithinDuration(t, time.Now(), dl, internaltesttime.Second)

					if tt.withServiceRespHeader {
						// test with response service name header
						err = writeArgs(call.Response(),
							[]byte{
								0x00, 0x02,
								0x00, 0x03, 'f', 'o', 'o',
								0x00, 0x03, 'b', 'a', 'r',
								0x00, 0x0d, '$', 'r', 'p', 'c', '$', '-', 's', 'e', 'r', 'v', 'i', 'c', 'e',
								0x00, 0x07, 's', 'e', 'r', 'v', 'i', 'c', 'e',
							}, []byte("great success"))
					} else {
						// test without response service name header
						err = writeArgs(call.Response(),
							[]byte{
								0x00, 0x01,
								0x00, 0x03, 'f', 'o', 'o',
								0x00, 0x03, 'b', 'a', 'r',
							}, []byte("great success"))
					}
					assert.NoError(t, err, "no write response")
				}))

			ctx, cancel := context.WithTimeout(context.Background(), 200*internaltesttime.Millisecond)
			defer cancel()

			dialer := &Dialer{Caller: "caller"}
			require.NoError(t, dialer.Start(ctx))
			defer dialer.Stop(ctx)

			tt.outbound.Dialer = dialer
			tt.outbound.Addr = serverAddr

			res, resBody, err := tt.outbound.Call(
				ctx,
				&yarpc.Request{
					Caller:    "caller",
					Service:   "service",
					Encoding:  yarpc.Encoding("raw"),
					Procedure: "hello",
					Headers:   yarpc.NewHeaders().With(headerKey, headerVal),
				},
				yarpc.NewBufferString("world"),
			)

			if !assert.NoError(t, err, "failed to make call") {
				return
			}

			require.NotNil(t, res)
			assert.Nil(t, res.ApplicationErrorInfo, "not application error")

			foo, ok := res.Headers.Get("foo")
			assert.True(t, ok, "value for foo expected")
			assert.Equal(t, "bar", foo, "foo value mismatch")

			assert.Equal(t, yarpc.NewBufferString("great success"), resBody)
		})
	}
}

func TestCallFailures(t *testing.T) {
	const (
		unexpectedMethod = "unexpected"
		unknownMethod    = "unknown"
	)
	server := testutils.NewServer(t, nil)
	defer server.Close()
	serverAddr := server.PeerInfo().HostPort

	server.GetSubChannel("service").SetHandler(tchannel.HandlerFunc(
		func(ctx context.Context, call *tchannel.InboundCall) {
			var err error
			if call.MethodString() == unexpectedMethod {
				err = tchannel.NewSystemError(
					tchannel.ErrCodeUnexpected, "great sadness")
				call.Response().SendSystemError(err)
			} else if call.MethodString() == unknownMethod {
				err = tchannel.NewSystemError(
					tchannel.ErrCodeBadRequest, "unknown method")
				call.Response().SendSystemError(err)
			} else {
				err = writeArgs(call.Response(),
					[]byte{
						0x00, 0x01,
						0x00, 0x0d, '$', 'r', 'p', 'c', '$', '-', 's', 'e', 'r', 'v', 'i', 'c', 'e',
						0x00, 0x05, 'w', 'r', 'o', 'n', 'g',
					}, []byte("bad sadness"))
				assert.NoError(t, err, "o write response")
			}
		}))

	type testCase struct {
		procedure string
		message   string
	}

	tests := map[string]testCase{
		"unexpected method": {
			procedure: unexpectedMethod,
			message:   "great sadness",
		},
		"unknown method": {
			procedure: unknownMethod,
			message:   "unknown method",
		},
		"service name mismatch": {
			procedure: "wrong service name",
			message:   "does not match",
		},
	}

	for msg, tt := range tests {
		t.Run(msg, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 200*internaltesttime.Millisecond)
			defer cancel()

			dialer := &Dialer{Caller: "caller"}
			require.NoError(t, dialer.Start(ctx))
			defer dialer.Stop(ctx)

			outbound := &Outbound{Dialer: dialer, Addr: serverAddr}

			_, _, err := outbound.Call(
				ctx,
				&yarpc.Request{
					Caller:    "caller",
					Service:   "service",
					Encoding:  yarpc.Encoding("raw"),
					Procedure: tt.procedure,
				},
				yarpc.NewBufferString("sup"),
			)

			assert.Error(t, err, "expected failure")
			assert.Contains(t, err.Error(), tt.message)
		})
	}
}

func TestCallError(t *testing.T) {
	server := testutils.NewServer(t, nil)
	defer server.Close()
	serverAddr := server.PeerInfo().HostPort

	server.GetSubChannel("service").SetHandler(tchannel.HandlerFunc(
		func(ctx context.Context, call *tchannel.InboundCall) {
			assert.Equal(t, "caller", call.CallerName())
			assert.Equal(t, "service", call.ServiceName())
			assert.Equal(t, tchannel.Raw, call.Format())
			assert.Equal(t, "hello", call.MethodString())

			headers, body, err := readArgs(call)
			if assert.NoError(t, err, "failed to read request") {
				assert.Equal(t, []byte{0x00, 0x00}, headers)
				assert.Equal(t, []byte("world"), body)
			}

			dl, ok := ctx.Deadline()
			assert.True(t, ok, "deadline expected")
			assert.WithinDuration(t, time.Now(), dl, 200*internaltesttime.Millisecond)

			call.Response().SetApplicationError()

			err = writeArgs(
				call.Response(),
				[]byte{0x00, 0x00},
				[]byte("such fail"),
			)
			assert.NoError(t, err, "failed to write response")
		}))

	ctx, cancel := context.WithTimeout(context.Background(), 200*internaltesttime.Millisecond)
	defer cancel()

	dialer := &Dialer{
		Caller: "caller",
	}
	require.NoError(t, dialer.Start(ctx))
	defer dialer.Stop(ctx)

	outbound := &Outbound{
		Dialer: dialer,
		Addr:   serverAddr,
	}

	res, resBody, err := outbound.Call(
		ctx,
		&yarpc.Request{
			Caller:    "caller",
			Service:   "service",
			Encoding:  yarpc.Encoding("raw"),
			Procedure: "hello",
		},
		yarpc.NewBufferString("world"),
	)

	if !assert.NoError(t, err, "failed to make call") {
		return
	}

	assert.NotNil(t, res.ApplicationErrorInfo, "application error")
	assert.Equal(t, resBody, yarpc.NewBufferString("such fail"))
}

func TestCallWithoutStarting(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 200*internaltesttime.Millisecond)
	defer cancel()

	dialer := &Dialer{Caller: "caller"}
	outbound := &Outbound{Dialer: dialer, Addr: "127.0.0.1:4040"}

	_, _, err := outbound.Call(
		ctx,
		&yarpc.Request{
			Caller:    "caller",
			Service:   "service",
			Encoding:  yarpc.Encoding("raw"),
			Procedure: "foo",
		},
		yarpc.NewBufferBytes(nil),
	)

	assert.Error(t, err)
}

func TestNoRequest(t *testing.T) {
	outbound := &Outbound{}
	_, _, err := outbound.Call(context.Background(), nil, nil)
	assert.Equal(t, yarpcerror.InvalidArgumentErrorf("request for tchannel outbound was nil"), err)
}
