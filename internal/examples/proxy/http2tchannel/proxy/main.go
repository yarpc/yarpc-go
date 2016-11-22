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

package main

import (
	"context"
	"flag"
	"log"

	"github.com/uber/tchannel-go"
	transport "go.uber.org/yarpc/transport"
	yhttp "go.uber.org/yarpc/transport/http"
	ytchannel "go.uber.org/yarpc/transport/tchannel"
)

func main() {
	channel, err := tchannel.NewChannel("proxy", nil)
	if err != nil {
		log.Fatalf("failed to create channel %v", err)
	}

	flag.Parse()
	inbound := yhttp.NewInbound(flag.Arg(0))
	outbound := ytchannel.NewOutbound(channel, ytchannel.HostPort(flag.Arg(1)))

	detail := transport.ServiceDetail{
		Name: "proxy",
		Registry: &registry{
			outbound: transport.NewUnaryRelay(outbound),
		},
	}

	err = inbound.Start(detail, transport.NoDeps)
	if err != nil {
		log.Fatalf("failed to start inbound %v", err)
	}
	defer inbound.Stop()

	err = outbound.Start(transport.NoDeps)
	if err != nil {
		log.Fatalf("failed to start outbound %v", err)
	}
	defer outbound.Stop()

	log.Printf("server started\n")
	select {}
}

type registry struct {
	outbound transport.UnaryHandler
}

func (r *registry) Choose(context context.Context, req *transport.Request) (transport.HandlerSpec, error) {
	return transport.NewUnaryHandlerSpec(r.outbound), nil
}

func (r *registry) ServiceProcedures() []transport.ServiceProcedure {
	return make([]transport.ServiceProcedure, 0)
}
