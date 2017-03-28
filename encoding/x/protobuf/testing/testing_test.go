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
	keyValueServer := example.NewKeyValueServer()
	sinkServer := example.NewSinkServer(true)
	assert.NoError(
		t,
		example.WithClients(
			transportType,
			keyValueServer,
			sinkServer,
			func(keyValueClient examplepb.KeyValueClient, sinkClient examplepb.SinkClient) error {
				testIntegration(t, keyValueClient, sinkClient, keyValueServer, sinkServer)
				return nil
			},
		),
	)
}

func testIntegration(
	t *testing.T,
	keyValueClient examplepb.KeyValueClient,
	sinkClient examplepb.SinkClient,
	keyValueServer *example.KeyValueServer,
	sinkServer *example.SinkServer,
) {
	_, err := getValue(keyValueClient, "foo")
	assert.Error(t, err)
	assert.NotNil(t, protobuf.GetApplicationError(err))

	assert.NoError(t, setValue(keyValueClient, "foo", "bar"))
	value, err := getValue(keyValueClient, "foo")
	assert.NoError(t, err)
	assert.Equal(t, "bar", value)

	assert.NoError(t, setValue(keyValueClient, "foo", ""))
	_, err = getValue(keyValueClient, "foo")
	assert.Error(t, err)
	assert.NotNil(t, protobuf.GetApplicationError(err))

	assert.NoError(t, setValue(keyValueClient, "foo", "baz"))
	assert.NoError(t, setValue(keyValueClient, "baz", "bat"))
	value, err = getValue(keyValueClient, "foo")
	assert.NoError(t, err)
	assert.Equal(t, "baz", value)
	value, err = getValue(keyValueClient, "baz")
	assert.NoError(t, err)
	assert.Equal(t, "bat", value)

	assert.NoError(t, fire(sinkClient, "foo"))
	assert.NoError(t, sinkServer.WaitFireDone())
	assert.NoError(t, fire(sinkClient, "bar"))
	assert.NoError(t, sinkServer.WaitFireDone())
	assert.Equal(t, []string{"foo", "bar"}, sinkServer.Values())
}

func getValue(keyValueClient examplepb.KeyValueClient, key string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	response, err := keyValueClient.GetValue(ctx, &examplepb.GetValueRequest{key})
	if err != nil {
		return "", err
	}
	return response.Value, nil
}

func setValue(keyValueClient examplepb.KeyValueClient, key string, value string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	_, err := keyValueClient.SetValue(ctx, &examplepb.SetValueRequest{key, value})
	return err
}

func fire(sinkClient examplepb.SinkClient, value string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	_, err := sinkClient.Fire(ctx, &examplepb.FireRequest{value})
	return err
}
