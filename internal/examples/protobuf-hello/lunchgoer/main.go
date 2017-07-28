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
  tacotruck:
    http:
      url: "http://localhost:8080/"
`

func main() {
	// build a configurator with the HTTP transport registered
	configurator := config.New()
	configurator.MustRegisterTransport(http.TransportSpec())

	// create a dispatcher for the lunchgoer service
	dispatcher, err := configurator.NewDispatcherFromYAML("lunchgoer", strings.NewReader(yaml))
	if err != nil {
		log.Panicf("Dispatcher could not be created: %v", err)
	}

	// create a client for the tacotruck service
	tacotruck := hello.NewTacoTruckYARPCClient(dispatcher.ClientConfig("tacotruck"))

	// start service
	dispatcher.Start()
	defer dispatcher.Stop()

	// prepare a context to call
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// make an rpc to the tacotruck service
	order, err := tacotruck.Order(ctx, &hello.OrderRequest{Order: &hello.Order{Type: hello.ORDER_TYPE_TACO}})
	if err != nil {
		log.Panicf("Could not call tacotruck: %v", err)
	}

	fmt.Printf("You ordered successfully: %v\n", order.Message)
}
