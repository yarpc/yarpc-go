package main

import (
	"context"
	"fmt"
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

	// create a dispatcher for the "server" service
	dispatcher, err := configurator.NewDispatcherFromYAML("server", strings.NewReader(yaml))
	if err != nil {
		log.Panicf("Dispatcher could not be created: %v", err)
	}

	// register handler
	procedures := hello.BuildHelloWorldYARPCProcedures(handler{})
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

func (handler) Hello(ctx context.Context, req *hello.HelloRequest) (*hello.HelloResponse, error) {
	message := fmt.Sprintf("Hello %s!", req.Name)
	return &hello.HelloResponse{Message: message}, nil
}
