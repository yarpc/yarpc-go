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
	"os"

	"github.com/yarpc/yarpc-go"
	"github.com/yarpc/yarpc-go/encoding/json"
	"github.com/yarpc/yarpc-go/transport"
	"github.com/yarpc/yarpc-go/transport/http"
	tch "github.com/yarpc/yarpc-go/transport/tchannel"

	"github.com/uber/tchannel-go"
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

func (h handler) Get(req *json.Request, body *GetRequest) (*GetResponse, *json.Response, error) {
	return &GetResponse{Value: h.items[body.Key]}, nil, nil
}

func (h handler) Set(req *json.Request, body *SetRequest) (*SetResponse, *json.Response, error) {
	h.items[body.Key] = body.Value
	return &SetResponse{}, nil, nil
}

func main() {
	channel, err := tchannel.NewChannel("keyvalue", nil)
	if err != nil {
		log.Fatalln(err)
	}

	rpc := yarpc.New(yarpc.Config{
		Name: "keyvalue",
		Inbounds: []transport.Inbound{
			tch.NewInbound(channel, tch.ListenAddr(":4040")),
		},
		Outbounds: transport.Outbounds{
			"moe": http.NewOutbound("http://localhost:8080"),
		},
	})

	handler := handler{items: make(map[string]string)}
	json.Register(rpc, json.Procedure("get", handler.Get))
	json.Register(rpc, json.Procedure("set", handler.Set))

	if err := rpc.Start(); err != nil {
		fmt.Println("error:", err.Error())
		os.Exit(1)
	}

	select {}
}
