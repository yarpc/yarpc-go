package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	hello "go.uber.org/yarpc/internal/examples/protobuf-hello"
	"go.uber.org/yarpc/transport/http"
	"go.uber.org/yarpc/x/config"
)

const yaml = `
outbounds:
  server:
    http:
      url: "http://localhost:8080/"
`

func main() {
	// build a configurator with the HTTP transport registered
	configurator := config.New()
	configurator.MustRegisterTransport(http.TransportSpec())

	// create a dispatcher for the "client" service
	dispatcher, err := configurator.NewDispatcherFromYAML("client", strings.NewReader(yaml))
	if err != nil {
		log.Panicf("Dispatcher could not be created: %v", err)
	}

	// create a client for the "server" service
	client := hello.NewHelloWorldYARPCClient(dispatcher.ClientConfig("server"))

	// start service
	dispatcher.Start()
	defer dispatcher.Stop()

	// prepare a context to call
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// make an rpc to the "server" service
	resp, err := client.Hello(ctx, &hello.HelloRequest{Name: "client"})
	if err != nil {
		log.Panicf("Could not call server: %v", err)
	}

	fmt.Printf("Called server successfully: %v\n", resp.Message)
}
