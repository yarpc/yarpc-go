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

package oneway

import (
	"context"
	"fmt"

	"go.uber.org/yarpc"
	"go.uber.org/yarpc/crossdock/client/params"
	"go.uber.org/yarpc/crossdock/client/random"
	"go.uber.org/yarpc/encoding/raw"
	"go.uber.org/yarpc/transport"
	"go.uber.org/yarpc/transport/http"

	"github.com/crossdock/crossdock-go"
)

var serverCalledBack chan []byte

// Run starts an http run using encoding types
func Run(t crossdock.T) {
	encoding := t.Param(params.Encoding)
	t.Tag("encoding", encoding)
	t.Tag("server", t.Param(params.Server))

	fatals := crossdock.Fatals(t)
	dispatcher := newDispatcher(t)

	fatals.NoError(dispatcher.Start(), "could not start Dispatcher")
	defer dispatcher.Stop()

	serverCalledBack = make(chan []byte)

	switch encoding {
	case "raw":
		Raw(t, dispatcher)
	case "json":
		JSON(t, dispatcher)
	case "thrift":
		Thrift(t, dispatcher)
	default:
		fatals.Fail("unknown encoding", "%v", encoding)
	}
}

func newDispatcher(t crossdock.T) yarpc.Dispatcher {
	server := t.Param(params.Server)
	crossdock.Fatals(t).NotEmpty(server, "server is required")

	dispatcher := yarpc.NewDispatcher(yarpc.Config{
		Name: "client",
		Outbounds: yarpc.Outbounds{
			"oneway-test": {
				Oneway: http.NewOutbound(fmt.Sprintf("http://%s:8084", server)),
			},
		},
		//for call back
		Inbounds: []transport.Inbound{http.NewInbound(fmt.Sprintf("%s:8089", server))},
	})

	// register procedure for remote server to call us back on
	dispatcher.Register(raw.OnewayProcedure("call-back", callBack))
	return dispatcher
}

func getRandomID() string {
	return random.String(10)
}

func callBack(ctx context.Context, reqMeta yarpc.ReqMeta, body []byte) error {
	serverCalledBack <- body
	return nil
}
