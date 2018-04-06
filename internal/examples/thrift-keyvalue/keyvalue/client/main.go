// Copyright (c) 2018 Uber Technologies, Inc.
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
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"go.uber.org/yarpc"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/internal/examples/thrift-keyvalue/keyvalue/kv/keyvalueclient"
	"go.uber.org/yarpc/transport/grpc"
	"go.uber.org/yarpc/transport/http"
	"go.uber.org/yarpc/transport/tchannel"
)

func main() {
	if err := do(); err != nil {
		log.Fatal(err)
	}
}

func do() error {
	flagSet := flag.NewFlagSet("client", flag.ExitOnError)
	outboundName := ""
	flagSet.StringVar(
		&outboundName,
		"outbound", "", "name of the outbound to use (http/tchannel/grpc)",
	)

	if err := flagSet.Parse(os.Args[1:]); err != nil {
		return err
	}

	httpTransport := http.NewTransport()
	tchannelTransport, err := tchannel.NewChannelTransport(tchannel.ServiceName("keyvalue-client"))
	if err != nil {
		return err
	}

	var outbound transport.UnaryOutbound
	switch strings.ToLower(outboundName) {
	case "http":
		outbound = httpTransport.NewSingleOutbound("http://127.0.0.1:24042")
	case "tchannel":
		outbound = tchannelTransport.NewSingleOutbound("127.0.0.1:28945")
	case "grpc":
		outbound = grpc.NewTransport().NewSingleOutbound("127.0.0.1:24046")
	default:
		return fmt.Errorf("invalid outbound: %q", outboundName)
	}

	cache := NewCacheOutboundMiddleware()
	dispatcher := yarpc.NewDispatcher(yarpc.Config{
		Name: "keyvalue-client",
		Outbounds: yarpc.Outbounds{
			"keyvalue": {
				Unary: outbound,
			},
		},
		OutboundMiddleware: yarpc.OutboundMiddleware{
			Unary: cache,
		},
	})
	if err := dispatcher.Start(); err != nil {
		return fmt.Errorf("failed to start Dispatcher: %v", err)
	}
	defer dispatcher.Stop()

	client := keyvalueclient.New(dispatcher.ClientConfig("keyvalue"))

	scanner := bufio.NewScanner(os.Stdin)
	rootCtx := context.Background()
	for scanner.Scan() {
		line := scanner.Text()
		args := strings.Split(line, " ")
		if len(args) < 1 || len(args[0]) < 3 {
			continue
		}

		cmd := args[0]
		args = args[1:]

		switch cmd {

		case "get":
			if len(args) != 1 {
				fmt.Println("usage: get key")
				continue
			}
			key := args[0]

			ctx, cancel := context.WithTimeout(rootCtx, 100*time.Millisecond)
			defer cancel()

			if value, err := client.GetValue(ctx, &key); err != nil {
				fmt.Printf("get %q failed: %s\n", key, err)
			} else {
				fmt.Println(key, "=", value)
			}
			continue

		case "set":
			if len(args) != 2 {
				fmt.Println("usage: set key value")
				continue
			}
			key, value := args[0], args[1]

			cache.Invalidate()
			ctx, cancel := context.WithTimeout(rootCtx, 100*time.Millisecond)
			defer cancel()

			if err := client.SetValue(ctx, &key, &value); err != nil {
				fmt.Printf("set %q = %q failed: %v\n", key, value, err.Error())
			}
			continue

		case "exit":
			return nil

		default:
			return fmt.Errorf("invalid command, valid commansd are: get, set, exit: %s", cmd)
		}
	}
	return scanner.Err()
}
