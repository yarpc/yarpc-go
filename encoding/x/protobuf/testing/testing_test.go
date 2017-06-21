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

	"github.com/stretchr/testify/assert"
	"go.uber.org/yarpc"
	"go.uber.org/yarpc/internal/examples/protobuf/example"
	"go.uber.org/yarpc/internal/examples/protobuf/examplepb"
	"go.uber.org/yarpc/internal/examples/protobuf/exampleutil"
	"go.uber.org/yarpc/internal/testtime"
	"go.uber.org/yarpc/internal/testutils"
	"go.uber.org/yarpc/transport/x/grpc/grpcheader"
)

func TestIntegration(t *testing.T) {
	t.Parallel()
	for _, transportType := range testutils.AllTransportTypes {
		transportType := transportType
		t.Run(transportType.String(), func(t *testing.T) { testIntegrationForTransportType(t, transportType) })
	}
}

func testIntegrationForTransportType(t *testing.T, transportType testutils.TransportType) {
	keyValueYarpcServer := example.NewKeyValueYarpcServer()
	sinkYarpcServer := example.NewSinkYarpcServer(true)
	assert.NoError(
		t,
		exampleutil.WithClients(
			transportType,
			keyValueYarpcServer,
			sinkYarpcServer,
			func(clients *exampleutil.Clients) error {
				testIntegration(t, clients, keyValueYarpcServer, sinkYarpcServer)
				return nil
			},
		),
	)
}

func testIntegration(
	t *testing.T,
	clients *exampleutil.Clients,
	keyValueYarpcServer *example.KeyValueYarpcServer,
	sinkYarpcServer *example.SinkYarpcServer,
) {
	_, err := getValue(clients.KeyValueYarpcClient, "foo")
	assert.Error(t, err)
	_, err = getValueGRPC(clients.KeyValueGRPCClient, clients.ContextWrapper, "foo")
	assert.Error(t, err)
	_, err = getValue(clients.KeyValueYarpcJSONClient, "foo")
	assert.Error(t, err)

	assert.NoError(t, setValue(clients.KeyValueYarpcClient, "foo", "bar"))
	value, err := getValue(clients.KeyValueYarpcClient, "foo")
	assert.NoError(t, err)
	assert.Equal(t, "bar", value)

	assert.NoError(t, setValue(clients.KeyValueYarpcJSONClient, "foo", "baz"))
	value, err = getValue(clients.KeyValueYarpcJSONClient, "foo")
	assert.NoError(t, err)
	assert.Equal(t, "baz", value)

	assert.NoError(t, setValueGRPC(clients.KeyValueGRPCClient, clients.ContextWrapper, "foo", "barGRPC"))
	value, err = getValueGRPC(clients.KeyValueGRPCClient, clients.ContextWrapper, "foo")
	assert.NoError(t, err)
	assert.Equal(t, "barGRPC", value)

	assert.NoError(t, setValue(clients.KeyValueYarpcClient, "foo", ""))
	_, err = getValue(clients.KeyValueYarpcClient, "foo")
	assert.Error(t, err)

	assert.NoError(t, setValue(clients.KeyValueYarpcClient, "foo", "baz"))
	assert.NoError(t, setValue(clients.KeyValueYarpcClient, "baz", "bat"))
	value, err = getValue(clients.KeyValueYarpcClient, "foo")
	assert.NoError(t, err)
	assert.Equal(t, "baz", value)
	value, err = getValue(clients.KeyValueYarpcClient, "baz")
	assert.NoError(t, err)
	assert.Equal(t, "bat", value)

	assert.NoError(t, fire(clients.SinkYarpcClient, "foo"))
	assert.NoError(t, sinkYarpcServer.WaitFireDone())
	assert.NoError(t, fire(clients.SinkYarpcClient, "bar"))
	assert.NoError(t, sinkYarpcServer.WaitFireDone())
	assert.NoError(t, fire(clients.SinkYarpcJSONClient, "baz"))
	assert.NoError(t, sinkYarpcServer.WaitFireDone())
	assert.Equal(t, []string{"foo", "bar", "baz"}, sinkYarpcServer.Values())
}

func getValue(keyValueYarpcClient examplepb.KeyValueYarpcClient, key string, options ...yarpc.CallOption) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), testtime.Second)
	defer cancel()
	response, err := keyValueYarpcClient.GetValue(ctx, &examplepb.GetValueRequest{key}, options...)
	if err != nil {
		return "", err
	}
	return response.Value, nil
}

func setValue(keyValueYarpcClient examplepb.KeyValueYarpcClient, key string, value string, options ...yarpc.CallOption) error {
	ctx, cancel := context.WithTimeout(context.Background(), testtime.Second)
	defer cancel()
	_, err := keyValueYarpcClient.SetValue(ctx, &examplepb.SetValueRequest{key, value}, options...)
	return err
}

func getValueGRPC(keyValueGRPCClient examplepb.KeyValueClient, contextWrapper *grpcheader.ContextWrapper, key string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), testtime.Second)
	defer cancel()
	response, err := keyValueGRPCClient.GetValue(contextWrapper.Wrap(ctx), &examplepb.GetValueRequest{key})
	if err != nil {
		return "", err
	}
	return response.Value, nil
}

func setValueGRPC(keyValueGRPCClient examplepb.KeyValueClient, contextWrapper *grpcheader.ContextWrapper, key string, value string) error {
	ctx, cancel := context.WithTimeout(context.Background(), testtime.Second)
	defer cancel()
	_, err := keyValueGRPCClient.SetValue(contextWrapper.Wrap(ctx), &examplepb.SetValueRequest{key, value})
	return err
}

func fire(sinkYarpcClient examplepb.SinkYarpcClient, value string, options ...yarpc.CallOption) error {
	ctx, cancel := context.WithTimeout(context.Background(), testtime.Second)
	defer cancel()
	_, err := sinkYarpcClient.Fire(ctx, &examplepb.FireRequest{value}, options...)
	return err
}
