// Copyright (c) 2026 Uber Technologies, Inc.
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
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"sync"

	"go.uber.org/yarpc"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/encoding/json"
	"go.uber.org/yarpc/transport/grpc"
	"go.uber.org/yarpc/transport/http"
	"go.uber.org/yarpc/transport/tchannel"
	"go.uber.org/yarpc/yarpcerrors"
)

var (
	flagSet     = flag.NewFlagSet("server", flag.ExitOnError)
	flagInbound = flagSet.String("inbound", "", "name of the inbound to use (http/tchannel/grpc)")
)

type getRequest struct {
	Key string `json:"key"`
}

type getResponse struct {
	Value string `json:"value"`
}

type setRequest struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type setResponse struct {
}

type handler struct {
	sync.RWMutex

	items map[string]string
}

func (h *handler) Get(ctx context.Context, body *getRequest) (*getResponse, error) {
	h.RLock()
	value, ok := h.items[body.Key]
	h.RUnlock()
	if !ok {
		return nil, yarpcerrors.Newf(yarpcerrors.CodeNotFound, body.Key)
	}
	return &getResponse{Value: value}, nil
}

func (h *handler) Set(ctx context.Context, body *setRequest) (*setResponse, error) {
	h.Lock()
	h.items[body.Key] = body.Value
	h.Unlock()
	return &setResponse{}, nil
}

type requestLogInboundMiddleware struct{}

func (requestLogInboundMiddleware) Handle(ctx context.Context, req *transport.Request, resw transport.ResponseWriter, handler transport.UnaryHandler) error {
	fmt.Printf("received a request to %q from client %q (encoding %q)\n",
		req.Procedure, req.Caller, req.Encoding)
	return handler.Handle(ctx, req, resw)
}

func main() {
	if err := do(); err != nil {
		log.Fatal(err)
	}
}

func do() error {
	if err := flagSet.Parse(os.Args[1:]); err != nil {
		return err
	}
	var inbound transport.Inbound
	switch strings.ToLower(*flagInbound) {
	case "http":
		inbound = http.NewTransport().NewInbound("127.0.0.1:24034")
	case "tchannel":
		tchannelTransport, err := tchannel.NewChannelTransport(
			tchannel.ServiceName("keyvalue"),
			tchannel.ListenAddr("127.0.0.1:28941"),
		)
		if err != nil {
			return err
		}
		inbound = tchannelTransport.NewInbound()
	case "grpc":
		listener, err := net.Listen("tcp", "127.0.0.1:24038")
		if err != nil {
			return err
		}
		inbound = grpc.NewTransport().NewInbound(listener)
	default:
		return fmt.Errorf("invalid inbound: %q", *flagInbound)
	}

	dispatcher := yarpc.NewDispatcher(yarpc.Config{
		Name:     "keyvalue",
		Inbounds: yarpc.Inbounds{inbound},
		InboundMiddleware: yarpc.InboundMiddleware{
			Unary: requestLogInboundMiddleware{},
		},
	})

	handler := handler{items: make(map[string]string)}

	dispatcher.Register(json.Procedure("get", handler.Get))
	dispatcher.Register(json.Procedure("set", handler.Set))

	if err := dispatcher.Start(); err != nil {
		return err
	}

	select {}
}
