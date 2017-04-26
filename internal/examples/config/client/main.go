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
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"go.uber.org/yarpc/encoding/json"
	"go.uber.org/yarpc/transport/http"
	"go.uber.org/yarpc/transport/tchannel"
	"go.uber.org/yarpc/transport/x/redis"
	"go.uber.org/yarpc/x/config"
)

var cfg = config.New()

func init() {
	cfg.MustRegisterTransport(http.TransportSpec())
	cfg.MustRegisterTransport(tchannel.TransportSpec())
	cfg.MustRegisterTransport(redis.TransportSpec())
}

func main() {
	config, err := os.Open("config.yaml")
	if err != nil {
		log.Fatal(err)
	}
	defer config.Close()

	d, err := cfg.NewDispatcherFromYAML("myservice-client", config)
	if err != nil {
		log.Fatal(err)
	}

	client := json.New(d.ClientConfig("myservice"))

	if err := d.Start(); err != nil {
		log.Fatal(err)
	}
	defer d.Stop()

	scanner := bufio.NewScanner(os.Stdin)
	rootCtx := context.Background()
loop:
	for scanner.Scan() {
		line := scanner.Text()
		cmd := strings.Split(line, " ")[0]
		switch cmd {
		case "echo":
			fmt.Println("sending echo")

			ctx, cancel := context.WithTimeout(rootCtx, 100*time.Millisecond)
			defer cancel()

			var res map[string]string
			err := client.Call(ctx, "echo", map[string]string{"hello": "world"}, &res)
			if err != nil {
				fmt.Println("failed:", err)
				break
			}
			fmt.Println("success:", res)
		case "enqueue":
			fmt.Println("enqueuing")

			ctx, cancel := context.WithTimeout(rootCtx, 100*time.Millisecond)
			defer cancel()

			ack, err := client.CallOneway(ctx, "enqueue", map[string]string{"hello": "world"})
			if err != nil {
				fmt.Println("failed:", err)
				break
			}
			fmt.Println("ack:", ack)
		case "exit":
			break loop
		default:
			fmt.Println("invalid command", cmd)
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Println("error:", err.Error())
	}
}
