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

package tchannel

import (
	"bytes"
	"io/ioutil"
	"testing"
	"time"

	"go.uber.org/yarpc/encoding/raw"
	iout "go.uber.org/yarpc/internal/outbound"
	"go.uber.org/yarpc/transport"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uber/tchannel-go"
	"github.com/uber/tchannel-go/testutils"
	"golang.org/x/net/context"
)

// Different ways in which outbounds can be constructed from a client Channel
// and a hostPort
var newOutbounds = []func(*tchannel.Channel, string) transport.Outbound{
	func(ch *tchannel.Channel, hostPort string) transport.Outbound {
		ch.Peers().Add(hostPort)
		return NewOutbound(ch)
	},
	func(ch *tchannel.Channel, hostPort string) transport.Outbound {
		return NewOutbound(ch, HostPort(hostPort))
	},
}

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

		for _, getOutbound := range newOutbounds {
			out := getOutbound(testutils.NewClient(t, &testutils.ChannelOpts{
				ServiceName: "caller",
			}), hostport)
			require.NoError(t, out.Start(transport.NoDeps), "failed to start outbound")
			defer out.Stop()

			ctx := tt.context
			if ctx == nil {
				ctx = context.Background()
			}
			ctx, _ = context.WithTimeout(ctx, time.Second)

			res, err := out.Call(
				ctx,
				iout.CallFromRequest(
					&transport.Request{
						Caller:    "caller",
						Service:   "service",
						Encoding:  raw.Encoding,
						Procedure: "hello",
						Headers:   tt.headers,
						Body:      bytes.NewReader([]byte("world")),
					}),
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
}

func TestCallSuccess(t *testing.T) {
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
			assert.WithinDuration(t, time.Now(), dl, 200*time.Millisecond)

			err = writeArgs(call.Response(),
				[]byte{
					0x00, 0x01,
					0x00, 0x03, 'f', 'o', 'o',
					0x00, 0x03, 'b', 'a', 'r',
				}, []byte("great success"))
			assert.NoError(t, err, "failed to write response")
		}))

	for _, getOutbound := range newOutbounds {
		out := getOutbound(testutils.NewClient(t, &testutils.ChannelOpts{
			ServiceName: "caller",
		}), serverHostPort)
		require.NoError(t, out.Start(transport.NoDeps), "failed to start outbound")
		defer out.Stop()

		ctx, _ := context.WithTimeout(context.Background(), 200*time.Millisecond)
		res, err := out.Call(
			ctx,
			iout.CallFromRequest(&transport.Request{
				Caller:    "caller",
				Service:   "service",
				Encoding:  raw.Encoding,
				Procedure: "hello",
				Body:      bytes.NewReader([]byte("world")),
			}),
		)

		if !assert.NoError(t, err, "failed to make call") {
			continue
		}

		foo, ok := res.Headers.Get("foo")
		assert.True(t, ok, "value for foo expected")
		assert.Equal(t, "bar", foo, "foo value mismatch")

		body, err := ioutil.ReadAll(res.Body)
		if assert.NoError(t, err, "failed to read response body") {
			assert.Equal(t, []byte("great success"), body)
		}

		assert.NoError(t, res.Body.Close(), "failed to close response body")
	}
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
		procedure   string
		getOutbound func(*tchannel.Channel, string) transport.Outbound
		message     string
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

	// cross product with newOutbounds
	newTests := make([]testCase, 0, len(tests)*len(newOutbounds))
	for _, tt := range tests {
		for _, getOutbound := range newOutbounds {
			tt.getOutbound = getOutbound
			newTests = append(newTests, tt)
		}
	}
	tests = newTests

	for _, tt := range tests {
		out := tt.getOutbound(testutils.NewClient(t, nil), serverHostPort)
		require.NoError(t, out.Start(transport.NoDeps), "failed to start outbound")
		defer out.Stop()

		ctx, _ := context.WithTimeout(context.Background(), 200*time.Millisecond)
		_, err := out.Call(
			ctx,
			iout.CallFromRequest(&transport.Request{
				Caller:    "caller",
				Service:   "service",
				Encoding:  raw.Encoding,
				Procedure: tt.procedure,
				Body:      bytes.NewReader([]byte("sup")),
			}),
		)

		assert.Error(t, err, "expected failure")
		assert.Contains(t, err.Error(), tt.message)
	}
}

func TestStartTwice(t *testing.T) {
	for _, getOutbound := range newOutbounds {
		out := getOutbound(testutils.NewClient(t, &testutils.ChannelOpts{
			ServiceName: "caller",
		}), "localhost:4040")
		// TODO: If we change Start() to establish a connection to the host, this
		// hostport will have to be changed to a real server.

		if assert.NoError(t, out.Start(transport.NoDeps)) {
			err := out.Start(transport.NoDeps)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "tchannel.Outbound has already been started")
		}
	}
}

func TestStopWithoutStarting(t *testing.T) {
	for _, getOutbound := range newOutbounds {
		out := getOutbound(testutils.NewClient(t, &testutils.ChannelOpts{
			ServiceName: "caller",
		}), "localhost:4040")
		// TODO: If we change Start() to establish a connection to the host, this
		// hostport will have to be changed to a real server.

		err := out.Stop()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "tchannel.Outbound has not been started")
	}
}

func TestCallWithoutStarting(t *testing.T) {
	for _, getOutbound := range newOutbounds {
		out := getOutbound(testutils.NewClient(t, &testutils.ChannelOpts{
			ServiceName: "caller",
		}), "localhost:4040")
		// TODO: If we change Start() to establish a connection to the host, this
		// hostport will have to be changed to a real server.

		assert.Panics(t, func() {
			ctx, _ := context.WithTimeout(context.Background(), 200*time.Millisecond)
			out.Call(
				ctx,
				iout.CallFromRequest(&transport.Request{
					Caller:    "caller",
					Service:   "service",
					Encoding:  raw.Encoding,
					Procedure: "foo",
					Body:      bytes.NewReader([]byte("sup")),
				}),
			)
		})
	}
}
