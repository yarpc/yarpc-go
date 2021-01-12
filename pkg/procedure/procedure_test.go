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

package procedure

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProcedureSplitEmpty(t *testing.T) {
	s, m := FromName("")
	assert.Equal(t, "", s)
	assert.Equal(t, "", m)
}

func TestProcedureNameAndSplit(t *testing.T) {
	tests := []struct {
		Procedure string
		Service   string
		Method    string
	}{
		{"foo::bar", "foo", "bar"},
		{"::bar", "", "bar"},
		{"foo::", "foo", ""},
		{"::", "", ""},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.Procedure, ToName(tt.Service, tt.Method))
		s, m := FromName(tt.Procedure)
		assert.Equal(t, tt.Service, s)
		assert.Equal(t, tt.Method, m)
	}
}
