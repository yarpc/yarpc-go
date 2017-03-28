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

	"go.uber.org/yarpc/internal/examples/protobuf/example"
	"go.uber.org/yarpc/internal/examples/protobuf/examplepb"
	"go.uber.org/yarpc/internal/testutils"
)

func main() {
	if err := do(); err != nil {
		log.Fatal(err)
	}
}

func do() error {
	keyValueYarpcServer := example.NewKeyValueYarpcServer()
	sinkYarpcServer := example.NewSinkYarpcServer(true)
	// TChannel will be used for unary
	// HTTP wil be used for oneway
	return example.WithClients(
		testutils.TransportTypeTChannel,
		keyValueYarpcServer,
		sinkYarpcServer,
		func(keyValueYarpcClient examplepb.KeyValueYarpcClient, sinkYarpcClient examplepb.SinkYarpcClient) error {
			return doClient(keyValueYarpcClient, sinkYarpcClient, keyValueYarpcServer, sinkYarpcServer)
		},
	)
}

func doClient(
	keyValueYarpcClient examplepb.KeyValueYarpcClient,
	sinkYarpcClient examplepb.SinkYarpcClient,
	keyValueYarpcServer *example.KeyValueYarpcServer,
	sinkYarpcServer *example.SinkYarpcServer,
) error {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := scanner.Text()
		fmt.Println(line)
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
			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
			defer cancel()
			if response, err := keyValueYarpcClient.GetValue(ctx, &examplepb.GetValueRequest{key}); err != nil {
				fmt.Printf("get %s failed: %s\n", key, err.Error())
			} else {
				fmt.Println(key, "=", response.Value)
			}
			continue
		case "set":
			if len(args) != 1 && len(args) != 2 {
				fmt.Println("usage: set key [value]")
				continue
			}
			key := args[0]
			value := ""
			if len(args) == 2 {
				value = args[1]
			}
			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
			defer cancel()
			if _, err := keyValueYarpcClient.SetValue(ctx, &examplepb.SetValueRequest{key, value}); err != nil {
				fmt.Printf("set %s = %s failed: %v\n", key, value, err.Error())
			}
			continue
		case "fire":
			if len(args) != 1 {
				fmt.Println("usage: fire value")
				continue
			}
			value := args[0]
			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
			defer cancel()
			if _, err := sinkYarpcClient.Fire(ctx, &examplepb.FireRequest{value}); err != nil {
				fmt.Printf("fire %s failed: %s\n", value, err.Error())
			}
			if err := sinkServer.WaitFireDone(); err != nil {
				fmt.Println(err)
			}
			continue
		case "fired-values":
			if len(args) != 0 {
				fmt.Println("usage: fired-values")
				continue
			}
			fmt.Println(strings.Join(sinkYarpcServer.Values(), " "))
			continue
		case "exit":
			return nil
		default:
			fmt.Printf("invalid command: %s\nvalid commands are: get, set, fire, fired-values, exit\n", cmd)
		}
	}
	return scanner.Err()
}
