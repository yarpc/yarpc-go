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

package errors

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestErrorString(t *testing.T) {
	assert.Equal(t, "type: CANCELLED", Cancelled().Error())
	assert.Equal(t, "type: CANCELLED", Cancelled("foo", "").Error())
	assert.Equal(t, "type: CANCELLED details: foo:missing", Cancelled("foo").Error())
	assert.Equal(t, "type: CANCELLED details: foo:bar", Cancelled("foo", "bar").Error())
	assert.Equal(t, "type: CANCELLED details: foo:bar baz:missing", Cancelled("foo", "bar", "baz").Error())
	assert.Equal(t, "type: CANCELLED details: foo:bar baz:bat", Cancelled("foo", "bar", "baz", "bat").Error())
	assert.Equal(t, "type: CANCELLED details: foo:bar baz:bat", Cancelled("foo", "bar", "baz", "bat", "ban", "").Error())
	assert.Equal(t, "type: CANCELLED details: foo:bar baz:bat too:tee", WithKeyValues(Cancelled("foo", "bar", "baz", "bat", "ban", ""), "too", "tee").Error())
	assert.Equal(t, "type: APPLICATION name: hello", Application("hello").Error())
	assert.Equal(t, "type: APPLICATION name: hello", Application("hello", "foo", "").Error())
	assert.Equal(t, "type: APPLICATION name: hello details: foo:missing", Application("hello", "foo").Error())
	assert.Equal(t, "type: APPLICATION name: hello details: foo:bar", Application("hello", "foo", "bar").Error())
	assert.Equal(t, "type: APPLICATION name: hello details: foo:bar baz:missing", Application("hello", "foo", "bar", "baz").Error())
	assert.Equal(t, "type: APPLICATION name: hello details: foo:bar baz:bat", Application("hello", "foo", "bar", "baz", "bat").Error())
	assert.Equal(t, "type: APPLICATION name: hello details: foo:bar baz:bat", Application("hello", "foo", "bar", "baz", "bat", "ban", "").Error())
}
