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

package keyvalue

import (
	"context"
	"fmt"
	"sync"

	commonpb "go.uber.org/yarpc/v2/yarpcprotobuf/protoc-gen-yarpc-go/internal/tests/gen/proto/src/common"
	keyvaluepb "go.uber.org/yarpc/v2/yarpcprotobuf/protoc-gen-yarpc-go/internal/tests/gen/proto/src/keyvalue"
)

type kvServer struct {
	rw sync.RWMutex

	store map[string]string
}

// NewServer returns a new keyvaluepb.StoreServer.
func NewServer() keyvaluepb.StoreYARPCServer {
	return &kvServer{
		store: make(map[string]string),
	}
}

func (s *kvServer) Get(ctx context.Context, req *commonpb.GetRequest) (*commonpb.GetResponse, error) {
	s.rw.RLock()
	defer s.rw.RUnlock()

	val, ok := s.store[req.GetKey()]
	if !ok {
		return nil, fmt.Errorf("failed to find value for key: %q", req.Key)
	}
	return &commonpb.GetResponse{Value: val}, nil
}

func (s *kvServer) Set(ctx context.Context, req *commonpb.SetRequest) (*commonpb.SetResponse, error) {
	s.rw.Lock()
	defer s.rw.Unlock()

	s.store[req.GetKey()] = req.GetValue()
	return &commonpb.SetResponse{}, nil
}
