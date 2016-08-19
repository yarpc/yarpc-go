// Copyright (c) 2016 Uber Technologies, Inc.
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

package http

import (
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

var server *http.Server
var serveMux *yarpcHTTPMux

// Start starts an http server that yarpc client will make requests to
func Start() {
	serveMux = &yarpcHTTPMux{
		handlers: make(map[string]http.Handler),
	}
	server = &http.Server{
		Addr:         ":8085",
		Handler:      serveMux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}

	register(serveMux)

	go func() {
		if err := server.ListenAndServe(); err != nil {
			log.Println("error:", err.Error())
		}
	}()
}

func register(mux *yarpcHTTPMux) {
	mux.HandleFunc("handlertimeout/raw", handlerTimeoutRawHandler)
}

type yarpcHTTPMux struct {
	sync.RWMutex
	handlers map[string]http.Handler
}

func (m *yarpcHTTPMux) HandleFunc(procedure string, f http.HandlerFunc) {
	m.Lock()
	defer m.Unlock()
	m.handlers[procedure] = f
}

func (m *yarpcHTTPMux) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	m.RLock()
	defer m.RUnlock()
	if req.Method != `POST` {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Invalid method: %q\n", req.Method)
		return
	}
	procedure := req.Header.Get(`RPC-Procedure`)
	if f, ok := m.handlers[procedure]; ok {
		f.ServeHTTP(w, req)
	} else {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Unknown procedure: %q\n", procedure)
	}
}
