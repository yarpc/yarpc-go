// Copyright (c) 2022 Uber Technologies, Inc.
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
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uber/tchannel-go"
	"github.com/uber/tchannel-go/testutils"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/encoding/raw"
	"go.uber.org/yarpc/internal/testtime"
	"go.uber.org/yarpc/peer"
	"go.uber.org/yarpc/peer/hostport"
	"golang.org/x/net/context"
)

func TestOutboundChannel(t *testing.T) {
	server := testutils.NewServer(t, nil)
	defer server.Close()
	serverHostPort := server.PeerInfo().HostPort

	var handlerInvoked bool
	server.GetSubChannel("service").SetHandler(tchannel.HandlerFunc(
		func(ctx context.Context, call *tchannel.InboundCall) {
			handlerInvoked = true
			_, err := readHeaders(tchannel.Raw, call.Arg2Reader)
			if !assert.NoError(t, err, "failed to read request") {
				return
			}

			// write a response
			err = writeArgs(call.Response(), []byte{0x00, 0x00}, []byte(""))
			assert.NoError(t, err, "failed to write response")
		}),
	)

	opts := []TransportOption{ServiceName("caller")}
	trans, err := NewTransport(opts...)
	require.NoError(t, err)

	var dialerInvoked bool
	dialerFunc := func(ctx context.Context, network, hostPort string) (net.Conn, error) {
		dialerInvoked = true
		return (&net.Dialer{}).DialContext(ctx, network, hostPort)
	}
	outboundChannel := trans.createOutboundChannel(dialerFunc)

	require.NoError(t, trans.Start(), "failed to start transport")
	defer trans.Stop()

	out := trans.NewOutbound(peer.NewSingle(hostport.PeerIdentifier(serverHostPort), outboundChannel))
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
			Body:      bytes.NewBufferString("body"),
		},
	)
	require.NoError(t, err, "failed to make call")
	assert.True(t, handlerInvoked, "handler was never called by client")
	assert.True(t, dialerInvoked, "dialer was not called")
}
