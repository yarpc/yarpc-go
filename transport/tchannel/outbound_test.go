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
		name            string
		originalHeaders bool
		giveHeaders     map[string]string
		wantHeaders     map[string]string
	}{
		{
			name: "exactCaseHeader options on",
			giveHeaders: map[string]string{
				"foo-BAR-BaZ": "PiE",
				"foo-bar":     "LEMON",
				"BAR-BAZ":     "orange",
			},
			wantHeaders: map[string]string{
				"foo-BAR-BaZ": "PiE",
				"foo-bar":     "LEMON",
				"BAR-BAZ":     "orange",
			},
			originalHeaders: true,
		},
		{
			name: "exactCaseHeader options off",
			giveHeaders: map[string]string{
				"foo-BAR-BaZ": "PiE",
				"foo-bar":     "LEMON",
				"BAR-BAZ":     "orange",
			},
			wantHeaders: map[string]string{
				"foo-bar-baz": "PiE",
				"foo-bar":     "LEMON",
				"bar-baz":     "orange",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var handlerInvoked bool
			server := testutils.NewServer(t, nil)
			defer server.Close()
			serverHostPort := server.PeerInfo().HostPort

			server.GetSubChannel("service").SetHandler(tchannel.HandlerFunc(
				func(ctx context.Context, call *tchannel.InboundCall) {
					handlerInvoked = true
					headers, err := readHeaders(tchannel.Raw, call.Arg2Reader)
					if !assert.NoError(t, err, "failed to read request") {
						return
					}

					deleteReservedHeaders(headers)
					assert.Equal(t, tt.wantHeaders, headers.OriginalItems(), "headers did not match")

					// write a response
					err = writeArgs(call.Response(), []byte{0x00, 0x00}, []byte(""))
					assert.NoError(t, err, "failed to write response")
				}))

			opts := []TransportOption{ServiceName("caller")}
			if tt.originalHeaders {
				opts = append(opts, OriginalHeaders())
			}

			trans, err := NewTransport(opts...)
			require.NoError(t, err)
			require.NoError(t, trans.Start(), "failed to start transport")

			out := trans.NewSingleOutbound(serverHostPort)
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
					Procedure: "hello",
					Headers:   transport.HeadersFromMap(tt.giveHeaders),
					Body:      bytes.NewBufferString("body"),
				},
			)

			require.NoError(t, err, "failed to make call")
			assert.True(t, handlerInvoked, "handler was never called by client")
		})
	}
}

func TestCallSuccess(t *testing.T) {
	var handlerInvoked bool
	server := testutils.NewServer(t, nil)
	defer server.Close()
	serverHostPort := server.PeerInfo().HostPort

	server.GetSubChannel("service").SetHandler(tchannel.HandlerFunc(
		func(ctx context.Context, call *tchannel.InboundCall) {
			handlerInvoked = true

			assert.Equal(t, "caller", call.CallerName())
			assert.Equal(t, "service", call.ServiceName())
			assert.Equal(t, tchannel.Raw, call.Format())
			assert.Equal(t, "hello", call.MethodString())
			_, body, err := readArgs(call)
			if assert.NoError(t, err, "failed to read request") {
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

	out := newSingleOutbound(t, serverHostPort)
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
			Body:      bytes.NewBufferString("world"),
		},
	)

	require.NoError(t, err, "failed to make call")
	require.False(t, res.ApplicationError, "unexpected application error")

	foo, ok := res.Headers.Get("foo")
	assert.True(t, ok, "value for foo expected")
	assert.Equal(t, "bar", foo, "foo value mismatch")

	body, err := ioutil.ReadAll(res.Body)
	if assert.NoError(t, err, "failed to read response body") {
		assert.Equal(t, []byte("great success"), body)
	}

	assert.NoError(t, res.Body.Close(), "failed to close response body")
	assert.True(t, handlerInvoked, "handler was never called by client")
}

func TestCallFailures(t *testing.T) {
	const (
		unexpectedMethod = "unexpected"
		unknownMethod    = "unknown"
	)

	server := testutils.NewServer(t, nil)
	defer server.Close()
	serverHostPort := server.PeerInfo().HostPort

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
		desc      string
		procedure string
		message   string
	}

	tests := []testCase{
		{
			desc:      "unexpected error",
			procedure: unexpectedMethod,
			message:   "great sadness",
		},
		{
			desc:      "missing procedure error",
			procedure: unknownMethod,
			message:   "unknown method",
		},
		{
			desc:      "service name mismatch error",
			procedure: "wrong service name",
			message:   "does not match",
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {

			out := newSingleOutbound(t, serverHostPort)
			require.NoError(t, out.Start(), "failed to start outbound")
			defer out.Stop()

			ctx, cancel := context.WithTimeout(context.Background(), 200*testtime.Millisecond)
			defer cancel()
			_, err := out.Call(
				ctx,
				&transport.Request{
					Caller:    "caller",
					Service:   "service",
					Encoding:  raw.Encoding,
					Procedure: tt.procedure,
					Body:      bytes.NewReader([]byte("sup")),
				},
			)

			require.Error(t, err, "expected failure")
			assert.Contains(t, err.Error(), tt.message)
		})
	}
}

func TestApplicationError(t *testing.T) {
	server := testutils.NewServer(t, nil)
	defer server.Close()
	serverHostPort := server.PeerInfo().HostPort

	server.GetSubChannel("service").SetHandler(tchannel.HandlerFunc(
		func(ctx context.Context, call *tchannel.InboundCall) {
			call.Response().SetApplicationError()

			err := writeArgs(call.Response(), []byte{0x00, 0x00}, []byte("foo"))
			assert.NoError(t, err, "failed to write response")
		}))

	out := newSingleOutbound(t, serverHostPort)
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
			Body:      &bytes.Buffer{},
		},
	)
	require.NoError(t, err, "failed to make call")
	assert.True(t, res.ApplicationError, "application error was not set")

}

func TestStartMultiple(t *testing.T) {
	out := newSingleOutbound(t, "localhost:4040")
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
	out := newSingleOutbound(t, "localhost:4040")
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
	out := newSingleOutbound(t, "localhost:4040")
	ctx, cancel := context.WithTimeout(context.Background(), 200*testtime.Millisecond)
	defer cancel()
	_, err := out.Call(
		ctx,
		&transport.Request{
			Caller:    "caller",
			Service:   "service",
			Encoding:  raw.Encoding,
			Procedure: "foo",
			Body:      bytes.NewReader([]byte("sup")),
		},
	)

	wantErr := yarpcerrors.FailedPreconditionErrorf("error waiting for tchannel outbound to start for service: service: context finished while waiting for instance to start: context deadline exceeded")
	assert.EqualError(t, err, wantErr.Error())

}

func TestOutboundNoRequest(t *testing.T) {
	out := newSingleOutbound(t, "localhost:4040")
	_, err := out.Call(context.Background(), nil)
	wantErr := yarpcerrors.InvalidArgumentErrorf("request for tchannel outbound was nil")
	assert.EqualError(t, err, wantErr.Error())
}

func newSingleOutbound(t *testing.T, serverAddr string) transport.UnaryOutbound {
	trans, err := NewTransport(ServiceName("caller"))
	require.NoError(t, err)
	require.NoError(t, trans.Start())
	return trans.NewSingleOutbound(serverAddr)
}
