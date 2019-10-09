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

package circus

const (
	_name                   = "circus"
	_noContextDeadlineError = "%q peer list can't wait for peer without a context deadline"
	_unavailableError       = "%q peer list timed out waiting for peer: %s"
	_noPeerError            = "%q peer list has no peer available"
)

// Programmatically initializing the internal state of a peer circus would be a
// waste of CPU.
// The structure is a fixed size allocation that always begins with four
// circular linked lists.
// Nodes within those lists refer to each other by index within their own
// array, not by pointers.
// Since the array is fixed to 256 entries, indicies can just be a byte wide.
// Consequently, we compile the entire initial state into the binary and copy
// it into each new list.

const (
	_no   = iota // not connected head node index
	_hi          // high concurrent requests head node index
	_lo          // low concurrent requests head node index
	_free        // available nodes head node index
)

// _zero is the initial state of a 256 node peer list table with four head nodes.
// Sufficiently advanced Vim is indistinguishable from metaprogramming.
var _zero nodes = [size]node{
	{prev: 0x00, next: 0x00}, // not connected (empty)
	{prev: 0x01, next: 0x01}, // low concurrent requests (empty)
	{prev: 0x02, next: 0x02}, // high concurrent requests (empty)
	{prev: 0xff, next: 0x04}, // free list loops in from end
	{prev: 0x03, next: 0x05},
	{prev: 0x04, next: 0x06},
	{prev: 0x05, next: 0x07},
	{prev: 0x06, next: 0x08},
	{prev: 0x07, next: 0x09},
	{prev: 0x08, next: 0x0a},
	{prev: 0x09, next: 0x0b},
	{prev: 0x0a, next: 0x0c},
	{prev: 0x0b, next: 0x0d},
	{prev: 0x0c, next: 0x0e},
	{prev: 0x0d, next: 0x0f},
	{prev: 0x0e, next: 0x10},
	{prev: 0x0f, next: 0x11},
	{prev: 0x10, next: 0x12},
	{prev: 0x11, next: 0x13},
	{prev: 0x12, next: 0x14},
	{prev: 0x13, next: 0x15},
	{prev: 0x14, next: 0x16},
	{prev: 0x15, next: 0x17},
	{prev: 0x16, next: 0x18},
	{prev: 0x17, next: 0x19},
	{prev: 0x18, next: 0x1a},
	{prev: 0x19, next: 0x1b},
	{prev: 0x1a, next: 0x1c},
	{prev: 0x1b, next: 0x1d},
	{prev: 0x1c, next: 0x1e},
	{prev: 0x1d, next: 0x1f},
	{prev: 0x1e, next: 0x20},
	{prev: 0x1f, next: 0x21},
	{prev: 0x20, next: 0x22},
	{prev: 0x21, next: 0x23},
	{prev: 0x22, next: 0x24},
	{prev: 0x23, next: 0x25},
	{prev: 0x24, next: 0x26},
	{prev: 0x25, next: 0x27},
	{prev: 0x26, next: 0x28},
	{prev: 0x27, next: 0x29},
	{prev: 0x28, next: 0x2a},
	{prev: 0x29, next: 0x2b},
	{prev: 0x2a, next: 0x2c},
	{prev: 0x2b, next: 0x2d},
	{prev: 0x2c, next: 0x2e},
	{prev: 0x2d, next: 0x2f},
	{prev: 0x2e, next: 0x30},
	{prev: 0x2f, next: 0x31},
	{prev: 0x30, next: 0x32},
	{prev: 0x31, next: 0x33},
	{prev: 0x32, next: 0x34},
	{prev: 0x33, next: 0x35},
	{prev: 0x34, next: 0x36},
	{prev: 0x35, next: 0x37},
	{prev: 0x36, next: 0x38},
	{prev: 0x37, next: 0x39},
	{prev: 0x38, next: 0x3a},
	{prev: 0x39, next: 0x3b},
	{prev: 0x3a, next: 0x3c},
	{prev: 0x3b, next: 0x3d},
	{prev: 0x3c, next: 0x3e},
	{prev: 0x3d, next: 0x3f},
	{prev: 0x3e, next: 0x40},
	{prev: 0x3f, next: 0x41},
	{prev: 0x40, next: 0x42},
	{prev: 0x41, next: 0x43},
	{prev: 0x42, next: 0x44},
	{prev: 0x43, next: 0x45},
	{prev: 0x44, next: 0x46},
	{prev: 0x45, next: 0x47},
	{prev: 0x46, next: 0x48},
	{prev: 0x47, next: 0x49},
	{prev: 0x48, next: 0x4a},
	{prev: 0x49, next: 0x4b},
	{prev: 0x4a, next: 0x4c},
	{prev: 0x4b, next: 0x4d},
	{prev: 0x4c, next: 0x4e},
	{prev: 0x4d, next: 0x4f},
	{prev: 0x4e, next: 0x50},
	{prev: 0x4f, next: 0x51},
	{prev: 0x50, next: 0x52},
	{prev: 0x51, next: 0x53},
	{prev: 0x52, next: 0x54},
	{prev: 0x53, next: 0x55},
	{prev: 0x54, next: 0x56},
	{prev: 0x55, next: 0x57},
	{prev: 0x56, next: 0x58},
	{prev: 0x57, next: 0x59},
	{prev: 0x58, next: 0x5a},
	{prev: 0x59, next: 0x5b},
	{prev: 0x5a, next: 0x5c},
	{prev: 0x5b, next: 0x5d},
	{prev: 0x5c, next: 0x5e},
	{prev: 0x5d, next: 0x5f},
	{prev: 0x5e, next: 0x60},
	{prev: 0x5f, next: 0x61},
	{prev: 0x60, next: 0x62},
	{prev: 0x61, next: 0x63},
	{prev: 0x62, next: 0x64},
	{prev: 0x63, next: 0x65},
	{prev: 0x64, next: 0x66},
	{prev: 0x65, next: 0x67},
	{prev: 0x66, next: 0x68},
	{prev: 0x67, next: 0x69},
	{prev: 0x68, next: 0x6a},
	{prev: 0x69, next: 0x6b},
	{prev: 0x6a, next: 0x6c},
	{prev: 0x6b, next: 0x6d},
	{prev: 0x6c, next: 0x6e},
	{prev: 0x6d, next: 0x6f},
	{prev: 0x6e, next: 0x70},
	{prev: 0x6f, next: 0x71},
	{prev: 0x70, next: 0x72},
	{prev: 0x71, next: 0x73},
	{prev: 0x72, next: 0x74},
	{prev: 0x73, next: 0x75},
	{prev: 0x74, next: 0x76},
	{prev: 0x75, next: 0x77},
	{prev: 0x76, next: 0x78},
	{prev: 0x77, next: 0x79},
	{prev: 0x78, next: 0x7a},
	{prev: 0x79, next: 0x7b},
	{prev: 0x7a, next: 0x7c},
	{prev: 0x7b, next: 0x7d},
	{prev: 0x7c, next: 0x7e},
	{prev: 0x7d, next: 0x7f},
	{prev: 0x7e, next: 0x80},
	{prev: 0x7f, next: 0x81},
	{prev: 0x80, next: 0x82},
	{prev: 0x81, next: 0x83},
	{prev: 0x82, next: 0x84},
	{prev: 0x83, next: 0x85},
	{prev: 0x84, next: 0x86},
	{prev: 0x85, next: 0x87},
	{prev: 0x86, next: 0x88},
	{prev: 0x87, next: 0x89},
	{prev: 0x88, next: 0x8a},
	{prev: 0x89, next: 0x8b},
	{prev: 0x8a, next: 0x8c},
	{prev: 0x8b, next: 0x8d},
	{prev: 0x8c, next: 0x8e},
	{prev: 0x8d, next: 0x8f},
	{prev: 0x8e, next: 0x90},
	{prev: 0x8f, next: 0x91},
	{prev: 0x90, next: 0x92},
	{prev: 0x91, next: 0x93},
	{prev: 0x92, next: 0x94},
	{prev: 0x93, next: 0x95},
	{prev: 0x94, next: 0x96},
	{prev: 0x95, next: 0x97},
	{prev: 0x96, next: 0x98},
	{prev: 0x97, next: 0x99},
	{prev: 0x98, next: 0x9a},
	{prev: 0x99, next: 0x9b},
	{prev: 0x9a, next: 0x9c},
	{prev: 0x9b, next: 0x9d},
	{prev: 0x9c, next: 0x9e},
	{prev: 0x9d, next: 0x9f},
	{prev: 0x9e, next: 0xa0},
	{prev: 0x9f, next: 0xa1},
	{prev: 0xa0, next: 0xa2},
	{prev: 0xa1, next: 0xa3},
	{prev: 0xa2, next: 0xa4},
	{prev: 0xa3, next: 0xa5},
	{prev: 0xa4, next: 0xa6},
	{prev: 0xa5, next: 0xa7},
	{prev: 0xa6, next: 0xa8},
	{prev: 0xa7, next: 0xa9},
	{prev: 0xa8, next: 0xaa},
	{prev: 0xa9, next: 0xab},
	{prev: 0xaa, next: 0xac},
	{prev: 0xab, next: 0xad},
	{prev: 0xac, next: 0xae},
	{prev: 0xad, next: 0xaf},
	{prev: 0xae, next: 0xb0},
	{prev: 0xaf, next: 0xb1},
	{prev: 0xb0, next: 0xb2},
	{prev: 0xb1, next: 0xb3},
	{prev: 0xb2, next: 0xb4},
	{prev: 0xb3, next: 0xb5},
	{prev: 0xb4, next: 0xb6},
	{prev: 0xb5, next: 0xb7},
	{prev: 0xb6, next: 0xb8},
	{prev: 0xb7, next: 0xb9},
	{prev: 0xb8, next: 0xba},
	{prev: 0xb9, next: 0xbb},
	{prev: 0xba, next: 0xbc},
	{prev: 0xbb, next: 0xbd},
	{prev: 0xbc, next: 0xbe},
	{prev: 0xbd, next: 0xbf},
	{prev: 0xbe, next: 0xc0},
	{prev: 0xbf, next: 0xc1},
	{prev: 0xc0, next: 0xc2},
	{prev: 0xc1, next: 0xc3},
	{prev: 0xc2, next: 0xc4},
	{prev: 0xc3, next: 0xc5},
	{prev: 0xc4, next: 0xc6},
	{prev: 0xc5, next: 0xc7},
	{prev: 0xc6, next: 0xc8},
	{prev: 0xc7, next: 0xc9},
	{prev: 0xc8, next: 0xca},
	{prev: 0xc9, next: 0xcb},
	{prev: 0xca, next: 0xcc},
	{prev: 0xcb, next: 0xcd},
	{prev: 0xcc, next: 0xce},
	{prev: 0xcd, next: 0xcf},
	{prev: 0xce, next: 0xd0},
	{prev: 0xcf, next: 0xd1},
	{prev: 0xd0, next: 0xd2},
	{prev: 0xd1, next: 0xd3},
	{prev: 0xd2, next: 0xd4},
	{prev: 0xd3, next: 0xd5},
	{prev: 0xd4, next: 0xd6},
	{prev: 0xd5, next: 0xd7},
	{prev: 0xd6, next: 0xd8},
	{prev: 0xd7, next: 0xd9},
	{prev: 0xd8, next: 0xda},
	{prev: 0xd9, next: 0xdb},
	{prev: 0xda, next: 0xdc},
	{prev: 0xdb, next: 0xdd},
	{prev: 0xdc, next: 0xde},
	{prev: 0xdd, next: 0xdf},
	{prev: 0xde, next: 0xe0},
	{prev: 0xdf, next: 0xe1},
	{prev: 0xe0, next: 0xe2},
	{prev: 0xe1, next: 0xe3},
	{prev: 0xe2, next: 0xe4},
	{prev: 0xe3, next: 0xe5},
	{prev: 0xe4, next: 0xe6},
	{prev: 0xe5, next: 0xe7},
	{prev: 0xe6, next: 0xe8},
	{prev: 0xe7, next: 0xe9},
	{prev: 0xe8, next: 0xea},
	{prev: 0xe9, next: 0xeb},
	{prev: 0xea, next: 0xec},
	{prev: 0xeb, next: 0xed},
	{prev: 0xec, next: 0xee},
	{prev: 0xed, next: 0xef},
	{prev: 0xee, next: 0xf0},
	{prev: 0xef, next: 0xf1},
	{prev: 0xf0, next: 0xf2},
	{prev: 0xf1, next: 0xf3},
	{prev: 0xf2, next: 0xf4},
	{prev: 0xf3, next: 0xf5},
	{prev: 0xf4, next: 0xf6},
	{prev: 0xf5, next: 0xf7},
	{prev: 0xf6, next: 0xf8},
	{prev: 0xf7, next: 0xf9},
	{prev: 0xf8, next: 0xfa},
	{prev: 0xf9, next: 0xfb},
	{prev: 0xfa, next: 0xfc},
	{prev: 0xfb, next: 0xfd},
	{prev: 0xfc, next: 0xfe},
	{prev: 0xfd, next: 0xff},
	{prev: 0xfe, next: 0x03}, // loop back to head of free list
}
