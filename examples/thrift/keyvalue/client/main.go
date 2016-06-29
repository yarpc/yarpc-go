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
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/yarpc/yarpc-go"
	"github.com/yarpc/yarpc-go/examples/thrift/keyvalue/kv/yarpc/keyvalueclient"
	"github.com/yarpc/yarpc-go/transport"
	"github.com/yarpc/yarpc-go/transport/http"
	tch "github.com/yarpc/yarpc-go/transport/tchannel"

	"github.com/uber/tchannel-go"
	"golang.org/x/net/context"
)

func main() {
	outboundName := ""
	flag.StringVar(
		&outboundName,
		"outbound", "", "name of the outbound to use (http/tchannel)",
	)

	flag.Parse()

	var outbound transport.Outbound
	switch strings.ToLower(outboundName) {
	case "http":
		outbound = http.NewOutbound("http://localhost:24034")
	case "tchannel":
		channel, err := tchannel.NewChannel("keyvalue-client", nil)
		if err != nil {
			log.Fatalln(err)
		}
		outbound = tch.NewOutbound(channel, tch.HostPort("localhost:28941"))
	default:
		log.Fatalf("invalid outbound: %q\n", outboundName)
	}

	cache := NewCacheFilter()
	rpc := yarpc.New(yarpc.Config{
		Name:      "keyvalue-client",
		Outbounds: transport.Outbounds{"keyvalue": outbound},
		Filter:    cache,
	})
	if err := rpc.Start(); err != nil {
		log.Fatalf("failed to start RPC: %v", err)
	}
	defer rpc.Stop()

	client := keyvalueclient.New(rpc.Channel("keyvalue"))

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

			ctx, _ := context.WithTimeout(rootCtx, 100*time.Millisecond)
			if value, _, err := client.GetValue(yarpc.NewReqMeta(ctx), &key); err != nil {
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
			ctx, _ := context.WithTimeout(rootCtx, 100*time.Millisecond)
			if _, err := client.SetValue(yarpc.NewReqMeta(ctx), &key, &value); err != nil {
				fmt.Printf("set %q = %q failed: %v\n", key, value, err.Error())
			}
			continue

		case "exit":
			return

		default:
			fmt.Println("invalid command", cmd)
			fmt.Println("valid commansd are: get, set, exit")
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Println("error:", err.Error())
	}
}
