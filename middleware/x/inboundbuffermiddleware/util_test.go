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

package inboundbuffermiddleware

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestJot(t *testing.T) {
	assert.Equal(t, []int{0, 1, 2}, jot(3))
}

func TestSwap(t *testing.T) {
	nums := []int{3, 2, 1, 0}
	coNums := []int{3, 2, 1, 0}

	swap(nums, coNums, 0, 0)
	assert.Equal(t, []int{3, 2, 1, 0}, nums)
	assert.Equal(t, []int{3, 2, 1, 0}, coNums)

	swap(nums, coNums, 0, 1)
	assert.Equal(t, []int{2, 3, 1, 0}, nums)
	assert.Equal(t, []int{3, 2, 0, 1}, coNums)

	swap(nums, coNums, 1, 2)
	assert.Equal(t, []int{2, 1, 3, 0}, nums)
	assert.Equal(t, []int{3, 1, 0, 2}, coNums)

	swap(nums, coNums, 2, 3)
	assert.Equal(t, []int{2, 1, 0, 3}, nums)
	assert.Equal(t, []int{2, 1, 0, 3}, coNums)

	swap(nums, coNums, 0, 3)
	assert.Equal(t, []int{3, 1, 0, 2}, nums)
	assert.Equal(t, []int{2, 1, 3, 0}, coNums)
}
