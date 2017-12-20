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
	"errors"
	"fmt"
	"io"
	"log"
	"net"

	"go.uber.org/yarpc"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/internal/examples/streaming"
	"go.uber.org/yarpc/transport/grpc"
	"go.uber.org/yarpc/transport/x/websocket"
)

type handler struct {
}

func (h *handler) HelloUnary(context.Context, *streaming.HelloRequest) (*streaming.HelloResponse, error) {
	return nil, errors.New("NOT READY")
}

func (h *handler) HelloOutStream(streaming.HelloServiceHelloOutStreamYARPCServer) (*streaming.HelloResponse, error) {
	return nil, nil
}

func (h *handler) HelloInStream(*streaming.HelloRequest, streaming.HelloServiceHelloInStreamYARPCServer) error {
	return nil
}

func (h *handler) HelloThere(stream streaming.HelloServiceHelloThereYARPCServer) error {
	fmt.Println("Stream created!")
	call := yarpc.CallFromContext(stream.Context())
	fmt.Println("Call: ", call.Service(), call.Procedure())
	for _, k := range call.HeaderNames() {
		fmt.Println("Got header: ", k, ":", call.Header(k))
	}
	for {
		msg, err := stream.Recv()
		if err != nil {
			fmt.Printf("Error reading from stream: %q\n", err.Error())
			if err == io.EOF {
				if err := stream.Send(&streaming.HelloResponse{"this is my final message"}); err != nil {
					fmt.Printf("Error sending message to stream: %q\n", err.Error())
					call.WriteResponseHeader("endofrequest", "error on response")
					return err
				}
				call.WriteResponseHeader("endofrequest", "sent response")
				return nil
			}
			return err
		}

		if msg.Id == "exit" {
			fmt.Println("Received 'exit' message, closing connection")
			call.WriteResponseHeader("endofrequest", "exit message received")
			return nil
		}
		fmt.Printf("Received a message: %q\n", msg.Id)

		resp := fmt.Sprintf("Got your message: %q, thanks!", msg.Id)
		if err := stream.Send(&streaming.HelloResponse{resp}); err != nil {
			fmt.Printf("Error sending message to stream: %q\n", err.Error())
			return err
		}
		fmt.Printf("Sent response message: %q\n", resp)
	}
}

func main() {
	if err := do(); err != nil {
		log.Fatal(err)
	}
}

func do() error {
	var inbound transport.Inbound
	listener, err := net.Listen("tcp", "127.0.0.1:24039")
	if err != nil {
		return err
	}
	inbound = grpc.NewTransport().NewInbound(listener)

	webListener, err := net.Listen("tcp", "127.0.0.1:24040")
	if err != nil {
		return err
	}
	webInbound := websocket.NewTransport().NewInbound(webListener)

	dispatcher := yarpc.NewDispatcher(yarpc.Config{
		Name:     "keyvalue",
		Inbounds: yarpc.Inbounds{inbound, webInbound},
	})

	handler := &handler{}

	dispatcher.Register(streaming.BuildHelloYARPCProcedures(handler))

	if err := dispatcher.Start(); err != nil {
		return err
	}
	fmt.Println("started!")

	select {}
}
