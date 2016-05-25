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

	"github.com/yarpc/yarpc-go"
	"github.com/yarpc/yarpc-go/crossdock/client/behavior"
	"github.com/yarpc/yarpc-go/crossdock/client/params"
	"github.com/yarpc/yarpc-go/transport"

	tch "github.com/yarpc/yarpc-go/transport/tchannel"

	"github.com/uber/tchannel-go"
)

const (
	serverPort = 8083
	serverName = "tchannel-server"
)

// Run executes the tchserver test
func Run(s behavior.Sink, ps behavior.Params) {
	fatals := behavior.Fatals(s)

	encoding := ps.Param(params.Encoding)
	server := ps.Param(params.Server)
	serverHostPort := fmt.Sprintf("%v:%v", server, serverPort)

	ch, err := tchannel.NewChannel("yarpc-client", nil)
	if err != nil {
		fatals.Fail("Could not create channel!", err)
	}

	rpc := yarpc.New(yarpc.Config{
		Name: "yarpc-client",
		Outbounds: transport.Outbounds{
			serverName: tch.NewOutbound(ch, tch.HostPort(serverHostPort)),
		},
	})

	switch encoding {
	case "raw":
		runRaw(s, rpc)
	case "json":
		runJSON(s, rpc)
	case "thrift":
		runThrift(s, rpc)
	default:
		fatals.Fail("", "unknown encoding %q", encoding)
	}
}
