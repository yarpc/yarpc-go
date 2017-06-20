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
	"flag"
	"fmt"
	"log"
	"net"
	gohttp "net/http"
	"strings"
	"sync"

	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/internal/examples/thrift-keyvalue/keyvalue/kv"
	"go.uber.org/yarpc/internal/examples/thrift-keyvalue/keyvalue/kv/keyvalueserver"
	"go.uber.org/yarpc/x/yarpcmeta"

	"go.uber.org/yarpc"
	"go.uber.org/yarpc/encoding/thrift"
	"go.uber.org/yarpc/transport/http"
	"go.uber.org/yarpc/transport/tchannel"
	"go.uber.org/yarpc/transport/x/grpc"
)

var (
	flagInbound  = flag.String("inbound", "", "name of the inbound to use (http/tchannel/grpc)")
	flagEnvelope = flag.Bool("enveloped", false, "enable thrift envelopes")
)

type handler struct {
	sync.RWMutex

	items map[string]string
}

func (h *handler) GetValue(ctx context.Context, key *string) (string, error) {
	h.RLock()
	defer h.RUnlock()

	if value, ok := h.items[*key]; ok {
		return value, nil
	}

	return "", &kv.ResourceDoesNotExist{Key: *key}
}

func (h *handler) SetValue(ctx context.Context, key *string, value *string) error {
	h.Lock()
	h.items[*key] = *value
	h.Unlock()
	return nil
}

func main() {
	if err := do(); err != nil {
		log.Fatal(err)
	}
}

func do() error {
	flag.Parse()
	var inbound transport.Inbound
	switch strings.ToLower(*flagInbound) {
	case "http":
		inbound = http.NewTransport().NewInbound("127.0.0.1:24042")
		go func() {
			if err := gohttp.ListenAndServe(":3242", nil); err != nil {
				log.Fatal(err)
			}
		}()
	case "tchannel":
		tchannelTransport, err := tchannel.NewChannelTransport(
			tchannel.ServiceName("keyvalue"),
			tchannel.ListenAddr("127.0.0.1:28945"),
		)
		if err != nil {
			return err
		}
		inbound = tchannelTransport.NewInbound()
		go func() {
			if err := gohttp.ListenAndServe(":3243", nil); err != nil {
				log.Fatal(err)
			}
		}()
	case "grpc":
		listener, err := net.Listen("tcp", "127.0.0.1:24046")
		if err != nil {
			return err
		}
		inbound = grpc.NewTransport().NewInbound(listener)
		go func() {
			if err := gohttp.ListenAndServe(":3244", nil); err != nil {
				log.Fatal(err)
			}
		}()
	default:
		return fmt.Errorf("invalid inbound: %q", *flagInbound)
	}

	dispatcher := yarpc.NewDispatcher(yarpc.Config{
		Name:     "keyvalue",
		Inbounds: yarpc.Inbounds{inbound},
	})

	handler := handler{items: make(map[string]string)}

	var opts []thrift.RegisterOption
	if *flagEnvelope {
		opts = append(opts, thrift.Enveloped)
	}
	dispatcher.Register(keyvalueserver.New(&handler, opts...))

	yarpcmeta.Register(dispatcher)

	if err := dispatcher.Start(); err != nil {
		return err
	}

	select {} // block forever
}
