// Copyright (c) 2018 Uber Technologies, Inc.
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

package yarpchttp_test

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"go.uber.org/yarpc/v2"
	"go.uber.org/yarpc/v2/yarpchttp"
	"go.uber.org/yarpc/v2/yarpcrouter"
)

func ExampleInbound() {
	router := yarpcrouter.NewMapRouter("my-service", []yarpc.TransportProcedure{})
	inbound := &yarpchttp.Inbound{
		Addr:   ":8888",
		Router: router,
	}
	if err := inbound.Start(context.Background()); err != nil {
		log.Fatal(err)
	}
	defer inbound.Stop(context.Background())
}

func ExampleMux() {
	// We set up a ServeMux which provides a /health endpoint.
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		if _, err := fmt.Fprintln(w, "hello from /health"); err != nil {
			panic(err)
		}
	})

	router := yarpcrouter.NewMapRouter("my-service", []yarpc.TransportProcedure{})
	inbound := &yarpchttp.Inbound{
		Addr:       ":8888",
		Router:     router,
		Mux:        mux,
		MuxPattern: "/yarpc",
	}
	if err := inbound.Start(context.Background()); err != nil {
		log.Fatal(err)
	}
	defer inbound.Stop(context.Background())

	// Make a request to the /health endpoint.
	res, err := http.Get("http://127.0.0.1:8888/health")
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
	// Given a fallback yarpchttp.Handler
	fallback := http.HandlerFunc(func(w http.ResponseWriter, httpReq *http.Request) {
		io.WriteString(w, "hello, world!")
	})

	// Create an interceptor that falls back to a handler when the HTTP request is
	// missing the RPC-Encoding header.
	intercept := func(transportHandler http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, httpReq *http.Request) {
			if httpReq.Header.Get(yarpchttp.EncodingHeader) == "" {
				// Not a YARPC request, use fallback handler.
				fallback.ServeHTTP(w, httpReq)
			} else {
				transportHandler.ServeHTTP(w, httpReq)
			}
		})
	}

	// Create a new inbound, attaching the interceptor
	router := yarpcrouter.NewMapRouter("server", []yarpc.TransportProcedure{})
	inbound := &yarpchttp.Inbound{
		Addr:        ":8889",
		Router:      router,
		Interceptor: intercept,
	}
	if err := inbound.Start(context.Background()); err != nil {
		log.Fatal(err)
	}
	defer inbound.Stop(context.Background())

	// Make a non-YARPC request to the / endpoint.
	res, err := http.Get("http://127.0.0.1:8889/")
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()

	if _, err := io.Copy(os.Stdout, res.Body); err != nil {
		log.Fatal(err)
	}
	// Output: hello, world!
}
