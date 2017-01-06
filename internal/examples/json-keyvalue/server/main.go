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

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"

	"go.uber.org/yarpc"
	"go.uber.org/yarpc/encoding/json"
	"go.uber.org/yarpc/transport/http"
	"go.uber.org/yarpc/transport/tchannel"
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
	result := &getResponse{Value: h.items[body.Key]}
	h.RUnlock()
	return result, nil
}

func (h *handler) Set(ctx context.Context, body *setRequest) (*setResponse, error) {
	h.Lock()
	h.items[body.Key] = body.Value
	h.Unlock()
	return &setResponse{}, nil
}

func main() {
	tchannelTransport, err := tchannel.NewChannelTransport(
		tchannel.ServiceName("keyvalue"),
		tchannel.ListenAddr(":28941"),
	)
	if err != nil {
		log.Fatal(err)
	}

	httpTransport := http.NewTransport()
	dispatcher := yarpc.NewDispatcher(yarpc.Config{
		Name: "keyvalue",
		Inbounds: yarpc.Inbounds{
			tchannelTransport.NewInbound(),
			httpTransport.NewInbound(":24034"),
		},
		InboundMiddleware: yarpc.InboundMiddleware{
			Unary: requestLogInboundMiddleware{},
		},
	})

	handler := handler{items: make(map[string]string)}

	dispatcher.Register(json.Procedure("get", handler.Get))
	dispatcher.Register(json.Procedure("set", handler.Set))

	if err := dispatcher.Start(); err != nil {
		fmt.Println("error:", err.Error())
		os.Exit(1)
	}

	select {}
}
