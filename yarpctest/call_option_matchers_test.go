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
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc"
)

func TestHeaderMatcher(t *testing.T) {
	opt1 := yarpc.WithHeader("a-header", "a-value")
	opt2 := yarpc.WithHeader("a-header", "another-value")

	matcher := NewHeaderMatcher(t, "a-header", "a-value")

	require.True(t, matcher.Matches(opt1))
	require.False(t, matcher.Matches(opt2))
}

func TestRoutingDelegateMatcher(t *testing.T) {
	opt1 := yarpc.WithRoutingDelegate("a-routing-delegate")
	opt2 := yarpc.WithRoutingDelegate("another-routing-delegate")

	matcher := NewRoutingDelegateMatcher(t, "a-routing-delegate")

	require.True(t, matcher.Matches(opt1))
	require.False(t, matcher.Matches(opt2))
}

func TestRoutingKeyMatcher(t *testing.T) {
	opt1 := yarpc.WithRoutingKey("a-routing-key")
	opt2 := yarpc.WithRoutingKey("another-routing-key")

	matcher := NewRoutingKeyMatcher(t, "a-routing-key")

	require.True(t, matcher.Matches(opt1))
	require.False(t, matcher.Matches(opt2))
}

func TestShardKeyMatcher(t *testing.T) {
	opt1 := yarpc.WithShardKey("a-shard-key")
	opt2 := yarpc.WithShardKey("another-shard-key")

	matcher := NewShardKeyMatcher(t, "a-shard-key")

	require.True(t, matcher.Matches(opt1))
	require.False(t, matcher.Matches(opt2))
}
