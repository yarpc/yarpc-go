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

	barReqBody := fmt.Sprintf("Request to bar with value %d", time.Now().Unix())
	sendRequest(client, "bar", barReqBody)

	mooReqBody := fmt.Sprintf("Request to moo with value %d", time.Now().Unix())
	sendRequest(client, "moo", mooReqBody)
}

func sendRequest(client raw.Client, procedure, msgBody string) {
	randDuration := time.Now().Unix()%100 + 1
	randTimeout := time.Duration(randDuration) * time.Second
	headers := yarpc.NewHeaders().With("from", "self")

	fmt.Println("---Sending a request---")
	fmt.Println("Timeout: ", randTimeout)
	fmt.Println("Caller: hello")
	fmt.Println("Encoding: raw")
	fmt.Println("Service: foo")
	fmt.Println("Procedure: ", procedure)
	fmt.Println("headers: ", headers)
	fmt.Println("Body: ", msgBody)

	ctx, _ := context.WithTimeout(context.Background(), randTimeout)
	ctx = yarpc.WithBaggage(ctx, "token", "42")

	resBody, resMeta, err := client.Call(
		ctx,
		yarpc.NewReqMeta().Procedure(procedure).Headers(headers),
		[]byte(msgBody),
	)
	if err != nil {
		log.Fatalf("call failed: %v", err)
	}

	fmt.Println("SUCCESS! Got response: ", string(resBody))
	fmt.Println("With Headers: ", resMeta.Headers())
	fmt.Println("---Finished request---")
}
