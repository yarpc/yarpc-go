package main

import (
	"log"

	yarpc "github.com/yarpc/yarpc-go"
	"github.com/yarpc/yarpc-go/transport"
	"github.com/yarpc/yarpc-go/transport/grpc"
)

func main() {
	dispatcher := yarpc.NewDispatcher(yarpc.Config{
		Name: "hello",
		Inbounds: []transport.Inbound{
			grpc.NewInbound(50014),
		},
	})

	if err := dispatcher.Start(); err != nil {
		log.Fatal(err)
	}
	defer dispatcher.Stop()

	select {}
}
