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
	"fmt"
	"log"
	"net"

	"go.uber.org/yarpc"
	"go.uber.org/yarpc/api/transport"
	//"go.uber.org/yarpc/encoding/raw"
	"context"
	"errors"
	"go.uber.org/yarpc/internal/examples/streaming"
	"go.uber.org/yarpc/transport/x/grpc"
)

//
//type handler struct {
//}
//
//func (h *handler) handle(ss *raw.ServerStream) error {
//	fmt.Println("Handling stream")
//	for {
//		msg, err := ss.Receive()
//		if err != nil {
//			fmt.Printf("Error reading from stream: %q\n", err.Error())
//			return err
//		}
//
//		if string(msg) == "exit" {
//			fmt.Println("Received 'exit' message, closing connection")
//			return nil
//		}
//		fmt.Printf("Received a message: %q\n", string(msg))
//
//		resp := fmt.Sprintf("Got your message: %q, thanks!", string(msg))
//		if err := ss.Send([]byte(resp)); err != nil {
//			fmt.Printf("Error sending message to stream: %q\n", err.Error())
//			return err
//		}
//		fmt.Printf("Sent response message: %q\n", resp)
//	}
//}

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
	fmt.Println("Handling stream")
	for {
		msg, err := stream.Recv()
		if err != nil {
			fmt.Printf("Error reading from stream: %q\n", err.Error())
			return err
		}

		if msg.Id == "exit" {
			fmt.Println("Received 'exit' message, closing connection")
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
	listener, err := net.Listen("tcp", "127.0.0.1:24038")
	if err != nil {
		return err
	}
	inbound = grpc.NewTransport().NewInbound(listener)

	dispatcher := yarpc.NewDispatcher(yarpc.Config{
		Name:     "keyvalue",
		Inbounds: yarpc.Inbounds{inbound},
	})

	handler := &handler{}

	dispatcher.Register(streaming.BuildHelloYARPCProcedures(handler))

	fmt.Println("Starting Dispatcher")
	if err := dispatcher.Start(); err != nil {
		return err
	}
	fmt.Println("Started Dispatcher")

	select {}
}
