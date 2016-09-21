package main

import (
	"log"

	"fmt"

	yarpc "github.com/yarpc/yarpc-go"
	"github.com/yarpc/yarpc-go/encoding/raw"
	"github.com/yarpc/yarpc-go/transport"
	"github.com/yarpc/yarpc-go/transport/grpc"
	"golang.org/x/net/context"
)

func bar(ctx context.Context, reqMeta yarpc.ReqMeta, body []byte) ([]byte, yarpc.ResMeta, error) {
	fmt.Printf("procedure called with %v, %v", reqMeta, body)
	res := []byte(fmt.Sprintf("server got request body: %s", string(body)))
	return res, nil, nil
}

func main() {
	dispatcher := yarpc.NewDispatcher(yarpc.Config{
		Name: "foo",
		Inbounds: []transport.Inbound{
			grpc.NewInbound(50054),
		},
	})

	raw.Register(dispatcher, raw.Procedure("bar", bar))

	if err := dispatcher.Start(); err != nil {
		log.Fatal(err)
	}
	defer dispatcher.Stop()

	select {}
}
