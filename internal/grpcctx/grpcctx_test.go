// Copyright (c) 2026 Uber Technologies, Inc.
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

package grpcctx

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/metadata"
)

func TestContextWrapper(t *testing.T) {
	t.Run("NewContextWrapper", func(t *testing.T) {
		testContextWrapper(t, NewContextWrapper)
	})
	t.Run("&ContextWrapper{}", func(t *testing.T) {
		testContextWrapper(t, func() *ContextWrapper { return &ContextWrapper{} })
	})
}

func testContextWrapper(t *testing.T, create func() *ContextWrapper) {
	c1 := create().WithCaller("test-caller")
	checkMetadata(t, c1, metadata.MD{
		"rpc-caller": []string{"test-caller"},
	})
	c2 := c1.WithService("test-service")
	checkMetadata(t, c1, metadata.MD{
		"rpc-caller": []string{"test-caller"},
	})
	checkMetadata(t, c2, metadata.MD{
		"rpc-caller":  []string{"test-caller"},
		"rpc-service": []string{"test-service"},
	})
	c2 = c2.WithShardKey("test-shard-key").
		WithRoutingKey("test-routing-key").
		WithRoutingDelegate("test-routing-delegate").
		WithEncoding("test-encoding")
	checkMetadata(t, c2, metadata.MD{
		"rpc-caller":           []string{"test-caller"},
		"rpc-service":          []string{"test-service"},
		"rpc-shard-key":        []string{"test-shard-key"},
		"rpc-routing-key":      []string{"test-routing-key"},
		"rpc-routing-delegate": []string{"test-routing-delegate"},
		"rpc-encoding":         []string{"test-encoding"},
	})
}

func checkMetadata(t *testing.T, contextWrapper *ContextWrapper, expectedMD metadata.MD) {
	md, ok := metadata.FromOutgoingContext(contextWrapper.Wrap(context.Background()))
	assert.True(t, ok)
	assert.Equal(t, expectedMD, md)
}
