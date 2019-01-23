// Copyright (c) 2019 Uber Technologies, Inc.
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

package apachethrift

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/apache/thrift/lib/go/thrift"
	"go.uber.org/yarpc/internal/crossdock/thrift/gen-go/gauntlet_apache"
	"go.uber.org/yarpc/internal/net"
)

const addr = ":8088"

var server *net.HTTPServer

// Start starts an Apache Thrift server on port 8088
func Start() {
	// We expose the following endpoints:
	// /thrift/ThriftTest:
	//   Thrift service using TBinaryProtocol
	// /thrift/SecondService
	//   Thrift service using TBinaryProtocol
	// /thrift/mutliplexed
	//   Thrift service using TBinaryProtocol with TMultiplexedProtocol,
	//   serving both, ThriftTest and SecondService

	pfactory := thrift.NewTBinaryProtocolFactoryDefault()

	thriftTest := gauntlet_apache.NewThriftTestProcessor(thriftTestHandler{})
	secondService := gauntlet_apache.NewSecondServiceProcessor(secondServiceHandler{})

	multiplexed := thrift.NewTMultiplexedProcessor()
	multiplexed.RegisterProcessor("ThriftTest", thriftTest)
	multiplexed.RegisterProcessor("SecondService", secondService)

	mux := http.NewServeMux()
	mux.HandleFunc("/thrift/ThriftTest", newThriftHandlerFunc(thriftTest, pfactory, pfactory))
	mux.HandleFunc("/thrift/SecondService", newThriftHandlerFunc(secondService, pfactory, pfactory))
	mux.HandleFunc("/thrift/multiplexed", newThriftHandlerFunc(multiplexed, pfactory, pfactory))

	server = net.NewHTTPServer(&http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	})

	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("failed to start Apache Thrift server: %v", err)
	}
}

// Stop stops the Apache Thrift server.
func Stop() {
	if err := server.Shutdown(context.Background()); err != nil {
		log.Printf("failed to stop Apache Thrift server: %v", err)
	}
}

func newThriftHandlerFunc(processor thrift.TProcessor, inPfactory, outPfactory thrift.TProtocolFactory) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/x-thrift")
		transport := thrift.NewStreamTransport(r.Body, w)
		_, _ = processor.Process(inPfactory.GetProtocol(transport), outPfactory.GetProtocol(transport))
	}
}
