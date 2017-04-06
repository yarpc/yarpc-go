// Copyright (c) 2017 Uber Technologies, Inc.
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
	"time"

	"go.uber.org/yarpc"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/encoding/raw"
	"go.uber.org/yarpc/internal/crossdock/client/params"
	"go.uber.org/yarpc/transport/http"
	"go.uber.org/yarpc/transport/tchannel"
	"go.uber.org/yarpc/transport/x/cherami"
	"go.uber.org/yarpc/transport/x/redis"

	"github.com/crossdock/crossdock-go"
	cherami_client "github.com/uber/cherami-client-go/client/cherami"
)

// Create creates an RPC from the given parameters or fails the whole behavior.
func Create(t crossdock.T) *yarpc.Dispatcher {
	fatals := crossdock.Fatals(t)

	server := t.Param(params.ProtobufServer)
	if server == "" {
		server = t.Param(params.Server)
	}
	fatals.NotEmpty(server, "server is required")

	var unaryOutbound transport.UnaryOutbound
	trans := t.Param(params.Transport)
	switch trans {
	case "http":
		httpTransport := http.NewTransport()
		unaryOutbound = httpTransport.NewSingleOutbound(fmt.Sprintf("http://%s:8081", server))
	case "tchannel":
		tchannelTransport, err := tchannel.NewChannelTransport(tchannel.ServiceName("client"))
		fatals.NoError(err, "Failed to build ChannelTransport")

		unaryOutbound = tchannelTransport.NewSingleOutbound(server + ":8082")
	default:
		fatals.Fail("", "unknown transport %q", trans)
	}

	return yarpc.NewDispatcher(yarpc.Config{
		Name: "client",
		Outbounds: yarpc.Outbounds{
			"yarpc-test": {
				Unary: unaryOutbound,
			},
		},
	})
}

// CreateOnewayDispatcher returns a started dispatcher and returns the address the
// server should call back to (ie this host)
func CreateOnewayDispatcher(t crossdock.T, handler raw.OnewayHandler) (*yarpc.Dispatcher, string) {
	fatals := crossdock.Fatals(t)

	server := t.Param("server_oneway")
	fatals.NotEmpty(server, "oneway server is required")

	httpTransport := http.NewTransport()
	var outbound transport.OnewayOutbound

	trans := t.Param("transport_oneway")
	switch trans {
	case "http":
		outbound = httpTransport.NewSingleOutbound(fmt.Sprintf("http://%s:8084", server))
	case "redis":
		outbound = redis.NewOnewayOutbound(
			redis.NewRedis5Client("redis:6379"),
			"yarpc/oneway",
		)
	case "cherami":
		cheramiClient, err := cherami_client.NewClient(`example`, `cherami`, 4922, &cherami_client.ClientOptions{
			Timeout: 5 * time.Second,
			ReconfigurationPollingInterval: 1 * time.Second,
		})
		fatals.NoError(err, "couldn't create cherami client")

		transport := cherami.NewTransport(cheramiClient)
		err = transport.Start()
		fatals.NoError(err, "couldn't start cherami transport")

		outbound = transport.NewOutbound(cherami.OutboundConfig{
			Destination: `/test/dest`})
	default:
		fatals.Fail("", "unknown transport %q", trans)
	}

	client := t.Param("client_oneway")
	callBackInbound := httpTransport.NewInbound(fmt.Sprintf("%s:0", client))
	dispatcher := yarpc.NewDispatcher(yarpc.Config{
		Name: "oneway-client",
		Outbounds: yarpc.Outbounds{
			"oneway-server": {Oneway: outbound},
		},
		Inbounds: yarpc.Inbounds{callBackInbound},
	})

	// register procedure for server to call us back on
	dispatcher.Register(raw.OnewayProcedure("call-back", raw.OnewayHandler(handler)))
	fatals.NoError(dispatcher.Start(), "could not start oneway Dispatcher")

	return dispatcher, callBackInbound.Addr().String()
}
