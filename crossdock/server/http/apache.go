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
	"log"
	"net/http"
	"time"

	"github.com/yarpc/yarpc-go/crossdock/thrift/gen-go/gauntlet_apache"

	"github.com/apache/thrift/lib/go/thrift"
)

func startApacheThrift() {
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
	mux.HandleFunc("/thrift/ThriftTest", thrift.NewThriftHandlerFunc(thriftTest, pfactory, pfactory))
	mux.HandleFunc("/thrift/SecondService", thrift.NewThriftHandlerFunc(secondService, pfactory, pfactory))
	mux.HandleFunc("/thrift/multiplexed", thrift.NewThriftHandlerFunc(multiplexed, pfactory, pfactory))

	server := &http.Server{
		Addr:         apacheThriftAddr,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil {
			log.Println("error:", err.Error())
		}
	}()
}
