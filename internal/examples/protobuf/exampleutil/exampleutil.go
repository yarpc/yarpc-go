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

package exampleutil

import (
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/encoding/x/protobuf"
	"go.uber.org/yarpc/internal/examples/protobuf/examplepb"
	"go.uber.org/yarpc/internal/testutils"
	"go.uber.org/yarpc/transport/x/grpc/grpcheader"
)

// Clients holds all clients.
type Clients struct {
	KeyValueYARPCClient     examplepb.KeyValueYARPCClient
	SinkYARPCClient         examplepb.SinkYARPCClient
	KeyValueYARPCJSONClient examplepb.KeyValueYARPCClient
	SinkYARPCJSONClient     examplepb.SinkYARPCClient
	KeyValueGRPCClient      examplepb.KeyValueClient
	SinkGRPCClient          examplepb.SinkClient
	ContextWrapper          *grpcheader.ContextWrapper
}

// WithClients calls f on the Clients.
func WithClients(
	transportType testutils.TransportType,
	keyValueYARPCServer examplepb.KeyValueYARPCServer,
	sinkYARPCServer examplepb.SinkYARPCServer,
	f func(*Clients) error,
) error {
	var procedures []transport.Procedure
	if keyValueYARPCServer != nil {
		procedures = append(procedures, examplepb.BuildKeyValueYARPCProcedures(keyValueYARPCServer)...)
	}
	if sinkYARPCServer != nil {
		procedures = append(procedures, examplepb.BuildSinkYARPCProcedures(sinkYARPCServer)...)
	}
	return testutils.WithClientInfo(
		"example",
		procedures,
		transportType,
		func(clientInfo *testutils.ClientInfo) error {
			return f(
				&Clients{
					examplepb.NewKeyValueYARPCClient(clientInfo.ClientConfig),
					examplepb.NewSinkYARPCClient(clientInfo.ClientConfig),
					examplepb.NewKeyValueYARPCClient(clientInfo.ClientConfig, protobuf.UseJSON),
					examplepb.NewSinkYARPCClient(clientInfo.ClientConfig, protobuf.UseJSON),
					examplepb.NewKeyValueClient(clientInfo.GRPCClientConn),
					examplepb.NewSinkClient(clientInfo.GRPCClientConn),
					clientInfo.ContextWrapper,
				},
			)
		},
	)
}
