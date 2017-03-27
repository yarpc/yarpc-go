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
	errRequestNil      = errors.New("request nil")
	errRequestKeyNil   = errors.New("request key nil")
	errRequestValueNil = errors.New("request value nil")
)

// WithClients calls f on the Clients.
func WithClients(
	transportType testutils.TransportType,
	keyValueServer examplepb.KeyValueServer,
	sinkServer examplepb.SinkServer,
	f func(examplepb.KeyValueClient, examplepb.SinkClient) error,
) error {
	var procedures []transport.Procedure
	if keyValueServer != nil {
		procedures = append(procedures, examplepb.BuildKeyValueProcedures(keyValueServer)...)
	}
	if sinkServer != nil {
		procedures = append(procedures, examplepb.BuildSinkProcedures(sinkServer)...)
	}
	return testutils.WithClientConfig(
		"example",
		procedures,
		transportType,
		func(clientConfig transport.ClientConfig) error {
			return f(
				examplepb.NewKeyValueClient(clientConfig),
				examplepb.NewSinkClient(clientConfig),
			)
		},
	)
}

// KeyValueServer implements examplepb.KeyValueServer.
type KeyValueServer struct {
	sync.RWMutex
	items map[string]string
}

// NewKeyValueServer returns a new KeyValueServer.
func NewKeyValueServer() *KeyValueServer {
	return &KeyValueServer{sync.RWMutex{}, make(map[string]string)}
}

// GetValue implements GetValue.
func (k *KeyValueServer) GetValue(ctx context.Context, request *examplepb.GetValueRequest) (*examplepb.GetValueResponse, error) {
	if request == nil {
		return nil, errRequestNil
	}
	if request.Key == "" {
		return nil, errRequestKeyNil
	}
	k.RLock()
	if value, ok := k.items[request.Key]; ok {
		k.RUnlock()
		return &examplepb.GetValueResponse{value}, nil
	}
	k.RUnlock()
	return nil, fmt.Errorf("key not set: %s", request.Key)
}

// SetValue implements SetValue.
func (k *KeyValueServer) SetValue(ctx context.Context, request *examplepb.SetValueRequest) (*examplepb.SetValueResponse, error) {
	if request == nil {
		return nil, errRequestNil
	}
	if request.Key == "" {
		return nil, errRequestKeyNil
	}
	k.Lock()
	if request.Value == "" {
		delete(k.items, request.Key)
	} else {
		k.items[request.Key] = request.Value
	}
	k.Unlock()
	return nil, nil
}

// SinkServer implements examplepb.SinkServer.
type SinkServer struct {
	sync.RWMutex
	values []string
}

// NewSinkServer returns a new SinkServer.
func NewSinkServer() *SinkServer {
	return &SinkServer{sync.RWMutex{}, make([]string, 0)}
}

// Fire implements Fire.
func (s *SinkServer) Fire(ctx context.Context, request *examplepb.FireRequest) error {
	if request == nil {
		return errRequestNil
	}
	if request.Value == "" {
		return errRequestValueNil
	}
	s.Lock()
	s.values = append(s.values, request.Value)
	s.Unlock()
	return nil
}

// Values returns a copy of the values that have been fired.
func (s *SinkServer) Values() []string {
	s.RLock()
	values := make([]string, len(s.values))
	copy(values, s.values)
	s.RUnlock()
	return values
}
