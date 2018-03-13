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

package example

import (
	"context"
	"io"
	"sync"
	"time"

	"go.uber.org/yarpc"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/internal/examples/protobuf/examplepb"
	"go.uber.org/yarpc/yarpcerrors"
)

const (
	// FireDoneTimeout is how long fireDone will wait for both sending and receiving.
	FireDoneTimeout = 3 * time.Second
)

var (
	errRequestNil             = yarpcerrors.Newf(yarpcerrors.CodeInvalidArgument, "request nil")
	errRequestKeyNil          = yarpcerrors.Newf(yarpcerrors.CodeInvalidArgument, "request key nil")
	errRequestValueNil        = yarpcerrors.Newf(yarpcerrors.CodeInvalidArgument, "request value nil")
	errRequestMessageNil      = yarpcerrors.Newf(yarpcerrors.CodeInvalidArgument, "request message nil")
	errRequestNumResponsesNil = yarpcerrors.Newf(yarpcerrors.CodeInvalidArgument, "request num responses nil")
)

// KeyValueYARPCServer implements examplepb.KeyValueYARPCServer.
type KeyValueYARPCServer struct {
	sync.RWMutex
	items map[string]string
	// if next error is set it will be returned along with an empty response
	// from any call to KeyValueYarpcServer, and then set to nil
	nextError error
}

// NewKeyValueYARPCServer returns a new KeyValueYARPCServer.
func NewKeyValueYARPCServer() *KeyValueYARPCServer {
	return &KeyValueYARPCServer{sync.RWMutex{}, make(map[string]string), nil}
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
		var nextError error
		k.Lock()
		if k.nextError != nil {
			nextError = k.nextError
			k.nextError = nil
		}
		k.Unlock()
		return &examplepb.GetValueResponse{Value: value}, nextError
	}
	k.RUnlock()
	return nil, yarpcerrors.Newf(yarpcerrors.CodeNotFound, request.Key)
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
	var nextError error
	if k.nextError != nil {
		nextError = k.nextError
		k.nextError = nil
	}
	k.Unlock()
	return nil, nextError
}

// SetNextError sets the error to return on the next call to KeyValueYARPCServer.
func (k *KeyValueYARPCServer) SetNextError(err error) {
	k.Lock()
	defer k.Unlock()
	k.nextError = err
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
		return yarpcerrors.Newf(yarpcerrors.CodeDeadlineExceeded, "fire done not handled after %v", FireDoneTimeout)
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
		return yarpcerrors.Newf(yarpcerrors.CodeDeadlineExceeded, "fire not done after %v", FireDoneTimeout)
	}
	return nil
}

// FooYARPCServer implements examplepb.FooYARPCServer.
type FooYARPCServer struct {
	expectedHeaders transport.Headers
}

// NewFooYARPCServer returns a new FooYARPCServer.
func NewFooYARPCServer(expectedHeaders transport.Headers) *FooYARPCServer {
	return &FooYARPCServer{
		expectedHeaders: expectedHeaders,
	}
}

// EchoOut reads from a stream and echos all requests in the response.
func (f *FooYARPCServer) EchoOut(server examplepb.FooServiceEchoOutYARPCServer) (*examplepb.EchoOutResponse, error) {
	var allMessages []string
	call := yarpc.CallFromContext(server.Context())
	for k, v := range f.expectedHeaders.Items() {
		if call.Header(k) != v {
			return nil, yarpcerrors.InvalidArgumentErrorf("did not receive proper headers, missing %q:%q", k, v)
		}
	}
	for request, err := server.Recv(); err != io.EOF; request, err = server.Recv() {
		if err != nil {
			return nil, err
		}
		if request == nil {
			return nil, errRequestNil
		}
		if request.Message == "" {
			return nil, errRequestMessageNil
		}
		allMessages = append(allMessages, request.Message)
	}
	return &examplepb.EchoOutResponse{
		AllMessages: allMessages,
	}, nil
}

// EchoIn echos a series of requests back on a stream.
func (f *FooYARPCServer) EchoIn(request *examplepb.EchoInRequest, server examplepb.FooServiceEchoInYARPCServer) error {
	if request == nil {
		return errRequestNil
	}
	if request.Message == "" {
		return errRequestMessageNil
	}
	if request.NumResponses == 0 {
		return errRequestNumResponsesNil
	}
	call := yarpc.CallFromContext(server.Context())
	for k, v := range f.expectedHeaders.Items() {
		if call.Header(k) != v {
			return yarpcerrors.InvalidArgumentErrorf("did not receive proper headers, missing %q:%q", k, v)
		}
	}
	for i := 0; i < int(request.NumResponses); i++ {
		if err := server.Send(&examplepb.EchoInResponse{Message: request.Message}); err != nil {
			return err
		}
	}
	return nil
}

// EchoBoth immediately echos a request back to the client.
func (f *FooYARPCServer) EchoBoth(server examplepb.FooServiceEchoBothYARPCServer) error {
	call := yarpc.CallFromContext(server.Context())
	for k, v := range f.expectedHeaders.Items() {
		if call.Header(k) != v {
			return yarpcerrors.InvalidArgumentErrorf("did not receive proper headers, missing %q:%q", k, v)
		}
	}
	for request, err := server.Recv(); err != io.EOF; request, err = server.Recv() {
		if err != nil {
			return err
		}
		if request == nil {
			return errRequestNil
		}
		if request.Message == "" {
			return errRequestMessageNil
		}
		if request.NumResponses == 0 {
			return errRequestNumResponsesNil
		}
		for i := 0; i < int(request.NumResponses); i++ {
			if err := server.Send(&examplepb.EchoBothResponse{Message: request.Message}); err != nil {
				return err
			}
		}
	}
	return nil
}
