package main

import (
	"log"

	"fmt"
	"time"

	"go.uber.org/yarpc"
	"go.uber.org/yarpc/encoding/raw"
	"go.uber.org/yarpc/transport"
	"go.uber.org/yarpc/transport/grpc"
	"golang.org/x/net/context"
)

func yarpcFunc(ctx context.Context, reqMeta yarpc.ReqMeta, body []byte) ([]byte, yarpc.ResMeta, error) {
	fmt.Println("---NEW REQUEST TO YARPC---")

	printReqInfo(ctx, reqMeta, body)

	res := []byte(fmt.Sprintf("server got request body: %s for YARPC", string(body)))

	fmt.Println("---END OF REQUEST TO YARPC---")
	return res, nil, nil
}

func printReqInfo(ctx context.Context, reqMeta yarpc.ReqMeta, body []byte) {
	if dl, ok := ctx.Deadline(); ok {
		fmt.Println("Timeout: ", dl.Sub(time.Now()))
	} else {
		fmt.Println("no deadline")
	}

	fmt.Println("Caller: ", reqMeta.Caller())
	fmt.Println("Encoding: ", reqMeta.Encoding())
	fmt.Println("Procedure: ", reqMeta.Procedure())
	fmt.Println("Service: ", reqMeta.Service())
	fmt.Println("Headers: ", reqMeta.Headers())
	fmt.Println("Body: ", string(body))
}

func main() {
	dispatcher := yarpc.NewDispatcher(yarpc.Config{
		Name: "yarpc",
		Inbounds: []transport.Inbound{
			grpc.NewInbound(50054),
		},
	})

	// TODO support non-default procedure names
	raw.Register(dispatcher, raw.Procedure("yarpc", yarpcFunc))

	if err := dispatcher.Start(); err != nil {
		log.Fatal(err)
	}
	defer dispatcher.Stop()

	select {}
}
