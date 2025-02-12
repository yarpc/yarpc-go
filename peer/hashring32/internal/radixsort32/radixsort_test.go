// Copyright (c) 2025 Uber Technologies, Inc.
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

package radixsort32

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestRadixSort tests radix sort correctness
func TestRadixSort(t *testing.T) {
	sorter := makeSorter(16, 100, 15000)
	// short break
	testRadixSortReversedSlice(t, sorter, 10)
	// radix sort with buffers in pool
	testRadixSortReversedSlice(t, sorter, 300)
	// radix sort with dynamically allocated buffers
	testRadixSortReversedSlice(t, sorter, 20000)
}

// TestOptions tests options and fallbacks
func TestOptions(t *testing.T) {
	sorter := makeSorter(3, 500000, 100)
	assert.Equal(t, uint(8), sorter.radix)
	assert.Equal(t, 100, sorter.maxLength)
	assert.Equal(t, 100, sorter.minLength)

	sorter2 := makeSorter(3, -50, 100)
	assert.Equal(t, 0, sorter2.minLength)
	assert.Equal(t, 100, sorter2.maxLength)
}

func testRadixSortReversedSlice(t *testing.T, sorter *RadixSorter32, size int) {
	slice := makeRevertedSlice(size)
	sorter.Sort(slice)
	assert.EqualValues(t, makeSortedSlice(size), slice)
}

func makeSorter(radix, minLen, maxLen int) *RadixSorter32 {
	sorter := New(
		Radix(radix),
		MinLen(minLen),
		MaxLen(maxLen))
	return sorter
}

func makeRevertedSlice(size int) []uint32 {
	a := make([]uint32, size)
	for i := size - 1; i >= 0; i-- {
		a[i] = uint32(len(a) - 1 - i)
	}
	return a
}

func makeSortedSlice(size int) []uint32 {
	a := make([]uint32, size)
	for i := 0; i < len(a); i++ {
		a[i] = uint32(i)
	}
	return a
}

// Benchmarks

func BenchmarkSortSize150KCompare(b *testing.B) {
	benchmarkCompareSort(b, 150000)
}

func BenchmarkSortSize150KRadix4(b *testing.B) {
	benchmarkRadixSort(b, 4, 150000)
}

func BenchmarkSortSize150KRadix8(b *testing.B) {
	benchmarkRadixSort(b, 8, 150000)
}

func BenchmarkSortSize150KRadix16(b *testing.B) {
	benchmarkRadixSort(b, 16, 150000)
}

func BenchmarkSortSize15KCompare(b *testing.B) {
	benchmarkCompareSort(b, 15000)
}

func BenchmarkSortSize15KRadix4(b *testing.B) {
	benchmarkRadixSort(b, 4, 15000)
}

func BenchmarkSortSize15KRadix8(b *testing.B) {
	benchmarkRadixSort(b, 8, 15000)
}

func BenchmarkSortSize15KRadix16(b *testing.B) {
	benchmarkRadixSort(b, 16, 15000)
}

func BenchmarkSortSize1500Compare(b *testing.B) {
	benchmarkCompareSort(b, 1500)
}

func BenchmarkSortSize1500Radix4(b *testing.B) {
	benchmarkRadixSort(b, 4, 1500)
}

func BenchmarkSortSize1500Radix8(b *testing.B) {
	benchmarkRadixSort(b, 8, 1500)
}

func BenchmarkSortSize15000KRadix16(b *testing.B) {
	benchmarkRadixSort(b, 16, 1500)
}

const par = 8

func benchmarkRadixSort(b *testing.B, radix, size int) {
	sorter := makeSorter(radix, 0, size)
	b.SetParallelism(par)
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			sorter.Sort(makeRevertedSlice(size))
		}
	})
}

func benchmarkCompareSort(b *testing.B, size int) {
	sorter := makeSorter(8, size, size)
	b.SetParallelism(par)
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			sorter.Sort(makeRevertedSlice(size))
		}
	})
}
