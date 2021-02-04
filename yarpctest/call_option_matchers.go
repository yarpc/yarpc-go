package yarpctest

// Copyright (c) 2021 Uber Technologies, Inc.
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

import (
	"context"
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	"go.uber.org/yarpc"
	"go.uber.org/yarpc/api/encoding"
	"go.uber.org/yarpc/api/transport"
)

var (
	_ gomock.Matcher = (*headerMatcher)(nil)
	_ gomock.Matcher = (*routingDelegateMatcher)(nil)
	_ gomock.Matcher = (*shardKeyMatcher)(nil)
	_ gomock.Matcher = (*routingKeyMatcher)(nil)
)

type headerMatcher struct {
	t             *testing.T
	expectedKey   string
	expectedValue string
}

// Returns a gomock.Matcher that matches a CallOption that sets a header.
func NewHeaderMatcher(t *testing.T, key string, value string) gomock.Matcher {
	return &headerMatcher{
		t:             t,
		expectedKey:   key,
		expectedValue: value,
	}
}

func (h headerMatcher) Matches(x interface{}) bool {
	option, ok := x.(yarpc.CallOption)
	if !ok {
		return false
	}

	req := writeOptionToRequest(h.t, option)

	if req.Headers.Len() != 1 {
		return false
	}

	for k, v := range req.Headers.Items() {
		return h.expectedKey == k && h.expectedValue == v
	}

	return false
}

func (h headerMatcher) String() string {
	return fmt.Sprintf("header %s:%s", h.expectedKey, h.expectedValue)
}

type routingDelegateMatcher struct {
	t             *testing.T
	expectedValue string
}

// Returns a gomock.Matcher that matches a CallOption that sets the routing delegate.
func NewRoutingDelegateMatcher(t *testing.T, value string) gomock.Matcher {
	return &routingDelegateMatcher{
		t:             t,
		expectedValue: value,
	}
}

func (r routingDelegateMatcher) Matches(x interface{}) bool {
	option, ok := x.(yarpc.CallOption)
	if !ok {
		return false
	}

	req := writeOptionToRequest(r.t, option)

	return r.expectedValue == req.RoutingDelegate
}

func (r routingDelegateMatcher) String() string {
	return fmt.Sprintf("routing delegate: %s", r.expectedValue)
}

type shardKeyMatcher struct {
	t             *testing.T
	expectedValue string
}

// Returns a gomock.Matcher that matches a CallOption that sets the shard key
func NewShardKeyMatcher(t *testing.T, value string) gomock.Matcher {
	return &shardKeyMatcher{
		t:             t,
		expectedValue: value,
	}
}

func (r shardKeyMatcher) Matches(x interface{}) bool {
	option, ok := x.(yarpc.CallOption)
	if !ok {
		return false
	}

	req := writeOptionToRequest(r.t, option)

	return r.expectedValue == req.ShardKey
}

func (r shardKeyMatcher) String() string {
	return fmt.Sprintf("shard key: %s", r.expectedValue)
}

type routingKeyMatcher struct {
	t             *testing.T
	expectedValue string
}

// Returns a gomock.Matcher that matches a CallOption that sets the routing key
func NewRoutingKeyMatcher(t *testing.T, value string) gomock.Matcher {
	return &routingKeyMatcher{
		t:             t,
		expectedValue: value,
	}
}

func (r routingKeyMatcher) Matches(x interface{}) bool {
	option, ok := x.(yarpc.CallOption)
	if !ok {
		return false
	}

	req := writeOptionToRequest(r.t, option)

	return r.expectedValue == req.RoutingKey
}

func (r routingKeyMatcher) String() string {
	return fmt.Sprintf("routing key: %s", r.expectedValue)
}

func writeOptionToRequest(t *testing.T, opt yarpc.CallOption) *transport.Request {
	outboundCall := encoding.NewOutboundCall(encoding.CallOption(opt))

	req := &transport.Request{}
	_, err := outboundCall.WriteToRequest(context.Background(), req)
	if err != nil {
		t.Fatalf("failed to write option request: %v", err)
	}

	return req
}
