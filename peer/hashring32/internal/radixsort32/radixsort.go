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

package radixsort32

import (
	"math/bits"
	"sort"
	"sync"
)

const (
	defaultRadix  = 8
	defaultMinLen = 1000
	defaultMaxLen = 200000
)

// RadixSorter32 is a radix sorter for sorting []uint32
type RadixSorter32 struct {
	// radix can be 1, 2, 4, 8, 16
	// cannot be 32 because the size of counting buffer overflows integer
	radix uint
	// for slice shorter than minLength it uses regular sort algorithm
	minLength int
	// for slice longer than maxLength it allocate buffer dynamically
	// which increases garbage collector's pressure =
	maxLength int

	// immutable after construction
	mask       uint32
	keyOffsets []uint

	// buffer used to reduce memory allocation
	buffer buf
}

// Option is an option for the radix sorter constructor.
type Option interface {
	apply(*RadixSorter32)
}

type noOption struct{}

func (noOption) apply(*RadixSorter32) {}

// Radix specifies the radix sorter's radix.
//
// Radix may be 1, 2, 4, 8, or 16.
func Radix(radix int) Option {
	for i := range validRadixes {
		if validRadixes[i] == radix {
			return radixOption{radix: radix}
		}
	}
	// fallback to default value
	return noOption{}
}

var validRadixes = []int{1, 2, 4, 8, 16}

type radixOption struct {
	radix int
}

func (o radixOption) apply(r *RadixSorter32) {
	r.radix = uint(o.radix)
}

// MinLen specifies minimum length of slice to use radix sort.
//
// The radix sorter will use quick sort for shorter slices.
func MinLen(minLen int) Option {
	return minLenOption{minLen: minLen}
}

type minLenOption struct {
	minLen int
}

func (o minLenOption) apply(r *RadixSorter32) {
	if o.minLen < 0 {
		o.minLen = 0
	}
	r.minLength = o.minLen
	if o.minLen > r.maxLength {
		r.maxLength = r.minLength
	}
}

// MaxLen is maximum length of slice that can utilize buffers in pool,
// would use dynamic buffer allocation for larger slices
func MaxLen(maxLen int) Option {
	return maxLenOption{maxLen: maxLen}
}

type maxLenOption struct {
	maxLen int
}

func (o maxLenOption) apply(r *RadixSorter32) {
	if o.maxLen >= r.minLength {
		r.maxLength = o.maxLen
	}
	r.maxLength = o.maxLen
	if o.maxLen < r.minLength {
		r.minLength = r.maxLength
	}
}

// New creates a radix sorter for sorting []uint32
func New(options ...Option) *RadixSorter32 {
	rs := &RadixSorter32{
		radix:     defaultRadix,
		minLength: defaultMinLen,
		maxLength: defaultMaxLen,
	}
	for _, opt := range options {
		opt.apply(rs)
	}
	// set key mask, e.g., 0xFF when radix is 8
	for i := 0; i < int(rs.radix); i++ {
		rs.mask = bits.RotateLeft32(rs.mask, 1) | 1
	}

	rs.buffer = newBuffer(rs.maxLength, int(rs.mask)+1)

	iterations := 32 / rs.radix
	rs.keyOffsets = make([]uint, iterations)
	for i := range rs.keyOffsets {
		rs.keyOffsets[i] = uint(i) * rs.radix
	}
	return rs
}

// Sort sorts a slice of type []uint32
func (r *RadixSorter32) Sort(origin []uint32) {
	if len(origin) <= r.minLength {
		sort.Slice(origin, func(i, j int) bool {
			return origin[i] < origin[j]
		})
		return
	}

	// Utilize buffer from pool or allocate slice when size too large
	length := len(origin)
	var buf, swap *[]uint32
	if len(origin) > r.maxLength {
		tmp := make([]uint32, length)
		swap = &tmp
	} else {
		buf = r.buffer.getSwap()
		t := (*buf)[:len(origin)]
		swap = &t
		defer r.buffer.putSwap(buf)
	}

	var key uint32

	offset := r.buffer.getCounters()
	defer r.buffer.putCounters(offset)

	counts := r.buffer.getCounters()
	defer r.buffer.putCounters(counts)

	for _, keyOffset := range r.keyOffsets {
		keyMask := uint32(r.mask << keyOffset)

		// counting
		for i := range *counts {
			(*counts)[i] = 0
		}

		for _, h := range origin {
			key = r.mask & ((h & keyMask) >> keyOffset)
			(*counts)[key]++
		}
		for i := range *offset {
			if i == 0 {
				(*offset)[0] = 0
				continue
			}
			(*offset)[i] = (*offset)[i-1] + (*counts)[i-1]
		}

		for _, h := range origin {
			key = r.mask & ((h & keyMask) >> keyOffset)
			(*swap)[(*offset)[key]] = h
			(*offset)[key]++
		}
		*swap, origin = origin, *swap
	}
}

// buffer
type buf struct {
	swapPool     sync.Pool
	countersPool sync.Pool
}

func (b *buf) getSwap() *[]uint32 {
	buf := *b.swapPool.Get().(*[]uint32)
	bb := buf[:]
	return &bb
}

func (b *buf) putSwap(buf *[]uint32) {
	b.swapPool.Put(buf)
}

func (b *buf) getCounters() *[]int {
	buf := *b.countersPool.Get().(*[]int)
	bb := buf[:]
	return &bb
}

func (b *buf) putCounters(buf *[]int) {
	b.countersPool.Put(buf)
}

func newBuffer(swapSize, countersSize int) buf {
	return buf{
		swapPool: sync.Pool{
			New: func() interface{} {
				b := make([]uint32, swapSize)
				return &b
			},
		},
		countersPool: sync.Pool{
			New: func() interface{} {
				b := make([]int, countersSize)
				return &b
			},
		},
	}
}
