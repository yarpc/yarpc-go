package tchannel_test

import (
	"log"

	"go.uber.org/yarpc"
	"go.uber.org/yarpc/transport/tchannel"
)

func ExampleChannelInbound() {
	transport := tchannel.NewChannelTransport(tchannel.ServiceName("myservice"))
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
	transport := tchannel.NewChannelTransport(tchannel.ServiceName("myclient"))
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
	transport := tchannel.NewChannelTransport(tchannel.ServiceName("myclient"))
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
