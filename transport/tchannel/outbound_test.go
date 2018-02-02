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

package tchannel

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uber/tchannel-go"
	"github.com/uber/tchannel-go/testutils"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/encoding/raw"
	"go.uber.org/yarpc/internal/testtime"
	"go.uber.org/yarpc/yarpcerrors"
	"golang.org/x/net/context"
)

func TestOutboundHeaders(t *testing.T) {
	tests := []struct {
		context context.Context
		headers transport.Headers

		wantHeaders []byte
		wantError   string
	}{
		{
			headers: transport.NewHeaders().With("contextfoo", "bar"),
			wantHeaders: []byte{
				0x00, 0x01,
				0x00, 0x0A, 'c', 'o', 'n', 't', 'e', 'x', 't', 'f', 'o', 'o',
				0x00, 0x03, 'b', 'a', 'r',
			},
		},
		{
			headers: transport.NewHeaders().With("Foo", "bar"),
			wantHeaders: []byte{
				0x00, 0x01,
				0x00, 0x03, 'f', 'o', 'o',
				0x00, 0x03, 'b', 'a', 'r',
			},
		},
	}

	for _, tt := range tests {
		server := testutils.NewServer(t, nil)
		defer server.Close()
		hostport := server.PeerInfo().HostPort

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

		x, err := NewChannelTransport(ServiceName("foo"))
		require.NoError(t, err)
		out := x.NewSingleOutbound(hostport)
		require.NoError(t, err)
		require.NoError(t, out.Start(), "failed to start outbound")
		defer out.Stop()

		ctx := tt.context
		if ctx == nil {
			ctx = context.Background()
		}
		ctx, cancel := context.WithTimeout(ctx, testtime.Second)
		defer cancel()

		res, err := out.Call(
			ctx,
			&transport.Request{
				Caller:    "caller",
				Service:   "service",
				Encoding:  raw.Encoding,
				Procedure: "hello",
				Headers:   tt.headers,
				Body:      bytes.NewReader([]byte("world")),
			},
		)
		if tt.wantError != "" {
			if assert.Error(t, err, "expected error") {
				assert.Contains(t, err.Error(), tt.wantError)
			}
		} else {
			if assert.NoError(t, err, "call failed") {
				defer res.Body.Close()
			}
		}
	}
}

func TestCallSuccess(t *testing.T) {
	const (
		headerKey = "foo-BAR-BaZ"
		headerVal = "FooBarBaz"
	)
	server := testutils.NewServer(t, nil)
	defer server.Close()
	serverHostPort := server.PeerInfo().HostPort

	server.GetSubChannel("service").SetHandler(tchannel.HandlerFunc(
		func(ctx context.Context, call *tchannel.InboundCall) {
			assert.Equal(t, "caller", call.CallerName())
			assert.Equal(t, "service", call.ServiceName())
			assert.Equal(t, tchannel.Raw, call.Format())
			assert.Equal(t, "hello", call.MethodString())
			fmt.Println(call)
			headers, body, err := readArgs(call)
			if assert.NoError(t, err, "failed to read request") {
				assert.Equal(t, []byte{
					0x00, 0x01,
					0x00, 0x0b, 'f', 'o', 'o', '-', 'B', 'A', 'R', '-', 'B', 'a', 'Z',
					0x00, 0x09, 'F', 'o', 'o', 'B', 'a', 'r', 'B', 'a', 'z',
				}, headers)
				assert.Equal(t, []byte("world"), body)
			}

			dl, ok := ctx.Deadline()
			assert.True(t, ok, "deadline expected")
			assert.WithinDuration(t, time.Now(), dl, 200*testtime.Millisecond)

			err = writeArgs(call.Response(),
				[]byte{
					0x00, 0x01,
					0x00, 0x03, 'f', 'o', 'o',
					0x00, 0x03, 'b', 'a', 'r',
				}, []byte("great success"))
			assert.NoError(t, err, "failed to write response")
		}))

	x, err := NewTransport(ServiceName("caller"), RawHeader())
	require.NoError(t, err)
	require.NoError(t, x.Start(), "failed to start transport")

	out := x.NewSingleOutbound(serverHostPort)
	require.NoError(t, out.Start(), "failed to start outbound")
	defer out.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), 200*testtime.Millisecond)
	defer cancel()
	res, err := out.Call(
		ctx,
		&transport.Request{
			Caller:    "caller",
			Service:   "service",
			Encoding:  raw.Encoding,
			Procedure: "hello",
			Headers:   transport.NewHeaders().With(headerKey, headerVal),
			Body:      bytes.NewReader([]byte("world")),
		},
	)

	if !assert.NoError(t, err, "failed to make call") {
		return
	}

	assert.Equal(t, false, res.ApplicationError, "not application error")

	foo, ok := res.Headers.Get("foo")
	assert.True(t, ok, "value for foo expected")
	assert.Equal(t, "bar", foo, "foo value mismatch")

	body, err := ioutil.ReadAll(res.Body)
	if assert.NoError(t, err, "failed to read response body") {
		assert.Equal(t, []byte("great success"), body)
	}

	assert.NoError(t, res.Body.Close(), "failed to close response body")
}

func TestCallFailures(t *testing.T) {
	server := testutils.NewServer(t, nil)
	defer server.Close()
	serverHostPort := server.PeerInfo().HostPort

	server.GetSubChannel("service").SetHandler(tchannel.HandlerFunc(
		func(ctx context.Context, call *tchannel.InboundCall) {
			var err error
			if call.MethodString() == "unexpected" {
				err = tchannel.NewSystemError(
					tchannel.ErrCodeUnexpected, "great sadness")
			} else {
				err = tchannel.NewSystemError(
					tchannel.ErrCodeBadRequest, "unknown method")
			}

			call.Response().SendSystemError(err)
		}))

	type testCase struct {
		procedure string
		message   string
	}

	tests := []testCase{
		{
			procedure: "unexpected",
			message:   "great sadness",
		},
		{
			procedure: "not a procedure",
			message:   "unknown method",
		},
	}

	for _, tt := range tests {
		x, err := NewTransport(ServiceName("caller"))
		require.NoError(t, x.Start())
		defer x.Stop()
		require.NoError(t, err)
		out := x.NewSingleOutbound(serverHostPort)
		require.NoError(t, out.Start(), "failed to start outbound")
		defer out.Stop()

		ctx, cancel := context.WithTimeout(context.Background(), 200*testtime.Millisecond)
		defer cancel()
		_, err = out.Call(
			ctx,
			&transport.Request{
				Caller:    "caller",
				Service:   "service",
				Encoding:  raw.Encoding,
				Procedure: tt.procedure,
				Body:      bytes.NewReader([]byte("sup")),
			},
		)

		assert.Error(t, err, "expected failure")
		assert.Contains(t, err.Error(), tt.message)
	}
}

func TestCallError(t *testing.T) {
	server := testutils.NewServer(t, nil)
	defer server.Close()
	serverHostPort := server.PeerInfo().HostPort

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
			assert.WithinDuration(t, time.Now(), dl, 200*testtime.Millisecond)

			call.Response().SetApplicationError()

			err = writeArgs(
				call.Response(),
				[]byte{0x00, 0x00},
				[]byte("such fail"),
			)
			assert.NoError(t, err, "failed to write response")
		}))

	x, err := NewTransport(ServiceName("caller"))
	require.NoError(t, err)
	require.NoError(t, x.Start(), "failed to start transport")

	out := x.NewSingleOutbound(serverHostPort)
	require.NoError(t, out.Start(), "failed to start outbound")
	defer out.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), 200*testtime.Millisecond)
	defer cancel()
	res, err := out.Call(
		ctx,
		&transport.Request{
			Caller:    "caller",
			Service:   "service",
			Encoding:  raw.Encoding,
			Procedure: "hello",
			Body:      bytes.NewReader([]byte("world")),
		},
	)

	if !assert.NoError(t, err, "failed to make call") {
		return
	}

	assert.Equal(t, true, res.ApplicationError, "application error")

	body, err := ioutil.ReadAll(res.Body)
	if assert.NoError(t, err, "failed to read response body") {
		assert.Equal(t, []byte("such fail"), body)
	}

	assert.NoError(t, res.Body.Close(), "failed to close response body")
}

func TestStartMultiple(t *testing.T) {
	serverHostPort := "localhost:4040"
	x, err := NewTransport(ServiceName("caller"))
	require.NoError(t, err)
	out := x.NewSingleOutbound(serverHostPort)
	require.NoError(t, err)
	// TODO: If we change Start() to establish a connection to the host, this
	// hostport will have to be changed to a real server.

	var wg sync.WaitGroup
	signal := make(chan struct{})

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-signal

			err := out.Start()
			assert.NoError(t, err)
		}()
	}
	close(signal)
	wg.Wait()
}

func TestStopMultiple(t *testing.T) {
	serverHostPort := "localhost:4040"
	x, err := NewTransport(ServiceName("caller"))
	require.NoError(t, err)
	out := x.NewSingleOutbound(serverHostPort)
	require.NoError(t, err)
	// TODO: If we change Start() to establish a connection to the host, this
	// hostport will have to be changed to a real server.

	require.NoError(t, out.Start())

	var wg sync.WaitGroup
	signal := make(chan struct{})

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-signal

			err := out.Stop()
			assert.NoError(t, err)
		}()
	}
	close(signal)
	wg.Wait()
}

func TestCallWithoutStarting(t *testing.T) {
	serverHostPort := "localhost:4040"
	x, err := NewTransport(ServiceName("caller"))
	require.NoError(t, err)
	out := x.NewSingleOutbound(serverHostPort)
	require.NoError(t, err)
	// TODO: If we change Start() to establish a connection to the host, this
	// hostport will have to be changed to a real server.

	ctx, cancel := context.WithTimeout(context.Background(), 200*testtime.Millisecond)
	defer cancel()
	_, err = out.Call(
		ctx,
		&transport.Request{
			Caller:    "caller",
			Service:   "service",
			Encoding:  raw.Encoding,
			Procedure: "foo",
			Body:      bytes.NewReader([]byte("sup")),
		},
	)

	assert.Equal(t, yarpcerrors.FailedPreconditionErrorf("error waiting for tchannel outbound to start for service: service: context finished while waiting for instance to start: context deadline exceeded"), err)
}

func TestNoRequest(t *testing.T) {
	tran, err := NewTransport(ServiceName("caller"))
	require.NoError(t, err)
	out := tran.NewSingleOutbound("localhost:0")

	_, err = out.Call(context.Background(), nil)
	assert.Equal(t, yarpcerrors.InvalidArgumentErrorf("request for tchannel outbound was nil"), err)
}
