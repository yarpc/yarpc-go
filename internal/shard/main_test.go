// Copyright (c) 2020 Uber Technologies, Inc.
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

package main

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDo(t *testing.T) {
	tests := []struct {
		Args       []string
		WantOutput string
		WantError  bool
	}{
		{
			WantError: true,
		},
		{
			Args:      []string{},
			WantError: true,
		},
		{
			Args:      []string{"1"},
			WantError: true,
		},
		{
			Args:      []string{"1", "x"},
			WantError: true,
		},
		{
			Args:      []string{"x", "1"},
			WantError: true,
		},
		{
			Args: []string{"1", "3"},
		},
		{
			Args:       []string{"1", "3", "a"},
			WantOutput: "a",
		},
		{
			Args:       []string{"1", "3", "a", "b"},
			WantOutput: "a",
		},
		{
			Args:       []string{"1", "3", "a", "b", "c"},
			WantOutput: "a",
		},
		{
			Args:       []string{"1", "3", "a", "b", "c", "d"},
			WantOutput: "a d",
		},
	}
	for _, tt := range tests {
		buffer := bytes.NewBuffer(nil)
		err := do(tt.Args, buffer)
		if tt.WantError {
			assert.Error(t, err)
		} else {
			assert.NoError(t, err)
		}
		assert.Equal(t, tt.WantOutput, buffer.String())
	}
}
