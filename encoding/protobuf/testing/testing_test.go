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
	"context"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/stretchr/testify/assert"
	"go.uber.org/yarpc"
	"go.uber.org/yarpc/internal/examples/protobuf/example"
	"go.uber.org/yarpc/internal/examples/protobuf/examplepb"
	"go.uber.org/yarpc/internal/examples/protobuf/exampleutil"
	"go.uber.org/yarpc/internal/grpcctx"
	"go.uber.org/yarpc/internal/testtime"
	"go.uber.org/yarpc/internal/testutils"
	"go.uber.org/yarpc/yarpcerrors"
)

func TestIntegration(t *testing.T) {
	t.Parallel()
	for _, transportType := range testutils.AllTransportTypes {
		transportType := transportType
		t.Run(transportType.String(), func(t *testing.T) { testIntegrationForTransportType(t, transportType) })
	}
}

func testIntegrationForTransportType(t *testing.T, transportType testutils.TransportType) {
	keyValueYARPCServer := example.NewKeyValueYARPCServer()
	sinkYARPCServer := example.NewSinkYARPCServer(true)
	assert.NoError(
		t,
		exampleutil.WithClients(
			transportType,
			keyValueYARPCServer,
			sinkYARPCServer,
			func(clients *exampleutil.Clients) error {
				testIntegration(t, transportType, clients, keyValueYARPCServer, sinkYARPCServer)
				return nil
			},
		),
	)
}

func testIntegration(
	t *testing.T,
	transportType testutils.TransportType,
	clients *exampleutil.Clients,
	keyValueYARPCServer *example.KeyValueYARPCServer,
	sinkYARPCServer *example.SinkYARPCServer,
) {
	keyValueYARPCServer.SetNextError(yarpcerrors.Newf(yarpcerrors.CodeUnknown, "baz").WithName("foo-bar"))
	err := setValue(clients.KeyValueYARPCClient, "foo", "bar")
	assert.Equal(t, yarpcerrors.Newf(yarpcerrors.CodeUnknown, "baz").WithName("foo-bar"), err)
	keyValueYARPCServer.SetNextError(yarpcerrors.Newf(yarpcerrors.CodeUnknown, "baz").WithName("foo-bar"))
	err = setValueGRPC(clients.KeyValueGRPCClient, clients.ContextWrapper, "foo", "bar")
	assert.Equal(t, status.Error(codes.Unknown, "foo-bar: baz"), err)

	assert.NoError(t, setValue(clients.KeyValueYARPCClient, "foo", ""))

	_, err = getValue(clients.KeyValueYARPCClient, "foo")
	assert.Equal(t, yarpcerrors.Newf(yarpcerrors.CodeNotFound, "foo"), err)
	_, err = getValueGRPC(clients.KeyValueGRPCClient, clients.ContextWrapper, "foo")
	assert.Equal(t, status.Error(codes.NotFound, "foo"), err)
	_, err = getValue(clients.KeyValueYARPCJSONClient, "foo")
	assert.Equal(t, yarpcerrors.Newf(yarpcerrors.CodeNotFound, "foo"), err)

	assert.NoError(t, setValue(clients.KeyValueYARPCClient, "foo", "bar"))
	value, err := getValue(clients.KeyValueYARPCClient, "foo")
	assert.NoError(t, err)
	assert.Equal(t, "bar", value)

	assert.NoError(t, setValue(clients.KeyValueYARPCJSONClient, "foo", "baz"))
	value, err = getValue(clients.KeyValueYARPCJSONClient, "foo")
	assert.NoError(t, err)
	assert.Equal(t, "baz", value)

	//switch transportType {
	//case testutils.TransportTypeGRPC, testutils.TransportTypeTChannel:
	keyValueYARPCServer.SetNextError(yarpcerrors.Newf(yarpcerrors.CodeFailedPrecondition, "baz"))
	value, err = getValue(clients.KeyValueYARPCClient, "foo")
	assert.Equal(t, yarpcerrors.Newf(yarpcerrors.CodeFailedPrecondition, "baz"), err)
	assert.Equal(t, "baz", value)
	//}

	assert.NoError(t, setValueGRPC(clients.KeyValueGRPCClient, clients.ContextWrapper, "foo", "barGRPC"))
	value, err = getValueGRPC(clients.KeyValueGRPCClient, clients.ContextWrapper, "foo")
	assert.NoError(t, err)
	assert.Equal(t, "barGRPC", value)

	assert.NoError(t, setValue(clients.KeyValueYARPCClient, "foo", ""))
	_, err = getValue(clients.KeyValueYARPCClient, "foo")
	assert.Error(t, err)

	assert.NoError(t, setValue(clients.KeyValueYARPCClient, "foo", "baz"))
	assert.NoError(t, setValue(clients.KeyValueYARPCClient, "baz", "bat"))
	value, err = getValue(clients.KeyValueYARPCClient, "foo")
	assert.NoError(t, err)
	assert.Equal(t, "baz", value)
	value, err = getValue(clients.KeyValueYARPCClient, "baz")
	assert.NoError(t, err)
	assert.Equal(t, "bat", value)

	assert.NoError(t, fire(clients.SinkYARPCClient, "foo"))
	assert.NoError(t, sinkYARPCServer.WaitFireDone())
	assert.NoError(t, fire(clients.SinkYARPCClient, "bar"))
	assert.NoError(t, sinkYARPCServer.WaitFireDone())
	assert.NoError(t, fire(clients.SinkYARPCJSONClient, "baz"))
	assert.NoError(t, sinkYARPCServer.WaitFireDone())
	assert.Equal(t, []string{"foo", "bar", "baz"}, sinkYARPCServer.Values())
}

func getValue(keyValueYARPCClient examplepb.KeyValueYARPCClient, key string, options ...yarpc.CallOption) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), testtime.Second)
	defer cancel()
	response, err := keyValueYARPCClient.GetValue(ctx, &examplepb.GetValueRequest{key}, options...)
	if response != nil {
		return response.Value, err
	}
	return "", err
}

func setValue(keyValueYARPCClient examplepb.KeyValueYARPCClient, key string, value string, options ...yarpc.CallOption) error {
	ctx, cancel := context.WithTimeout(context.Background(), testtime.Second)
	defer cancel()
	_, err := keyValueYARPCClient.SetValue(ctx, &examplepb.SetValueRequest{key, value}, options...)
	return err
}

func getValueGRPC(keyValueGRPCClient examplepb.KeyValueClient, contextWrapper *grpcctx.ContextWrapper, key string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), testtime.Second)
	defer cancel()
	response, err := keyValueGRPCClient.GetValue(contextWrapper.Wrap(ctx), &examplepb.GetValueRequest{key})
	if response != nil {
		return response.Value, err
	}
	return "", err
}

func setValueGRPC(keyValueGRPCClient examplepb.KeyValueClient, contextWrapper *grpcctx.ContextWrapper, key string, value string) error {
	ctx, cancel := context.WithTimeout(context.Background(), testtime.Second)
	defer cancel()
	_, err := keyValueGRPCClient.SetValue(contextWrapper.Wrap(ctx), &examplepb.SetValueRequest{key, value})
	return err
}

func fire(sinkYARPCClient examplepb.SinkYARPCClient, value string, options ...yarpc.CallOption) error {
	ctx, cancel := context.WithTimeout(context.Background(), testtime.Second)
	defer cancel()
	_, err := sinkYARPCClient.Fire(ctx, &examplepb.FireRequest{value}, options...)
	return err
}
