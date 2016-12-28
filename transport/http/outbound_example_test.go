package http_test

import (
	"go.uber.org/yarpc"
	"go.uber.org/yarpc/transport/http"
)

func ExampleOutbound() {
	transport := http.NewTransport()

	yarpc.NewDispatcher(yarpc.Config{
		Name: "myservice",
		Outbounds: yarpc.Outbounds{
			"myservice": {
				Unary: transport.NewSingleOutbound("http://127.0.0.1:8888"),
			},
			"anotherservice": {
				Unary: transport.NewSingleOutbound("http://127.0.0.1:9999"),
			},
		},
	})
}
