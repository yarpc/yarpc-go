// Copyright (c) 2020 Uber Technologies, Inc.
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

func ExampleInterceptor() {
	// import nethttp "net/http"

	// Given a fallback http.Handler
	fallback := nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		io.WriteString(w, "hello, world!")
	})

	// Create an interceptor that falls back to a handler when the HTTP request is
	// missing the RPC-Encoding header.
	intercept := func(transportHandler nethttp.Handler) nethttp.Handler {
		return nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
			if r.Header.Get(http.EncodingHeader) == "" {
				// Not a YARPC request, use fallback handler.
				fallback.ServeHTTP(w, r)
			} else {
				transportHandler.ServeHTTP(w, r)
			}
		})
	}

	// Create a new inbound, attaching the interceptor
	transport := http.NewTransport()
	inbound := transport.NewInbound(":8889", http.Interceptor(intercept))

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

	if _, err := io.Copy(os.Stdout, res.Body); err != nil {
		log.Fatal(err)
	}
	// Output: hello, world!
}
