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

package testing

import (
	"testing"

	"go.uber.org/yarpc/internal/examples/protobuf/example"
	"go.uber.org/yarpc/internal/examples/protobuf/examplepb"
	"go.uber.org/yarpc/internal/testutils"
)

func BenchmarkIntegration(b *testing.B) {
	for _, transportType := range testutils.AllTransportTypes {
		b.Run(transportType.String(), func(b *testing.B) { benchmarkIntegrationForTransportType(b, transportType) })
	}
}

func benchmarkIntegrationForTransportType(b *testing.B, transportType testutils.TransportType) {
	keyValueYarpcServer := example.NewKeyValueYarpcServer()
	sinkYarpcServer := example.NewSinkYarpcServer(false)
	example.WithClients(
		transportType,
		keyValueYarpcServer,
		sinkYarpcServer,
		func(clients *example.Clients) error {
			benchmarkIntegration(b, clients.KeyValueYarpcClient, clients.SinkYarpcClient, keyValueYarpcServer, sinkYarpcServer)
			return nil
		},
	)
}

func benchmarkIntegration(
	b *testing.B,
	keyValueYarpcClient examplepb.KeyValueYarpcClient,
	sinkYarpcClient examplepb.SinkYarpcClient,
	keyValueYarpcServer *example.KeyValueYarpcServer,
	sinkYarpcServer *example.SinkYarpcServer,
) {
	b.Run("Get", func(b *testing.B) {
		setValue(keyValueYarpcClient, "foo", "bar")
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			getValue(keyValueYarpcClient, "foo")
		}
	})
}
