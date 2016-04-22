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

package echo

import (
	"fmt"
	"net/http"

	"github.com/yarpc/yarpc-go"
	"github.com/yarpc/yarpc-go/crossdock/client/behavior"
	"github.com/yarpc/yarpc-go/transport"
	ht "github.com/yarpc/yarpc-go/transport/http"
	tch "github.com/yarpc/yarpc-go/transport/tchannel"

	"github.com/uber/tchannel-go"
)

// echoEntry is an entry emitted by the echo behaviors.
type echoEntry struct {
	behavior.Entry

	Transport string `json:"transport"`
	Encoding  string `json:"encoding"`
	Server    string `json:"server"`
}

// echoSink wraps a sink to emit echoEntry entries instead.
type echoSink struct {
	behavior.Sink

	Transport string
	Encoding  string
	Server    string
}

func (s echoSink) Put(e interface{}) {
	s.Sink.Put(echoEntry{
		Entry:     e.(behavior.Entry),
		Transport: s.Transport,
		Encoding:  s.Encoding,
		Server:    s.Server,
	})
}

// createEchoSink wraps a Sink to have transport, encoding, and server
// information.
func createEchoSink(encoding string, s behavior.Sink, p behavior.Params) behavior.Sink {
	return echoSink{
		Sink:      s,
		Transport: p.Param(TransportParam),
		Encoding:  encoding,
		Server:    p.Param(ServerParam),
	}
}

// createRPC creates an RPC from the given parameters or fails the whole
// behavior.
func createRPC(s behavior.Sink, p behavior.Params) yarpc.RPC {
	fatals := behavior.Fatals(s)

	server := p.Param(ServerParam)
	fatals.NotEmpty(server, "server is required")

	var outbound transport.Outbound
	trans := p.Param(TransportParam)
	switch trans {
	case "http":
		// Go HTTP servers have keep-alive enabled by default. If we re-use
		// HTTP clients, the same connection will be used to make requests.
		// This is undesirable during tests because we want to isolate the
		// different test requests. Additionally, keep-alive causes the test
		// server to continue listening on the existing connection for some
		// time after we close the listener.
		cl := &http.Client{Transport: new(http.Transport)}
		outbound = ht.NewOutboundWithClient(fmt.Sprintf("http://%s:8081", server), cl)
	case "tchannel":
		ch, err := tchannel.NewChannel("client", nil)
		fatals.NoError(err, "couldn't create tchannel")
		outbound = tch.NewOutbound(ch, tch.HostPort(server+":8082"))
	default:
		fatals.True(false, "unknown transport %q", trans)
	}

	return yarpc.New(yarpc.Config{
		Name:      "client",
		Outbounds: transport.Outbounds{"yarpc-test": outbound},
	})
}
