package tchannel_test

import (
	"log"

	"go.uber.org/yarpc"
	"go.uber.org/yarpc/transport/tchannel"
)

func ExampleChannelInbound() {
	transport, err := tchannel.NewChannelTransport(tchannel.ServiceName("myservice"))
	if err != nil {
		log.Fatal(err)
	}

	dispatcher := yarpc.NewDispatcher(yarpc.Config{
		Name:     "myservice",
		Inbounds: yarpc.Inbounds{transport.NewInbound()},
	})

	if err := dispatcher.Start(); err != nil {
		log.Fatal(err)
	}
	defer dispatcher.Stop()
}

func ExampleChannelOutbound() {
	transport, err := tchannel.NewChannelTransport(tchannel.ServiceName("myclient"))
	if err != nil {
		log.Fatal(err)
	}

	dispatcher := yarpc.NewDispatcher(yarpc.Config{
		Name: "myclient",
		Outbounds: yarpc.Outbounds{
			"myservice": {Unary: transport.NewOutbound()},
		},
	})

	if err := dispatcher.Start(); err != nil {
		log.Fatal(err)
	}
	defer dispatcher.Stop()
}

func ExampleChannelOutbound_single() {
	transport, err := tchannel.NewChannelTransport(tchannel.ServiceName("myclient"))
	if err != nil {
		log.Fatal(err)
	}

	dispatcher := yarpc.NewDispatcher(yarpc.Config{
		Name: "myclient",
		Outbounds: yarpc.Outbounds{
			"myservice": {Unary: transport.NewSingleOutbound("127.0.0.0:4040")},
		},
	})

	if err := dispatcher.Start(); err != nil {
		log.Fatal(err)
	}
	defer dispatcher.Stop()
}
