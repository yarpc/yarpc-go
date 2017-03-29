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
	"time"

	"go.uber.org/yarpc/encoding/x/protobuf"
	"go.uber.org/yarpc/internal/examples/protobuf/example"
	"go.uber.org/yarpc/internal/examples/protobuf/examplepb"
	"go.uber.org/yarpc/internal/testutils"

	"github.com/stretchr/testify/assert"
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
		example.WithClients(
			transportType,
			keyValueYarpcServer,
			sinkYarpcServer,
			func(keyValueYarpcClient examplepb.KeyValueYarpcClient, sinkYarpcClient examplepb.SinkYarpcClient) error {
				testIntegration(t, keyValueYarpcClient, sinkYarpcClient, keyValueYarpcServer, sinkYarpcServer)
				return nil
			},
		),
	)
}

func testIntegration(
	t *testing.T,
	keyValueYarpcClient examplepb.KeyValueYarpcClient,
	sinkYarpcClient examplepb.SinkYarpcClient,
	keyValueYarpcServer *example.KeyValueYarpcServer,
	sinkYarpcServer *example.SinkYarpcServer,
) {
	_, err := getValue(keyValueYarpcClient, "foo")
	assert.Error(t, err)
	assert.NotNil(t, protobuf.GetApplicationError(err))

	assert.NoError(t, setValue(keyValueYarpcClient, "foo", "bar"))
	value, err := getValue(keyValueYarpcClient, "foo")
	assert.NoError(t, err)
	assert.Equal(t, "bar", value)

	assert.NoError(t, setValue(keyValueYarpcClient, "foo", ""))
	_, err = getValue(keyValueYarpcClient, "foo")
	assert.Error(t, err)
	assert.NotNil(t, protobuf.GetApplicationError(err))

	assert.NoError(t, setValue(keyValueYarpcClient, "foo", "baz"))
	assert.NoError(t, setValue(keyValueYarpcClient, "baz", "bat"))
	value, err = getValue(keyValueYarpcClient, "foo")
	assert.NoError(t, err)
	assert.Equal(t, "baz", value)
	value, err = getValue(keyValueYarpcClient, "baz")
	assert.NoError(t, err)
	assert.Equal(t, "bat", value)

	assert.NoError(t, fire(sinkYarpcClient, "foo"))
	assert.NoError(t, sinkYarpcServer.WaitFireDone())
	assert.NoError(t, fire(sinkYarpcClient, "bar"))
	assert.NoError(t, sinkYarpcServer.WaitFireDone())
	assert.Equal(t, []string{"foo", "bar"}, sinkYarpcServer.Values())
}

func getValue(keyValueYarpcClient examplepb.KeyValueYarpcClient, key string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	response, err := keyValueYarpcClient.GetValue(ctx, &examplepb.GetValueRequest{key})
	if err != nil {
		return "", err
	}
	return response.Value, nil
}

func setValue(keyValueYarpcClient examplepb.KeyValueYarpcClient, key string, value string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	_, err := keyValueYarpcClient.SetValue(ctx, &examplepb.SetValueRequest{key, value})
	return err
}

func fire(sinkYarpcClient examplepb.SinkYarpcClient, value string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	_, err := sinkYarpcClient.Fire(ctx, &examplepb.FireRequest{value})
	return err
}
