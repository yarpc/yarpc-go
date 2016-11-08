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

package dispatcher

import (
	"fmt"

	"go.uber.org/yarpc"
	"go.uber.org/yarpc/crossdock/client/params"
	"go.uber.org/yarpc/transport"
	ht "go.uber.org/yarpc/transport/http"
	tch "go.uber.org/yarpc/transport/tchannel"

	"github.com/crossdock/crossdock-go"
	"github.com/uber/tchannel-go"
)

// Create creates an RPC from the given parameters or fails the whole behavior.
func Create(t crossdock.T) yarpc.Dispatcher {
	fatals := crossdock.Fatals(t)

	server := t.Param(params.Server)
	fatals.NotEmpty(server, "server is required")

	var unaryOutbound transport.UnaryOutbound
	trans := t.Param(params.Transport)
	switch trans {
	case "http":
		unaryOutbound = ht.NewOutbound(fmt.Sprintf("http://%s:8081", server))
	case "tchannel":
		ch, err := tchannel.NewChannel("client", nil)
		fatals.NoError(err, "couldn't create tchannel")
		unaryOutbound = tch.NewOutbound(ch, tch.HostPort(server+":8082"))
	default:
		fatals.Fail("", "unknown transport %q", trans)
	}

	return yarpc.NewDispatcher(yarpc.Config{
		Name:      "client",
		Outbounds: transport.Outbounds{"yarpc-test": unaryOutbound},
	})
}
