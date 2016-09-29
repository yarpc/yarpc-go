package main

import (
	"fmt"
	"log"
	"time"

	"go.uber.org/yarpc"
	"go.uber.org/yarpc/encoding/thrift"
	"go.uber.org/yarpc/examples/thrift/hello/thrift/hello"
	"go.uber.org/yarpc/examples/thrift/hello/thrift/hello/yarpc/helloclient"
	"go.uber.org/yarpc/examples/thrift/hello/thrift/hello/yarpc/helloserver"
	"go.uber.org/yarpc/transport"
	"go.uber.org/yarpc/transport/grpc"

	"golang.org/x/net/context"
)

//go:generate thriftrw-go --out thrift --plugin=yarpc hello.thrift

func main() {
	dispatcher := yarpc.NewDispatcher(yarpc.Config{
		Name: "hello",
		Inbounds: []transport.Inbound{
			grpc.NewInbound(8086),
		},
		Outbounds: transport.Outbounds{
			"hello": grpc.NewOutbound("localhost:8086"),
		},
	})

	thrift.Register(dispatcher, helloserver.New(&helloHandler{}))
	client := helloclient.New(dispatcher.Channel("hello"))

	if err := dispatcher.Start(); err != nil {
		log.Fatal(err)
	}
	defer dispatcher.Stop()

	response, headers := call(client, "Hi There")
	fmt.Println(response, headers)

	select {}
}

type helloHandler struct{}

func (h helloHandler) Echo(ctx context.Context, reqMeta yarpc.ReqMeta, echo *hello.EchoRequest) (*hello.EchoResponse, yarpc.ResMeta, error) {
	return &hello.EchoResponse{Message: echo.Message, Count: echo.Count + 1},
		yarpc.NewResMeta().Headers(reqMeta.Headers()),
		nil
}

func call(client helloclient.Interface, message string) (*hello.EchoResponse, yarpc.Headers) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	resBody, resMeta, err := client.Echo(
		ctx,
		yarpc.NewReqMeta(),
		&hello.EchoRequest{Message: message, Count: 1},
	)
	if err != nil {
		log.Fatal(err)
	}

	return resBody, resMeta.Headers()
}
