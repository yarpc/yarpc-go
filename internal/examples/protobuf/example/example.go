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

	"go.uber.org/yarpc/internal/examples/protobuf/examplepb"
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

// KeyValueYARPCServer implements examplepb.KeyValueYARPCServer.
type KeyValueYARPCServer struct {
	sync.RWMutex
	items map[string]string
}

// NewKeyValueYARPCServer returns a new KeyValueYARPCServer.
func NewKeyValueYARPCServer() *KeyValueYARPCServer {
	return &KeyValueYARPCServer{sync.RWMutex{}, make(map[string]string)}
}

// GetValue implements GetValue.
func (k *KeyValueYARPCServer) GetValue(ctx context.Context, request *examplepb.GetValueRequest) (*examplepb.GetValueResponse, error) {
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
func (k *KeyValueYARPCServer) SetValue(ctx context.Context, request *examplepb.SetValueRequest) (*examplepb.SetValueResponse, error) {
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

// SinkYARPCServer implements examplepb.SinkYARPCServer.
type SinkYARPCServer struct {
	sync.RWMutex
	values   []string
	fireDone chan struct{}
}

// NewSinkYARPCServer returns a new SinkYARPCServer.
func NewSinkYARPCServer(withFireDone bool) *SinkYARPCServer {
	var fireDone chan struct{}
	if withFireDone {
		fireDone = make(chan struct{})
	}
	return &SinkYARPCServer{sync.RWMutex{}, make([]string, 0), fireDone}
}

// Fire implements Fire.
func (s *SinkYARPCServer) Fire(ctx context.Context, request *examplepb.FireRequest) error {
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
func (s *SinkYARPCServer) Values() []string {
	s.RLock()
	values := make([]string, len(s.values))
	copy(values, s.values)
	s.RUnlock()
	return values
}

// WaitFireDone blocks until a fire is done, if withFireDone is set.
//
// If will timeout after FireDoneTimeout and return error.
func (s *SinkYARPCServer) WaitFireDone() error {
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
