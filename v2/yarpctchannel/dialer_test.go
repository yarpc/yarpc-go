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

package yarpctchannel_test

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	yarpc "go.uber.org/yarpc/v2"
	"go.uber.org/yarpc/v2/internal/internaltesttime"
	"go.uber.org/yarpc/v2/yarpcbackoff"
	"go.uber.org/yarpc/v2/yarpcerror"
	"go.uber.org/yarpc/v2/yarpcjson"
	"go.uber.org/yarpc/v2/yarpcroundrobin"
	"go.uber.org/yarpc/v2/yarpcrouter"
	"go.uber.org/yarpc/v2/yarpctchannel"
)

func TestDialerNotStarted(t *testing.T) {
	dialer := &yarpctchannel.Dialer{}
	peer, err := dialer.RetainPeer(yarpc.Address("127.0.0.1:0"), yarpc.NopSubscriber)
	require.Nil(t, peer)
	require.Error(t, err)
}

func TestReleaseBeforeRetain(t *testing.T) {
	dialer := &yarpctchannel.Dialer{}
	err := dialer.ReleasePeer(yarpc.Address("127.0.0.1:0"), yarpc.NopSubscriber)
	require.Error(t, err)
}

func TestDialerBasics(t *testing.T) {
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()

	type Payload struct {
		Note string
	}

	handleEcho := yarpc.EncodingToTransportProcedures(
		yarpcjson.Procedure("echo", func(ctx context.Context, req *Payload) (*Payload, error) {
			t.Logf("handle echo\n")
			return req, nil
		}),
	)

	router := yarpcrouter.NewMapRouter("service", handleEcho)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	inbound := &yarpctchannel.Inbound{
		Service:  "service",
		Listener: listener,
		Router:   router,
	}
	require.NoError(t, inbound.Start(ctx))
	defer inbound.Stop(ctx)
	t.Logf("inbound started %s\n", listener.Addr())

	backoff, err := yarpcbackoff.NewExponential(yarpcbackoff.FirstBackoff(time.Nanosecond))
	require.NoError(t, err)
	dialer := &yarpctchannel.Dialer{
		Caller:      "caller",
		ConnBackoff: backoff,
	}
	require.NoError(t, dialer.Start(ctx))
	defer dialer.Stop(ctx)
	t.Logf("dialer started\n")

	outbound := &yarpctchannel.Outbound{
		Dialer: dialer,
		Addr:   listener.Addr().String(),
	}

	client := yarpcjson.New(yarpc.Client{
		Caller:  "caller",
		Service: "service",
		Unary:   outbound,
	})

	t.Logf("calling\n")
	req := &Payload{Note: "forthcoming"}
	res := &Payload{}
	errDetails := &Payload{}
	err = client.Call(ctx, "echo", req, res, errDetails)
	require.NoError(t, err)
	require.Equal(t, req, res)
}

func TestDialerBellsAndWhistles(t *testing.T) {
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()

	type Payload struct {
		Note string
	}

	handleEcho := yarpc.EncodingToTransportProcedures(
		yarpcjson.Procedure("echo", func(ctx context.Context, req *Payload) (*Payload, error) {
			t.Logf("handle echo\n")
			// This time echoing a header.
			call := yarpc.CallFromContext(ctx)
			call.WriteResponseHeader("header", call.Header("header"))
			return req, nil
		}),
	)

	router := yarpcrouter.NewMapRouter("service", handleEcho)

	inbound := &yarpctchannel.Inbound{
		Service: "service",
		Addr:    "127.0.0.1:0",
		Router:  router,
	}
	require.NoError(t, inbound.Start(ctx))
	defer inbound.Stop(ctx)
	t.Logf("inbound started %s\n", inbound.Listener.Addr())

	dialer := &yarpctchannel.Dialer{
		Caller: "caller",
	}
	require.NoError(t, dialer.Start(ctx))
	defer dialer.Stop(ctx)
	t.Logf("dialer started\n")

	// This time using a peer list instead of using the dialer directly.
	peerlist := yarpcroundrobin.New("roundrobin", dialer)
	peerlist.Update(yarpc.ListUpdates{
		Additions: []yarpc.Identifier{
			yarpc.Address(inbound.Listener.Addr().String()),
		},
	})

	outbound := &yarpctchannel.Outbound{
		Chooser: peerlist,
		Addr:    inbound.Listener.Addr().String(),
	}

	client := yarpcjson.New(yarpc.Client{
		Caller:  "caller",
		Service: "service",
		Unary:   outbound,
	})

	t.Logf("calling\n")
	req := &Payload{Note: "forthcoming"}
	res := &Payload{}
	errDetails := &Payload{}
	var headers map[string]string
	err := client.Call(
		ctx,
		"echo",
		req,
		res,
		errDetails,
		yarpc.WithHeader("HeAdEr", "forthcoming"),
		yarpc.ResponseHeaders(&headers),
	)
	require.NoError(t, err)
	assert.Equal(t, req, res)
	assert.Equal(t, "forthcoming", headers["header"])
}

func TestPeerListChanges(t *testing.T) {
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()

	type Payload struct {
		Note string
	}

	handleEcho := yarpc.EncodingToTransportProcedures(
		yarpcjson.Procedure("echo", func(ctx context.Context, req *Payload) (*Payload, error) {
			t.Logf("handle echo\n")
			// This time echoing a header.
			call := yarpc.CallFromContext(ctx)
			call.WriteResponseHeader("header", call.Header("header"))
			return req, nil
		}),
	)

	router := yarpcrouter.NewMapRouter("service", handleEcho)

	inbound := &yarpctchannel.Inbound{
		Service: "service",
		Addr:    "127.0.0.1:0",
		Router:  router,
	}
	require.NoError(t, inbound.Start(ctx))
	defer inbound.Stop(ctx)
	t.Logf("inbound started %s\n", inbound.Listener.Addr())

	dialer := &yarpctchannel.Dialer{
		Caller: "caller",
	}
	require.NoError(t, dialer.Start(ctx))
	defer dialer.Stop(ctx)
	t.Logf("dialer started\n")

	// Retain with multiple peer lists.
	avery := yarpcroundrobin.New("avery", dialer)
	blake := yarpcroundrobin.New("blake", dialer)

	avery.Update(yarpc.ListUpdates{
		Additions: []yarpc.Identifier{
			yarpc.Address(inbound.Listener.Addr().String()),
		},
	})

	blake.Update(yarpc.ListUpdates{
		Additions: []yarpc.Identifier{
			yarpc.Address(inbound.Listener.Addr().String()),
		},
	})

	avery.Update(yarpc.ListUpdates{
		Removals: []yarpc.Identifier{
			yarpc.Address(inbound.Listener.Addr().String()),
		},
	})

	// Leave blake's reference alone to exercise disconnect in dialer stop.
}

func TestConnectionFailure(t *testing.T) {
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, 100*internaltesttime.Millisecond)
	defer cancel()

	type Payload struct {
		Note string
	}

	// Listener accepts but never completes a TChannel handshake,
	// so never becomes available.
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	dialer := &yarpctchannel.Dialer{
		Caller: "caller",
	}
	require.NoError(t, dialer.Start(ctx))
	defer dialer.Stop(ctx)
	t.Logf("dialer started\n")

	outbound := &yarpctchannel.Outbound{
		Dialer: dialer,
		Addr:   listener.Addr().String(),
	}

	client := yarpcjson.New(yarpc.Client{
		Caller:  "caller",
		Service: "service",
		Unary:   outbound,
	})

	t.Logf("calling\n")
	req := &Payload{Note: "forthcoming"}
	res := &Payload{}
	errDetails := &Payload{}
	err = client.Call(ctx, "echo", req, res, errDetails)
	assert.Equal(t, yarpcerror.New(yarpcerror.CodeDeadlineExceeded, "timeout"), err)
}

func TestErrors(t *testing.T) {
	tests := map[string]struct {
		procedure  string
		give, want error
	}{
		"implicit system error": {
			procedure: "echo",
			give:      fmt.Errorf("system error"),
			want:      yarpcerror.New(yarpcerror.CodeUnknown, `system error`),
		},
		"explicit system error": {
			procedure: "echo",
			give:      yarpcerror.New(yarpcerror.CodeUnknown, `error for service "service" and procedure "echo": system error`),
			want:      yarpcerror.New(yarpcerror.CodeUnknown, `error for service "service" and procedure "echo": system error`),
		},
		"unimplemented": {
			procedure: "bogus",
			want:      yarpcerror.New(yarpcerror.CodeInvalidArgument, `unrecognized procedure "bogus" for service "service"`),
		},
		// This case verifies that TChannel "black holes" resource exhausted
		// errors, inducing a client side timeout.
		// This is an unfortunate but necessary behavior since TChannel clients
		// across languages do a poor job of retry backoff.
		"resource exhausted": {
			procedure: "echo",
			give:      yarpcerror.New(yarpcerror.CodeResourceExhausted, "no response for you"),
			want:      yarpcerror.New(yarpcerror.CodeDeadlineExceeded, "timeout"),
		},
	}

	for desc, tt := range tests {
		t.Run(desc, func(t *testing.T) {
			ctx := context.Background()
			ctx, cancel := context.WithTimeout(ctx, 40*internaltesttime.Millisecond)
			defer cancel()

			type Payload struct {
				Note string
			}

			handleEcho := yarpc.EncodingToTransportProcedures(
				yarpcjson.Procedure("echo", func(ctx context.Context, req *Payload) (*Payload, error) {
					return nil, tt.give
				}),
			)

			router := yarpcrouter.NewMapRouter("service", handleEcho)

			listener, err := net.Listen("tcp", "127.0.0.1:0")
			require.NoError(t, err)

			inbound := &yarpctchannel.Inbound{
				Service:  "service",
				Listener: listener,
				Router:   router,
			}
			require.NoError(t, inbound.Start(ctx))
			defer inbound.Stop(ctx)

			backoff, err := yarpcbackoff.NewExponential(yarpcbackoff.FirstBackoff(time.Nanosecond))
			require.NoError(t, err)
			dialer := &yarpctchannel.Dialer{
				Caller:      "caller",
				ConnBackoff: backoff,
			}
			require.NoError(t, dialer.Start(ctx))
			defer dialer.Stop(ctx)

			outbound := &yarpctchannel.Outbound{
				Dialer: dialer,
				Addr:   listener.Addr().String(),
			}

			client := yarpcjson.New(yarpc.Client{
				Caller:  "caller",
				Service: "service",
				Unary:   outbound,
			})

			req := &Payload{Note: "forthcoming"}
			res := &Payload{}
			errDetails := &Payload{}
			err = client.Call(ctx, tt.procedure, req, res, errDetails)
			require.Equal(t, tt.want, err)
		})
	}
}
