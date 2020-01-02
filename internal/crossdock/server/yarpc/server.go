// Copyright (c) 2020 Uber Technologies, Inc.
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
	"net"

	"go.uber.org/yarpc"
	"go.uber.org/yarpc/encoding/json"
	"go.uber.org/yarpc/encoding/raw"
	"go.uber.org/yarpc/internal/crossdock/crossdockpb"
	"go.uber.org/yarpc/internal/crossdock/thrift/echo/echoserver"
	"go.uber.org/yarpc/internal/crossdock/thrift/gauntlet/secondserviceserver"
	"go.uber.org/yarpc/internal/crossdock/thrift/gauntlet/thrifttestserver"
	"go.uber.org/yarpc/transport/grpc"
	"go.uber.org/yarpc/transport/http"
	"go.uber.org/yarpc/transport/tchannel"
)

var dispatcher *yarpc.Dispatcher

// Start starts the test server that clients will make requests to
func Start() {
	tchannelTransport, err := tchannel.NewChannelTransport(
		tchannel.ListenAddr(":8082"),
		tchannel.ServiceName("yarpc-test"),
	)
	if err != nil {
		log.Panicf("failed to build ChannelTransport: %v", err)
	}
	grpcListener, err := net.Listen("tcp", ":8089")
	if err != nil {
		log.Panic(err)
	}

	httpTransport := http.NewTransport()
	dispatcher = yarpc.NewDispatcher(yarpc.Config{
		Name: "yarpc-test",
		Inbounds: yarpc.Inbounds{
			tchannelTransport.NewInbound(),
			httpTransport.NewInbound(":8081"),
			grpc.NewTransport().NewInbound(grpcListener),
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

func register(reg *yarpc.Dispatcher) {
	reg.Register(raw.Procedure("echo/raw", EchoRaw))
	reg.Register(json.Procedure("echo", EchoJSON))

	reg.Register(echoserver.New(EchoThrift{}))
	reg.Register(thrifttestserver.New(thriftTest{}))
	reg.Register(secondserviceserver.New(secondService{}))

	reg.Register(json.Procedure("unexpected-error", UnexpectedError))
	reg.Register(json.Procedure("bad-response", BadResponse))
	reg.Register(json.Procedure("phone", Phone))
	reg.Register(json.Procedure("sleep", Sleep))

	reg.Register(raw.Procedure("sleep/raw", SleepRaw))
	reg.Register(raw.Procedure("waitfortimeout/raw", WaitForTimeoutRaw))

	reg.Register(crossdockpb.BuildEchoYARPCProcedures(EchoProtobuf{}))
}
