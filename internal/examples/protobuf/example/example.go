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
	"time"

	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/internal/examples/protobuf/examplepb"
	"go.uber.org/yarpc/internal/testutils"
)

const (
	// FireDoneTimeout is how long fireDone will wait for both sending and receiving.
	FireDoneTimeout = 3 * time.Second
)

var (
	errRequestNil      = errors.New("request nil")
	errRequestKeyNil   = errors.New("request key nil")
	errRequestValueNil = errors.New("request value nil")
)

// Clients holds all clients.
type Clients struct {
	KeyValueYarpcClient examplepb.KeyValueYarpcClient
	SinkYarpcClient     examplepb.SinkYarpcClient
	KeyValueGRPCClient  examplepb.KeyValueClient
	SinkGRPCClient      examplepb.SinkClient
}

// WithClients calls f on the Clients.
func WithClients(
	transportType testutils.TransportType,
	keyValueYarpcServer examplepb.KeyValueYarpcServer,
	sinkYarpcServer examplepb.SinkYarpcServer,
	f func(*Clients) error,
) error {
	var procedures []transport.Procedure
	if keyValueYarpcServer != nil {
		procedures = append(procedures, examplepb.BuildKeyValueYarpcProcedures(keyValueYarpcServer)...)
	}
	if sinkYarpcServer != nil {
		procedures = append(procedures, examplepb.BuildSinkYarpcProcedures(sinkYarpcServer)...)
	}
	return testutils.WithClientInfo(
		"example",
		procedures,
		transportType,
		func(clientInfo *testutils.ClientInfo) error {
			return f(
				&Clients{
					examplepb.NewKeyValueYarpcClient(clientInfo.ClientConfig),
					examplepb.NewSinkYarpcClient(clientInfo.ClientConfig),
					examplepb.NewKeyValueClient(clientInfo.GRPCClientConn),
					examplepb.NewSinkClient(clientInfo.GRPCClientConn),
				},
			)
		},
	)
}

// KeyValueYarpcServer implements examplepb.KeyValueYarpcServer.
type KeyValueYarpcServer struct {
	sync.RWMutex
	items map[string]string
}

// NewKeyValueYarpcServer returns a new KeyValueYarpcServer.
func NewKeyValueYarpcServer() *KeyValueYarpcServer {
	return &KeyValueYarpcServer{sync.RWMutex{}, make(map[string]string)}
}

// GetValue implements GetValue.
func (k *KeyValueYarpcServer) GetValue(ctx context.Context, request *examplepb.GetValueRequest) (*examplepb.GetValueResponse, error) {
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
func (k *KeyValueYarpcServer) SetValue(ctx context.Context, request *examplepb.SetValueRequest) (*examplepb.SetValueResponse, error) {
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

// SinkYarpcServer implements examplepb.SinkYarpcServer.
type SinkYarpcServer struct {
	sync.RWMutex
	values   []string
	fireDone chan struct{}
}

// NewSinkYarpcServer returns a new SinkYarpcServer.
func NewSinkYarpcServer(withFireDone bool) *SinkYarpcServer {
	var fireDone chan struct{}
	if withFireDone {
		fireDone = make(chan struct{})
	}
	return &SinkYarpcServer{sync.RWMutex{}, make([]string, 0), fireDone}
}

// Fire implements Fire.
func (s *SinkYarpcServer) Fire(ctx context.Context, request *examplepb.FireRequest) error {
	if request == nil {
		return errRequestNil
	}
	if request.Value == "" {
		return errRequestValueNil
	}
	s.Lock()
	s.values = append(s.values, request.Value)
	s.Unlock()
	if s.fireDone == nil {
		return nil
	}
	select {
	case s.fireDone <- struct{}{}:
	case <-time.After(FireDoneTimeout):
		return fmt.Errorf("fire done not handled after %v", FireDoneTimeout)
	}
	return nil
}

// Values returns a copy of the values that have been fired.
func (s *SinkYarpcServer) Values() []string {
	s.RLock()
	values := make([]string, len(s.values))
	copy(values, s.values)
	s.RUnlock()
	return values
}

// WaitFireDone blocks until a fire is done, if withFireDone is set.
//
// If will timeout after FireDoneTimeout and return error.
func (s *SinkYarpcServer) WaitFireDone() error {
	if s.fireDone == nil {
		return nil
	}
	select {
	case <-s.fireDone:
	case <-time.After(FireDoneTimeout):
		return fmt.Errorf("fire not done after %v", FireDoneTimeout)
	}
	return nil
}
