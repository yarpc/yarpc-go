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
	"fmt"
	"log"
	"sync"

	"github.com/yarpc/yarpc-go"
	"github.com/yarpc/yarpc-go/encoding/thrift"
	"github.com/yarpc/yarpc-go/examples/thrift/keyvalue/kv"
	"github.com/yarpc/yarpc-go/examples/thrift/keyvalue/kv/yarpc/keyvalueserver"
	"github.com/yarpc/yarpc-go/transport"
	"github.com/yarpc/yarpc-go/transport/http"
	tch "github.com/yarpc/yarpc-go/transport/tchannel"

	"github.com/uber/tchannel-go"
)

type handler struct {
	sync.RWMutex

	items map[string]string
}

func (h *handler) GetValue(req *thrift.Request, key *string) (string, *thrift.Response, error) {
	h.RLock()
	defer h.RUnlock()

	if value, ok := h.items[*key]; ok {
		return value, nil, nil
	}

	return "", nil, &kv.ResourceDoesNotExist{Key: *key}
}

func (h *handler) SetValue(req *thrift.Request, key *string, value *string) (*thrift.Response, error) {
	h.Lock()
	h.items[*key] = *value
	h.Unlock()
	return nil, nil
}

func main() {
	channel, err := tchannel.NewChannel("keyvalue", nil)
	if err != nil {
		log.Fatalln(err)
	}

	rpc := yarpc.New(yarpc.Config{
		Name: "keyvalue",
		Inbounds: []transport.Inbound{
			tch.NewInbound(channel, tch.ListenAddr(":28941")),
			http.NewInbound(":24034"),
		},
	})

	handler := handler{items: make(map[string]string)}
	thrift.Register(rpc, keyvalueserver.New(&handler))

	if err := rpc.Start(); err != nil {
		fmt.Println("error:", err.Error())
	}

	select {} // block forever
}
