// Copyright (c) 2025 Uber Technologies, Inc.
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

	"go.uber.org/yarpc/api/peer"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/internal/integrationtest"
	"go.uber.org/yarpc/peer/hostport"
	"go.uber.org/yarpc/transport/http"
)

var http1spec = integrationtest.TransportSpec{
	Identify: hostport.Identify,
	NewServerTransport: func(t *testing.T, addr string) peer.Transport {
		return http.NewTransport()
	},
	NewClientTransport: func(t *testing.T) peer.Transport {
		return http.NewTransport()
	},
	NewUnaryOutbound: func(x peer.Transport, pc peer.Chooser) transport.UnaryOutbound {
		return x.(*http.Transport).NewOutbound(pc)
	},
	NewInbound: func(x peer.Transport, addr string) transport.Inbound {
		// disable http2
		return x.(*http.Transport).NewInbound(addr, http.DisableHTTP2(true))
	},
	Addr: func(x peer.Transport, ib transport.Inbound) string {
		return ib.(*http.Inbound).Addr().String()
	},
}

func TestIntegrationWithHTTP(t *testing.T) {
	http1spec.Test(t)
}

var http2spec = integrationtest.TransportSpec{
	Identify: hostport.Identify,
	NewServerTransport: func(t *testing.T, addr string) peer.Transport {
		return http.NewTransport()
	},
	NewClientTransport: func(t *testing.T) peer.Transport {
		// TODO: will update this once we add client support
		return http.NewTransport()
	},
	NewUnaryOutbound: func(x peer.Transport, pc peer.Chooser) transport.UnaryOutbound {
		return x.(*http.Transport).NewOutbound(pc)
	},
	NewInbound: func(x peer.Transport, addr string) transport.Inbound {
		// we don't disable http2
		return x.(*http.Transport).NewInbound(addr)
	},
	Addr: func(x peer.Transport, ib transport.Inbound) string {
		return ib.(*http.Inbound).Addr().String()
	},
}

func TestIntegrationWithHTTP2(t *testing.T) {
	http2spec.Test(t)
}
