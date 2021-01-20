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

// Package encoding contains helper functionality for encoding implementations.
package encoding

import (
	"go.uber.org/yarpc"
	"go.uber.org/yarpc/api/encoding"
)

// FromOptions converts a collection of yarpc.CallOptions to
// encoding.CallOptions. This is to allow the external API of
// yarpc.CallOptions to be compatible with the encoding.NewOutboundCall
// API without having to import api/encoding directly.
func FromOptions(opts []yarpc.CallOption) []encoding.CallOption {
	newOpts := make([]encoding.CallOption, len(opts))
	for i, o := range opts {
		newOpts[i] = encoding.CallOption(o)
	}
	return newOpts
}
