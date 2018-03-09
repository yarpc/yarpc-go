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

package testing

import (
	"io/ioutil"
	"log"
	"testing"

	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/internal/examples/protobuf/example"
	"go.uber.org/yarpc/internal/examples/protobuf/examplepb"
	"go.uber.org/yarpc/internal/examples/protobuf/exampleutil"
	"go.uber.org/yarpc/internal/grpcctx"
	"go.uber.org/yarpc/internal/testutils"
	"google.golang.org/grpc/grpclog"
)

func init() {
	log.SetOutput(ioutil.Discard)
	grpclog.SetLogger(log.New(ioutil.Discard, "", 0))
}

func BenchmarkIntegrationYARPC(b *testing.B) {
	for _, transportType := range testutils.AllTransportTypes {
		b.Run(transportType.String(), func(b *testing.B) {
			benchmarkForTransportType(b, transportType, func(clients *exampleutil.Clients) error {
				benchmarkIntegrationYARPC(b, clients.KeyValueYARPCClient)
				return nil
			})
		})
	}
}

func BenchmarkIntegrationGRPC(b *testing.B) {
	benchmarkForTransportType(b, testutils.TransportTypeGRPC, func(clients *exampleutil.Clients) error {
		benchmarkIntegrationGRPC(b, clients.KeyValueGRPCClient, clients.ContextWrapper)
		return nil
	})
}

func benchmarkForTransportType(b *testing.B, transportType testutils.TransportType, f func(*exampleutil.Clients) error) {
	keyValueYARPCServer := example.NewKeyValueYARPCServer()
	sinkYARPCServer := example.NewSinkYARPCServer(false)
	fooYARPCServer := example.NewFooYARPCServer(transport.NewHeaders())
	exampleutil.WithClients(transportType, keyValueYARPCServer, sinkYARPCServer, fooYARPCServer, nil, f)
}

func benchmarkIntegrationYARPC(b *testing.B, keyValueYARPCClient examplepb.KeyValueYARPCClient) {
	b.Run("Get", func(b *testing.B) {
		setValue(keyValueYARPCClient, "foo", "bar")
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			getValue(keyValueYARPCClient, "foo")
		}
	})
}

func benchmarkIntegrationGRPC(b *testing.B, keyValueClient examplepb.KeyValueClient, contextWrapper *grpcctx.ContextWrapper) {
	b.Run("Get", func(b *testing.B) {
		setValueGRPC(keyValueClient, contextWrapper, "foo", "bar")
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			getValueGRPC(keyValueClient, contextWrapper, "foo")
		}
	})
}
