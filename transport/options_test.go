// Copyright (c) 2016 Uber Technologies, Inc.
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

package transport

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOptionsIsImmutable(t *testing.T) {
	type foo struct{}
	type bar struct{}

	var o Options
	withFoo := o.With(foo{}, "foo")
	withBar := o.With(bar{}, "bar")
	withFooBar := withFoo.With(bar{}, "foobar")

	_, ok := o.Get(foo{})
	assert.False(t, ok, "did not expect to find foo{} in o")

	_, ok = o.Get(bar{})
	assert.False(t, ok, "did not expect to find bar{} in o")

	_, ok = withFoo.Get(bar{})
	assert.False(t, ok, "did not expect to find bar{} in withFoo")

	_, ok = withBar.Get(foo{})
	assert.False(t, ok, "did not expect to find foo{} in withBar")

	if v, ok := withFoo.Get(foo{}); assert.True(t, ok, "expected foo{} in withFoo") {
		assert.Equal(t, "foo", v, "withFoo[foo{}] did not match")
	}

	if v, ok := withBar.Get(bar{}); assert.True(t, ok, "expected bar{} in withBar") {
		assert.Equal(t, "bar", v, "withBar[bar{}] did not match")
	}

	if v, ok := withFooBar.Get(foo{}); assert.True(t, ok, "expected foo{} in withFooBar") {
		assert.Equal(t, "foo", v, "withFooBar[foo{}] did not match")
	}

	if v, ok := withFooBar.Get(bar{}); assert.True(t, ok, "expected bar{} in withFooBar") {
		assert.Equal(t, "foobar", v, "withFooBar[bar{}] did not match")
	}
}
