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

	"github.com/yarpc/yarpc-go"
	"github.com/yarpc/yarpc-go/encoding/json"
	"github.com/yarpc/yarpc-go/transport"
	"github.com/yarpc/yarpc-go/transport/http"

	"golang.org/x/net/context"
)

type GetRequest struct {
	Key string `json:"key"`
}

type GetResponse struct {
	Value string `json:"value"`
}

type SetRequest struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type SetResponse struct {
}

type handler struct {
	items map[string]string
}

func (h handler) Get(ctx context.Context, meta yarpc.Meta, req *GetRequest) (*GetResponse, yarpc.Meta, error) {
	return &GetResponse{Value: h.items[req.Key]}, nil, nil
}

func (h handler) Set(ctx context.Context, meta yarpc.Meta, req *SetRequest) (*SetResponse, yarpc.Meta, error) {
	h.items[req.Key] = req.Value
	return &SetResponse{}, nil, nil
}

func main() {
	yarpc := yarpc.New(yarpc.Config{
		Name: "keyvalue",
		Inbounds: []transport.Inbound{
			http.Inbound(":8080"),
			// TODO tchannel.Inbound{},
		},
		Outbounds: map[string]transport.Outbound{
			"moe": http.Outbound("http://localhost:8080"),
		},
	})

	handler := handler{items: make(map[string]string)}
	json.Register(yarpc, json.Procedure("get", handler.Get))
	json.Register(yarpc, json.Procedure("set", handler.Set))

	if err := yarpc.Serve(); err != nil {
		fmt.Println("error:", err.Error())
	}
}
