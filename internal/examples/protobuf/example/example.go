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

package example

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/internal/examples/protobuf/examplepb"
	"go.uber.org/yarpc/internal/testutils"
)

var (
	errRequestNil    = errors.New("request nil")
	errRequestKeyNil = errors.New("request key nil")
)

// WithKeyValueClient calls f on a KeyValueClient.
func WithKeyValueClient(transportType testutils.TransportType, f func(examplepb.KeyValueClient) error) error {
	return testutils.WithClientConfig(
		"example",
		examplepb.BuildKeyValueProcedures(newKeyValueServer()),
		transportType,
		func(clientConfig transport.ClientConfig) error {
			return f(examplepb.NewKeyValueClient(clientConfig))
		},
	)
}

type keyValueServer struct {
	sync.RWMutex
	items map[string]string
}

func newKeyValueServer() *keyValueServer {
	return &keyValueServer{sync.RWMutex{}, make(map[string]string)}
}

func (a *keyValueServer) GetValue(ctx context.Context, request *examplepb.GetValueRequest) (*examplepb.GetValueResponse, error) {
	if request == nil {
		return nil, errRequestNil
	}
	if request.Key == "" {
		return nil, errRequestKeyNil
	}
	a.RLock()
	if value, ok := a.items[request.Key]; ok {
		a.RUnlock()
		return &examplepb.GetValueResponse{value}, nil
	}
	a.RUnlock()
	return nil, fmt.Errorf("key not set: %s", request.Key)
}

func (a *keyValueServer) SetValue(ctx context.Context, request *examplepb.SetValueRequest) (*examplepb.SetValueResponse, error) {
	if request == nil {
		return nil, errRequestNil
	}
	if request.Key == "" {
		return nil, errRequestKeyNil
	}
	a.Lock()
	if request.Value == "" {
		delete(a.items, request.Key)
	} else {
		a.items[request.Key] = request.Value
	}
	a.Unlock()
	return nil, nil
}
