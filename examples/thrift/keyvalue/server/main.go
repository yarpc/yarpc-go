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
	_ "expvar"
	"fmt"
	"sync"

	"go.uber.org/yarpc"
	"go.uber.org/yarpc/examples/thrift/keyvalue/kv"
	"go.uber.org/yarpc/examples/thrift/keyvalue/kv/yarpc/keyvalueclient"
	"go.uber.org/yarpc/examples/thrift/keyvalue/kv/yarpc/keyvalueserver"
	"go.uber.org/yarpc/transport/http"
	"go.uber.org/yarpc/transport/tchannel"
)

type handler struct {
	sync.RWMutex

	items  map[string]string
	client keyvalueclient.Interface
}

func (h *handler) GetValue(ctx context.Context, reqMeta yarpc.ReqMeta,
	key *string, hopcnt *int8) (string, yarpc.ResMeta, error) {
	if hopcnt != nil && *hopcnt > 0 {
		*hopcnt--
		value, _, err := h.client.GetValue(ctx, nil, key, hopcnt)
		return value, nil, err
	}
	h.RLock()
	defer h.RUnlock()

	if value, ok := h.items[*key]; ok {
		return value, nil, nil
	}

	return "", nil, &kv.ResourceDoesNotExist{Key: *key}
}

func (h *handler) SetValue(ctx context.Context, reqMeta yarpc.ReqMeta,
	key *string, value *string, hopcnt *int8) (yarpc.ResMeta, error) {
	if hopcnt != nil && *hopcnt > 0 {
		*hopcnt--
		_, err := h.client.SetValue(ctx, nil, key, value, hopcnt)
		return nil, err
	}

	h.Lock()
	h.items[*key] = *value
	h.Unlock()
	return nil, nil
}

func main() {
	tchannelTransport := tchannel.NewChannelTransport(
		tchannel.ServiceName("keyvalue"),
		tchannel.ListenAddr(":28941"),
	)
	httpTransport := http.NewTransport()

	outboundhttp := httpTransport.NewSingleOutbound("http://127.0.0.1:24034")
	outboundtch := tchannelTransport.NewSingleOutbound("localhost:28941")

	dispatcher := yarpc.NewDispatcher(yarpc.Config{
		Name: "keyvalue",
		Inbounds: yarpc.Inbounds{
			tchannelTransport.NewInbound(),
			httpTransport.NewInbound(":24034"),
		},
		Outbounds: yarpc.Outbounds{
			"keyvalue_http": {
				Unary: outboundhttp,
			},
			"keyvalue": {
				Unary: outboundtch,
			},
		},
	})

	yarpc.AddDebugPagesFor(dispatcher)

	client := keyvalueclient.New(dispatcher.ClientConfig("keyvalue"))
	handler := handler{items: make(map[string]string), client: client}
	dispatcher.Register(keyvalueserver.New(&handler))

	if err := dispatcher.Start(); err != nil {
		fmt.Println("error:", err.Error())
	}

	select {} // block forever
}
