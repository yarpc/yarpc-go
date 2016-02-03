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
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/yarpc/yarpc-go"
	"github.com/yarpc/yarpc-go/scheme/json"
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

func get(ctx context.Context, c json.Client, k string) (string, error) {
	var response GetResponse
	_, err := c.Call(
		ctx,
		&json.Request{Procedure: "get", Body: &GetRequest{Key: k}},
		&response,
	)
	return response.Value, err
}

func set(ctx context.Context, c json.Client, k string, v string) error {
	var response SetResponse
	_, err := c.Call(
		ctx,
		&json.Request{Procedure: "set", Body: &SetRequest{Key: k, Value: v}},
		&response,
	)
	return err
}

func main() {
	yarpc := yarpc.New(yarpc.Config{
		Name: "keyvalue-client",
		Outbounds: map[string]transport.Outbounds{
			"keyvalue": {http.Outbound("http://localhost:8080")},
		},
	})

	client := json.New(yarpc.Channel("keyvalue"))

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
			if value, err := get(ctx, client, key); err != nil {
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

			ctx, _ := context.WithTimeout(rootCtx, 100*time.Millisecond)
			if err := set(ctx, client, key, value); err != nil {
				fmt.Println("set %q = %q failed: %v", key, value, err.Error())
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
