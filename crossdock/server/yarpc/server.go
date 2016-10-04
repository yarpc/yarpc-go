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

	"go.uber.org/yarpc"
	"go.uber.org/yarpc/crossdock/thrift/echo/yarpc/echoserver"
	"go.uber.org/yarpc/crossdock/thrift/gauntlet/yarpc/secondserviceserver"
	"go.uber.org/yarpc/crossdock/thrift/gauntlet/yarpc/thrifttestserver"
	"go.uber.org/yarpc/encoding/json"
	"go.uber.org/yarpc/encoding/raw"
	"go.uber.org/yarpc/encoding/thrift"
	"go.uber.org/yarpc/transport"
	"go.uber.org/yarpc/transport/http"
	tch "go.uber.org/yarpc/transport/tchannel"

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

func register(reg transport.Registry) {
	raw.Register(reg, raw.Procedure("echo/raw", EchoRaw))
	json.Register(reg, json.Procedure("echo", EchoJSON))

	// NOTE(abg): Enveloping is disabled in old cross-language tests until the
	// other YARPC implementations catch up.
	thrift.Register(reg, echoserver.New(EchoThrift{}), thrift.DisableEnveloping)
	thrift.Register(reg, thrifttestserver.New(thriftTest{}), thrift.DisableEnveloping)
	thrift.Register(reg, secondserviceserver.New(secondService{}), thrift.DisableEnveloping)

	json.Register(reg, json.Procedure("unexpected-error", UnexpectedError))
	json.Register(reg, json.Procedure("bad-response", BadResponse))
	json.Register(reg, json.Procedure("phone", Phone))
	json.Register(reg, json.Procedure("sleep", Sleep))

	raw.Register(reg, raw.Procedure("sleep/raw", SleepRaw))
	raw.Register(reg, raw.Procedure("waitfortimeout/raw", WaitForTimeoutRaw))
}
