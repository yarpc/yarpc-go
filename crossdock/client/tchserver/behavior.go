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

package tchserver

import (
	"fmt"

	"go.uber.org/yarpc"
	"go.uber.org/yarpc/crossdock/client/params"
	"go.uber.org/yarpc/transport"
	tch "go.uber.org/yarpc/transport/tchannel"

	"github.com/crossdock/crossdock-go"
	"github.com/uber/tchannel-go"
)

const (
	serverPort = 8083
	serverName = "tchannel-server"
)

// Run exercises a YARPC client against a tchannel server.
func Run(t crossdock.T) {
	fatals := crossdock.Fatals(t)

	encoding := t.Param(params.Encoding)
	server := t.Param(params.Server)
	serverHostPort := fmt.Sprintf("%v:%v", server, serverPort)

	ch, err := tchannel.NewChannel("yarpc-client", nil)
	fatals.NoError(err, "could not create channel")

	dispatcher := yarpc.NewDispatcher(yarpc.Config{
		Name: "yarpc-client",
		Outbounds: transport.Outbounds{
			serverName: tch.NewOutbound(ch, tch.HostPort(serverHostPort)),
		},
	})
	fatals.NoError(dispatcher.Start(), "could not start Dispatcher")
	defer dispatcher.Stop()

	switch encoding {
	case "raw":
		runRaw(t, dispatcher)
	case "json":
		runJSON(t, dispatcher)
	case "thrift":
		runThrift(t, dispatcher)
	default:
		fatals.Fail("", "unknown encoding %q", encoding)
	}
}
