// Copyright (c) 2017 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package http_test

import (
	"fmt"
	"io"
	"log"
	nethttp "net/http"
	"os"

	"go.uber.org/yarpc"
	"go.uber.org/yarpc/internal/iopool"
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

	if _, err := iopool.Copy(os.Stdout, res.Body); err != nil {
		log.Fatal(err)
	}
	// Output: hello from /health
}

func ExampleInterceptor() {
	// import nethttp "net/http"

	// We create an interceptor which executes an override http.Handler if
	// the HTTP request is not a well-formed YARPC request.
	override := func(overrideHandler nethttp.Handler) http.InboundOption {
		return http.Interceptor(func(transportHandler nethttp.Handler) nethttp.Handler {
			return nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
				if r.Header.Get("RPC-Encoding") == "" {
					overrideHandler.ServeHTTP(w, r)
				} else {
					transportHandler.ServeHTTP(w, r)
				}
			})
		})
	}

	// This handler represents some existing http.Handler.
	handler := nethttp.HandlerFunc(func(w nethttp.ResponseWriter, req *nethttp.Request) {
		io.WriteString(w, "hello, world!")
	})

	// We create the inbound, attaching the override interceptor.
	transport := http.NewTransport()
	inbound := transport.NewInbound(":8889", override(handler))

	// Fire up a dispatcher with the new inbound.
	dispatcher := yarpc.NewDispatcher(yarpc.Config{
		Name:     "server",
		Inbounds: yarpc.Inbounds{inbound},
	})
	if err := dispatcher.Start(); err != nil {
		log.Fatal(err)
	}
	defer dispatcher.Stop()

	// Make a non-YARPC request to the / endpoint.
	res, err := nethttp.Get("http://127.0.0.1:8889/")
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()

	if _, err := iopool.Copy(os.Stdout, res.Body); err != nil {
		log.Fatal(err)
	}
	// Output: hello, world!
}
