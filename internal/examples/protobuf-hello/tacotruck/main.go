package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	hello "go.uber.org/yarpc/internal/examples/protobuf-hello"
	"go.uber.org/yarpc/transport/http"
	"go.uber.org/yarpc/x/config"
)

const yaml = `
inbounds:
  http:
    address: ":8080"
`

func main() {
	// build a configurator with the HTTP transport registered
	configurator := config.New()
	configurator.MustRegisterTransport(http.TransportSpec())

	// create a dispatcher for the tacotruck service
	dispatcher, err := configurator.NewDispatcherFromYAML("tacotruck", strings.NewReader(yaml))
	if err != nil {
		log.Panicf("Dispatcher could not be created: %v", err)
	}

	// register handler
	procedures := hello.BuildTacoTruckYARPCProcedures(handler{})
	dispatcher.Register(procedures)

	// start service
	dispatcher.Start()
	defer dispatcher.Stop()

	// block until SIGINT/SIGTERM
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	<-signals
}

type handler struct{}

func (handler) Order(ctx context.Context, req *hello.OrderRequest) (*hello.OrderResponse, error) {
	message := "Thanks for your order, "
	if req.Order == nil || req.Order.Type == hello.ORDER_TYPE_NONE {
		message = message + "but you didn't order anything."
	} else if req.Order.Type == hello.ORDER_TYPE_TACO {
		message = message + "the taco is the most popular chowdom!"

	} else if req.Order.Type == hello.ORDER_TYPE_BURRITO {
		message = message + "here's your burrito."
	}

	return &hello.OrderResponse{Message: message}, nil
}
