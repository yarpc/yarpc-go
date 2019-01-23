// Copyright (c) 2019 Uber Technologies, Inc.
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

package integrationtest_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc/api/backoff"
	"go.uber.org/yarpc/api/peer"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/internal/integrationtest"
	"go.uber.org/yarpc/internal/testtime"
	"go.uber.org/yarpc/internal/yarpctest"
	"go.uber.org/yarpc/peer/hostport"
	"go.uber.org/yarpc/transport/tchannel"
)

var spec = integrationtest.TransportSpec{
	Identify: hostport.Identify,
	NewServerTransport: func(t *testing.T, addr string) peer.Transport {
		x, err := tchannel.NewTransport(
			tchannel.ServiceName("service"),
			tchannel.ListenAddr(addr),
		)
		require.NoError(t, err, "must construct transport")
		return x
	},
	NewInbound: func(x peer.Transport, addr string) transport.Inbound {
		return x.(*tchannel.Transport).NewInbound()
	},
	NewClientTransport: func(t *testing.T) peer.Transport {
		x, err := tchannel.NewTransport(
			tchannel.ServiceName("client"),
			tchannel.ConnTimeout(10*testtime.Millisecond),
			tchannel.ConnBackoff(backoff.None),
		)
		require.NoError(t, err, "must construct transport")
		return x
	},
	NewUnaryOutbound: func(x peer.Transport, pc peer.Chooser) transport.UnaryOutbound {
		return x.(*tchannel.Transport).NewOutbound(pc)
	},
	Addr: func(x peer.Transport, ib transport.Inbound) string {
		return yarpctest.ZeroAddrStringToHostPort(x.(*tchannel.Transport).ListenAddr())
	},
}

func TestIntegrationWithTChannel(t *testing.T) {
	spec.Test(t)
}
