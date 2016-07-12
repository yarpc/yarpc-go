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

package yarpc

import (
	"fmt"
	"log"

	"github.com/yarpc/yarpc-go"
	"github.com/yarpc/yarpc-go/crossdock/thrift/echo/yarpc/echoserver"
	"github.com/yarpc/yarpc-go/crossdock/thrift/gauntlet/yarpc/secondserviceserver"
	"github.com/yarpc/yarpc-go/crossdock/thrift/gauntlet/yarpc/thrifttestserver"
	"github.com/yarpc/yarpc-go/encoding/json"
	"github.com/yarpc/yarpc-go/encoding/raw"
	"github.com/yarpc/yarpc-go/encoding/thrift"
	"github.com/yarpc/yarpc-go/transport"
	"github.com/yarpc/yarpc-go/transport/http"
	tch "github.com/yarpc/yarpc-go/transport/tchannel"

	"github.com/uber/tchannel-go"
)

var dispatcher yarpc.Dispatcher

// Start starts the test server that clients will make requests to
func Start() {
	ch, err := tchannel.NewChannel("yarpc-test", nil)
	if err != nil {
		log.Fatalln("couldn't create tchannel: %v", err)
	}

	dispatcher = yarpc.NewDispatcher(yarpc.Config{
		Name: "yarpc-test",
		Inbounds: []transport.Inbound{
			http.NewInbound(":8081"),
			tch.NewInbound(ch, tch.ListenAddr(":8082")),
		},
	})

	register(dispatcher)

	if err := dispatcher.Start(); err != nil {
		fmt.Println("error:", err.Error())
	}
}

// Stop stops running the RPC test subject
func Stop() {
	if dispatcher == nil {
		return
	}
	if err := dispatcher.Stop(); err != nil {
		fmt.Println("failed to stop:", err.Error())
	}
}

func register(rpc transport.Registry) {
	raw.Register(rpc, raw.Procedure("echo/raw", EchoRaw))
	json.Register(rpc, json.Procedure("echo", EchoJSON))
	thrift.Register(rpc, echoserver.New(EchoThrift{}))
	thrift.Register(rpc, thrifttestserver.New(thriftTest{}))
	thrift.Register(rpc, secondserviceserver.New(secondService{}))

	json.Register(rpc, json.Procedure("unexpected-error", UnexpectedError))
	json.Register(rpc, json.Procedure("bad-response", BadResponse))
	json.Register(rpc, json.Procedure("phone", Phone))

	raw.Register(rpc, raw.Procedure("sleep/raw", SleepRaw))
}
