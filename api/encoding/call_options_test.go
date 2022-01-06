// Copyright (c) 2022 Uber Technologies, Inc.
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

package encoding

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

// Ensuring these are equal via `reflect.DeepEqual` ensures that gomock by default can assert equality on CallOptions
// in tests without custom matchers
func TestCallOptionsReflectEquals(t *testing.T) {
	opt1 := WithHeader("a-header", "a-value")
	opt2 := WithHeader("a-header", "a-value")

	require.True(t, reflect.DeepEqual(opt1, opt2))
}

func TestManyCallOptionsReflectEquals(t *testing.T) {
	opts1 := []CallOption{WithHeader("a-header", "a-value"), WithRoutingKey("a-routing-key")}
	opts2 := []CallOption{WithHeader("a-header", "a-value"), WithRoutingKey("a-routing-key")}
	opts3 := []CallOption{WithShardKey("a-shard-key")}

	require.True(t, reflect.DeepEqual(opts1, opts2))
	require.False(t, reflect.DeepEqual(opts1, opts3))
	require.False(t, reflect.DeepEqual(opts2, opts3))
}
