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
	"time"

	"go.uber.org/yarpc"
	"go.uber.org/yarpc/internal/examples/streaming"
	"go.uber.org/yarpc/transport/grpc"
)

func main() {
	if err := do(); err != nil {
		log.Fatal(err)
	}
}

func do() error {
	dispatcher := yarpc.NewDispatcher(yarpc.Config{
		Name: "keyvalue-client",
		Outbounds: yarpc.Outbounds{
			"keyvalue": {
				Stream: grpc.NewTransport().NewSingleOutbound("127.0.0.1:24039"),
			},
		},
	})

	client := streaming.NewHelloYARPCClient(dispatcher.MustOutboundConfig("keyvalue"))

	if err := dispatcher.Start(); err != nil {
		return fmt.Errorf("failed to start Dispatcher: %v", err)
	}
	defer dispatcher.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	stream, err := client.HelloThere(ctx, yarpc.WithHeader("test", "testtest"))
	if err != nil {
		return fmt.Errorf("failed to create stream: %s", err.Error())
	}

	scanner := bufio.NewScanner(os.Stdin)
	fmt.Printf(">>> ")
	for scanner.Scan() {
		line := scanner.Text()
		if line == "stop" {
			return stream.CloseSend()
		}

		fmt.Printf("sending message: %q\n", line)
		if err := stream.Send(&streaming.HelloRequest{line}); err != nil {
			return err
		}

		fmt.Println("waiting for response...")
		msg, err := stream.Recv()
		if err != nil {
			return err
		}

		fmt.Printf("got response: %q\n", msg.Id)
		fmt.Printf(">>> ")
	}
	return scanner.Err()
}
