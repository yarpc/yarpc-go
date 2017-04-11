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

package grpc

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"go.uber.org/yarpc/api/transport"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

func TestGetServiceDescs(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		Name                 string
		Procedures           []transport.Procedure
		ExpectedServiceDescs []*grpc.ServiceDesc
	}{
		{
			Name: "Basic",
			Procedures: []transport.Procedure{
				{
					Name:    "KeyValue::GetValue",
					Service: "Example",
				},
			},
			ExpectedServiceDescs: []*grpc.ServiceDesc{
				{
					ServiceName: "KeyValue",
					Methods: []grpc.MethodDesc{
						{
							MethodName: "GetValue",
						},
					},
				},
			},
		},
		{
			Name: "Two Methods",
			Procedures: []transport.Procedure{
				{
					Name:    "KeyValue::GetValue",
					Service: "Example",
				},
				{
					Name:    "KeyValue::SetValue",
					Service: "Example",
				},
			},
			ExpectedServiceDescs: []*grpc.ServiceDesc{
				{
					ServiceName: "KeyValue",
					Methods: []grpc.MethodDesc{
						{
							MethodName: "GetValue",
						},
						{
							MethodName: "SetValue",
						},
					},
				},
			},
		},
		{
			Name: "Two Services",
			Procedures: []transport.Procedure{
				{
					Name:    "KeyValue::GetValue",
					Service: "Example",
				},
				{
					Name:    "Sink::GetValue",
					Service: "Example",
				},
			},
			ExpectedServiceDescs: []*grpc.ServiceDesc{
				{
					ServiceName: "KeyValue",
					Methods: []grpc.MethodDesc{
						{
							MethodName: "GetValue",
						},
					},
				},
				{
					ServiceName: "Sink",
					Methods: []grpc.MethodDesc{
						{
							MethodName: "GetValue",
						},
					},
				},
			},
		},
		{
			Name: "Default Service Name",
			Procedures: []transport.Procedure{
				{
					Name:    "Foo",
					Service: "Example",
				},
			},
			ExpectedServiceDescs: []*grpc.ServiceDesc{
				{
					ServiceName: defaultServiceName,
					Methods: []grpc.MethodDesc{
						{
							MethodName: "Foo",
						},
					},
				},
			},
		},
	} {
		t.Run(tt.Name, func(t *testing.T) {
			inbound := NewInbound(nil)
			inbound.SetRouter(newTestTransportRouter(tt.Procedures))
			serviceDescs, err := inbound.getServiceDescs()
			require.NoError(t, err)
			testServiceDescServiceNamesCovered(t, serviceDescs, tt.ExpectedServiceDescs)
			for _, expectedServiceDesc := range tt.ExpectedServiceDescs {
				serviceDesc := testFindServiceDesc(t, serviceDescs, expectedServiceDesc.ServiceName)
				testServiceDescMethodNamesCovered(t, serviceDesc, expectedServiceDesc)
			}
		})
	}
}

type testTransportRouter struct {
	procedures []transport.Procedure
}

func newTestTransportRouter(procedures []transport.Procedure) *testTransportRouter {
	return &testTransportRouter{procedures}
}

func (t *testTransportRouter) Procedures() []transport.Procedure {
	return t.procedures
}

func (t *testTransportRouter) Choose(_ context.Context, _ *transport.Request) (transport.HandlerSpec, error) {
	return transport.HandlerSpec{}, errors.New("not implemented")
}

func testFindServiceDesc(t *testing.T, serviceDescs []*grpc.ServiceDesc, serviceName string) *grpc.ServiceDesc {
	for _, serviceDesc := range serviceDescs {
		if serviceDesc.ServiceName == serviceName {
			return serviceDesc
		}
	}
	require.NoError(t, fmt.Errorf("could not find *grpc.ServiceDesc with ServiceName %s", serviceName))
	return nil
}

func testServiceDescServiceNamesCovered(t *testing.T, serviceDescs []*grpc.ServiceDesc, expectedServiceDescs []*grpc.ServiceDesc) {
	require.Equal(t, len(expectedServiceDescs), len(serviceDescs))
	m := make(map[string]bool)
	n := make(map[string]bool)
	for _, serviceDesc := range expectedServiceDescs {
		m[serviceDesc.ServiceName] = true
	}
	for _, serviceDesc := range serviceDescs {
		n[serviceDesc.ServiceName] = true
	}
	require.Equal(t, m, n)
}

func testServiceDescMethodNamesCovered(t *testing.T, serviceDesc *grpc.ServiceDesc, expectedServiceDesc *grpc.ServiceDesc) {
	methodDescs := serviceDesc.Methods
	expectedMethodDescs := expectedServiceDesc.Methods
	require.Equal(t, len(expectedMethodDescs), len(methodDescs))
	m := make(map[string]bool)
	n := make(map[string]bool)
	for _, methodDesc := range expectedMethodDescs {
		m[methodDesc.MethodName] = true
	}
	for _, methodDesc := range methodDescs {
		n[methodDesc.MethodName] = true
	}
	require.Equal(t, m, n)
}
