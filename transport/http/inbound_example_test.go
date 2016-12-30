package http_test

import (
	"fmt"
	"io"
	"log"
	nethttp "net/http"
	"os"

	"go.uber.org/yarpc"
	"go.uber.org/yarpc/transport/http"
)

func ExampleInbound() {
	transport := http.NewTransport()
	inbound := transport.NewInbound(":8888")

	dispatcher := yarpc.NewDispatcher(yarpc.Config{
		Name:     "myservice",
		Inbounds: yarpc.Inbounds{inbound},
	})
	if err := dispatcher.Start(); err != nil {
		log.Fatal(err)
	}
	defer dispatcher.Stop()
}

func ExampleMux() {
	// import nethttp "net/http"

	// We set up a ServeMux which provides a /health endpoint.
	mux := nethttp.NewServeMux()
	mux.HandleFunc("/health", func(w nethttp.ResponseWriter, _ *nethttp.Request) {
		if _, err := fmt.Fprintln(w, "hello from /health"); err != nil {
			panic(err)
		}
	})

	// This inbound will serve the YARPC service on the path /yarpc.  The
	// /health endpoint on the Mux will be left alone.
	transport := http.NewTransport()
	inbound := transport.NewInbound(":8888", http.Mux("/yarpc", mux))

	// Fire up a dispatcher with the new inbound.
	dispatcher := yarpc.NewDispatcher(yarpc.Config{
		Name:     "server",
		Inbounds: yarpc.Inbounds{inbound},
	})
	if err := dispatcher.Start(); err != nil {
		log.Fatal(err)
	}
	defer dispatcher.Stop()

	// Make a request to the /health endpoint.
	res, err := nethttp.Get("http://127.0.0.1:8888/health")
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()

	if _, err := io.Copy(os.Stdout, res.Body); err != nil {
		log.Fatal(err)
	}
	// Output: hello from /health
}
