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

package internalprotoreflection

import (
	"context"
	"net"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc/v2"
	"go.uber.org/yarpc/v2/yarpcfx/internal/internalprotoreflection/grpc_reflection_v1alpha"
	"go.uber.org/yarpc/v2/yarpcgrpc"
	"go.uber.org/yarpc/v2/yarpctest"
	"google.golang.org/grpc/codes"
)

func TestReflection(t *testing.T) {
	ln, err := net.Listen("tcp", ":0")
	require.NoError(t, err, "failed to listen on a port")
	addr := ln.Addr().String()

	result, err := New(Params{})
	require.NoError(t, err)

	inbound := yarpcgrpc.Inbound{
		Listener: ln,
		Router:   yarpctest.NewFakeRouter(result.Procedures),
	}
	require.NoError(t, inbound.Start(context.Background()))
	defer func() { assert.NoError(t, inbound.Stop(context.Background())) }()

	dialer := &yarpcgrpc.Dialer{}
	outbound := &yarpcgrpc.Outbound{
		URL:    &url.URL{Host: addr},
		Dialer: dialer,
	}
	require.NoError(t, dialer.Start(context.Background()))
	defer func() { assert.NoError(t, inbound.Stop(context.Background())) }()

	client := grpc_reflection_v1alpha.NewServerReflectionYARPCClient(yarpc.Client{
		Name:    "server",
		Service: "server",
		Caller:  "self",
		Stream:  outbound,
	})
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	stream, err := client.ServerReflectionInfo(ctx)
	require.NoError(t, err)
	stream.Send(&grpc_reflection_v1alpha.ServerReflectionRequest{
		MessageRequest: &grpc_reflection_v1alpha.ServerReflectionRequest_ListServices{
			ListServices: "yes pls", // the grpc spec says this just has to be not ""  ¯\_(ツ)_/¯
		},
	})

	response, err := stream.Recv()
	require.NoError(t, err)

	assert.Equal(t, &grpc_reflection_v1alpha.ListServiceResponse{
		Service: []*grpc_reflection_v1alpha.ServiceResponse{
			// {Name: "grpc.health.v1.Health"},
			{Name: "grpc.reflection.v1alpha.ServerReflection"},
		},
	}, response.GetListServicesResponse())

	stream.Send(&grpc_reflection_v1alpha.ServerReflectionRequest{
		MessageRequest: &grpc_reflection_v1alpha.ServerReflectionRequest_FileContainingSymbol{
			FileContainingSymbol: "grpc.reflection.v1alpha.ServerReflection.ServerReflectionInfo",
		},
	})

	response, err = stream.Recv()
	require.NoError(t, err)
	require.NotNil(t, response.GetFileDescriptorResponse())

	stream.Send(&grpc_reflection_v1alpha.ServerReflectionRequest{
		MessageRequest: &grpc_reflection_v1alpha.ServerReflectionRequest_FileContainingSymbol{
			FileContainingSymbol: "not.a.valid.symbol",
		},
	})

	response, err = stream.Recv()
	require.NoError(t, err)

	assert.Equal(t, &grpc_reflection_v1alpha.ErrorResponse{
		ErrorCode:    int32(codes.NotFound),
		ErrorMessage: `could not find descriptor for symbol "not.a.valid.symbol"`,
	}, response.GetErrorResponse())
}
