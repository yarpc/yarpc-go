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

package testing

import (
	"context"
	"io"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/stretchr/testify/assert"
	"go.uber.org/yarpc"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/internal/examples/protobuf/example"
	"go.uber.org/yarpc/internal/examples/protobuf/examplepb"
	"go.uber.org/yarpc/internal/examples/protobuf/exampleutil"
	"go.uber.org/yarpc/internal/grpcctx"
	"go.uber.org/yarpc/internal/testtime"
	"go.uber.org/yarpc/internal/testutils"
	intyarpcerrors "go.uber.org/yarpc/internal/yarpcerrors"
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
	expectedStreamingHeaders := transport.NewHeaders().With("firstTestKey", "firstTestValue")
	keyValueYARPCServer := example.NewKeyValueYARPCServer()
	sinkYARPCServer := example.NewSinkYARPCServer(true)
	fooYARPCServer := example.NewFooYARPCServer(expectedStreamingHeaders)
	assert.NoError(
		t,
		exampleutil.WithClients(
			transportType,
			keyValueYARPCServer,
			sinkYARPCServer,
			fooYARPCServer,
			nil,
			func(clients *exampleutil.Clients) error {
				testIntegration(t, clients, keyValueYARPCServer, sinkYARPCServer, expectedStreamingHeaders)
				return nil
			},
		),
	)
}

func testIntegration(
	t *testing.T,
	clients *exampleutil.Clients,
	keyValueYARPCServer *example.KeyValueYARPCServer,
	sinkYARPCServer *example.SinkYARPCServer,
	expectedStreamingHeaders transport.Headers,
) {
	keyValueYARPCServer.SetNextError(intyarpcerrors.NewWithNamef(yarpcerrors.CodeUnknown, "foo-bar", "baz"))
	err := setValue(clients.KeyValueYARPCClient, "foo", "bar")
	assert.Equal(t, intyarpcerrors.NewWithNamef(yarpcerrors.CodeUnknown, "foo-bar", "baz"), err)
	keyValueYARPCServer.SetNextError(intyarpcerrors.NewWithNamef(yarpcerrors.CodeUnknown, "foo-bar", "baz"))
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

	keyValueYARPCServer.SetNextError(yarpcerrors.Newf(yarpcerrors.CodeFailedPrecondition, "baz"))
	value, err = getValue(clients.KeyValueYARPCClient, "foo")
	assert.Equal(t, yarpcerrors.Newf(yarpcerrors.CodeFailedPrecondition, "baz"), err)
	assert.Equal(t, "baz", value)

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

	contextWrapper := clients.ContextWrapper
	streamOptions := make([]yarpc.CallOption, 0, expectedStreamingHeaders.Len())
	for k, v := range expectedStreamingHeaders.Items() {
		streamOptions = append(streamOptions, yarpc.WithHeader(k, v))
		contextWrapper = contextWrapper.WithHeader(k, v)
	}

	messages := []string{"foo", "bar", "baz"}
	gotMessages, err := echoOut(clients.FooYARPCClient, messages, streamOptions...)
	assert.NoError(t, err)
	assert.Equal(t, messages, gotMessages)

	gotMessages, err = echoIn(clients.FooYARPCClient, "foo", 3, streamOptions...)
	assert.NoError(t, err)
	assert.Equal(t, []string{"foo", "foo", "foo"}, gotMessages)

	gotMessages, err = echoBoth(clients.FooYARPCClient, "foo", 2, 2, streamOptions...)
	assert.NoError(t, err)
	assert.Equal(t, []string{"foo", "foo", "foo", "foo"}, gotMessages)

	gotMessages, err = echoOutGRPC(clients.FooGRPCClient, contextWrapper, messages)
	assert.NoError(t, err)
	assert.Equal(t, messages, gotMessages)

	gotMessages, err = echoInGRPC(clients.FooGRPCClient, contextWrapper, "foo", 3)
	assert.NoError(t, err)
	assert.Equal(t, []string{"foo", "foo", "foo"}, gotMessages)

	gotMessages, err = echoBothGRPC(clients.FooGRPCClient, contextWrapper, "foo", 2, 2)
	assert.NoError(t, err)
	assert.Equal(t, []string{"foo", "foo", "foo", "foo"}, gotMessages)
}

func getValue(keyValueYARPCClient examplepb.KeyValueYARPCClient, key string, options ...yarpc.CallOption) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), testtime.Second)
	defer cancel()
	response, err := keyValueYARPCClient.GetValue(ctx, &examplepb.GetValueRequest{Key: key}, options...)
	if response != nil {
		return response.Value, err
	}
	return "", err
}

func setValue(keyValueYARPCClient examplepb.KeyValueYARPCClient, key string, value string, options ...yarpc.CallOption) error {
	ctx, cancel := context.WithTimeout(context.Background(), testtime.Second)
	defer cancel()
	_, err := keyValueYARPCClient.SetValue(ctx, &examplepb.SetValueRequest{Key: key, Value: value}, options...)
	return err
}

func getValueGRPC(keyValueGRPCClient examplepb.KeyValueClient, contextWrapper *grpcctx.ContextWrapper, key string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), testtime.Second)
	defer cancel()
	response, err := keyValueGRPCClient.GetValue(contextWrapper.Wrap(ctx), &examplepb.GetValueRequest{Key: key})
	if response != nil {
		return response.Value, err
	}
	return "", err
}

func setValueGRPC(keyValueGRPCClient examplepb.KeyValueClient, contextWrapper *grpcctx.ContextWrapper, key string, value string) error {
	ctx, cancel := context.WithTimeout(context.Background(), testtime.Second)
	defer cancel()
	_, err := keyValueGRPCClient.SetValue(contextWrapper.Wrap(ctx), &examplepb.SetValueRequest{Key: key, Value: value})
	return err
}

func fire(sinkYARPCClient examplepb.SinkYARPCClient, value string, options ...yarpc.CallOption) error {
	ctx, cancel := context.WithTimeout(context.Background(), testtime.Second)
	defer cancel()
	_, err := sinkYARPCClient.Fire(ctx, &examplepb.FireRequest{Value: value}, options...)
	return err
}

func echoOut(fooYARPCClient examplepb.FooYARPCClient, messages []string, options ...yarpc.CallOption) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), testtime.Second)
	defer cancel()
	client, err := fooYARPCClient.EchoOut(ctx, options...)
	if err != nil {
		return nil, err
	}
	for _, message := range messages {
		if err := client.Send(&examplepb.EchoOutRequest{Message: message}); err != nil {
			return nil, err
		}
	}
	response, err := client.CloseAndRecv()
	if err != nil {
		return nil, err
	}
	return response.AllMessages, nil
}

func echoIn(fooYARPCClient examplepb.FooYARPCClient, message string, numResponses int, options ...yarpc.CallOption) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), testtime.Second)
	defer cancel()
	client, err := fooYARPCClient.EchoIn(ctx, &examplepb.EchoInRequest{Message: message, NumResponses: int64(numResponses)}, options...)
	if err != nil {
		return nil, err
	}
	var messages []string
	for response, err := client.Recv(); err != io.EOF; response, err = client.Recv() {
		if err != nil {
			return nil, err
		}
		messages = append(messages, response.Message)
	}
	return messages, nil
}

func echoBoth(fooYARPCClient examplepb.FooYARPCClient, message string, numResponses int, count int, options ...yarpc.CallOption) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), testtime.Second)
	defer cancel()
	client, err := fooYARPCClient.EchoBoth(ctx, options...)
	if err != nil {
		return nil, err
	}

	var messages []string
	var recvErr error
	done := make(chan struct{})
	go func() {
		for response, err := client.Recv(); err != io.EOF; response, err = client.Recv() {
			if err != nil {
				recvErr = err
				break
			}
			messages = append(messages, response.Message)
		}
		close(done)
	}()

	for i := 0; i < count; i++ {
		if err := client.Send(&examplepb.EchoBothRequest{Message: message, NumResponses: int64(numResponses)}); err != nil {
			return nil, err
		}
	}
	if err := client.CloseSend(); err != nil {
		return nil, err
	}

	<-done
	return messages, recvErr
}

func echoOutGRPC(fooClient examplepb.FooClient, contextWrapper *grpcctx.ContextWrapper, messages []string) ([]string, error) {
	client, err := fooClient.EchoOut(contextWrapper.Wrap(context.Background()))
	if err != nil {
		return nil, err
	}
	for _, message := range messages {
		if err := client.Send(&examplepb.EchoOutRequest{Message: message}); err != nil {
			return nil, err
		}
	}
	response, err := client.CloseAndRecv()
	if err != nil {
		return nil, err
	}
	return response.AllMessages, nil
}

func echoInGRPC(fooClient examplepb.FooClient, contextWrapper *grpcctx.ContextWrapper, message string, numResponses int) ([]string, error) {
	client, err := fooClient.EchoIn(contextWrapper.Wrap(context.Background()), &examplepb.EchoInRequest{Message: message, NumResponses: int64(numResponses)})
	if err != nil {
		return nil, err
	}
	var messages []string
	for response, err := client.Recv(); err != io.EOF; response, err = client.Recv() {
		if err != nil {
			return nil, err
		}
		messages = append(messages, response.Message)
	}
	return messages, nil
}

func echoBothGRPC(fooClient examplepb.FooClient, contextWrapper *grpcctx.ContextWrapper, message string, numResponses int, count int) ([]string, error) {
	client, err := fooClient.EchoBoth(contextWrapper.Wrap(context.Background()))
	if err != nil {
		return nil, err
	}

	var messages []string
	var recvErr error
	done := make(chan struct{})
	go func() {
		for response, err := client.Recv(); err != io.EOF; response, err = client.Recv() {
			if err != nil {
				recvErr = err
				break
			}
			messages = append(messages, response.Message)
		}
		close(done)
	}()

	for i := 0; i < count; i++ {
		if err := client.Send(&examplepb.EchoBothRequest{Message: message, NumResponses: int64(numResponses)}); err != nil {
			return nil, err
		}
	}
	if err := client.CloseSend(); err != nil {
		return nil, err
	}

	<-done
	return messages, recvErr
}
