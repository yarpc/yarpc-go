package main

import (
	"fmt"
	"log"
	"time"

	"golang.org/x/net/context"

	yarpc "github.com/yarpc/yarpc-go"
	"github.com/yarpc/yarpc-go/encoding/raw"
	"github.com/yarpc/yarpc-go/transport"
	"github.com/yarpc/yarpc-go/transport/grpc"
)

func main() {
	dispatcher := yarpc.NewDispatcher(yarpc.Config{
		Name: "hello",
		Outbounds: transport.Outbounds{
			"foo": grpc.NewOutbound("localhost:50054"),
		},
	})

	if err := dispatcher.Start(); err != nil {
		log.Fatal(err)
	}
	defer dispatcher.Stop()

	client := raw.New(dispatcher.Channel("foo"))

	ctx, _ := context.WithTimeout(context.Background(), time.Second)
	resBody, _, err := client.Call(yarpc.NewReqMeta(ctx).Procedure("bar"), []byte("hi"))
	if err != nil {
		log.Fatalf("call failed: %v", err)
	}

	fmt.Println("SUCCESS!", string(resBody))
}
