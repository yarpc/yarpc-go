package main

import (
	"log"

	"fmt"
	"time"

	yarpc "github.com/yarpc/yarpc-go"
	"github.com/yarpc/yarpc-go/encoding/raw"
	"github.com/yarpc/yarpc-go/transport"
	"github.com/yarpc/yarpc-go/transport/grpc"
	"golang.org/x/net/context"
)

func bar(ctx context.Context, reqMeta yarpc.ReqMeta, body []byte) ([]byte, yarpc.ResMeta, error) {
	fmt.Println("---NEW REQUEST TO BAR---")

	printReqInfo(ctx, reqMeta, body)

	res := []byte(fmt.Sprintf("server got request body: %s for Bar", string(body)))

	headers := reqMeta.Headers().With("called_func", "bar")
	resMeta := yarpc.NewResMeta().Headers(headers)

	fmt.Println("---END OF REQUEST TO BAR---")
	return res, resMeta, nil
}

func moo(ctx context.Context, reqMeta yarpc.ReqMeta, body []byte) ([]byte, yarpc.ResMeta, error) {
	fmt.Println("---NEW REQUEST TO MOO---")

	printReqInfo(ctx, reqMeta, body)

	resBody := []byte(fmt.Sprintf("server got request body: %s", string(body)))

	headers := reqMeta.Headers().With("called_func", "moo")
	resMeta := yarpc.NewResMeta().Headers(headers)

	fmt.Println("---END OF REQUEST TO MOO---")
	return resBody, resMeta, nil
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
		Name: "foo",
		Inbounds: []transport.Inbound{
			grpc.NewInbound(50054),
		},
	})

	raw.Register(dispatcher, raw.Procedure("bar", bar))
	raw.Register(dispatcher, raw.Procedure("moo", moo))

	if err := dispatcher.Start(); err != nil {
		log.Fatal(err)
	}
	defer dispatcher.Stop()

	select {}
}
