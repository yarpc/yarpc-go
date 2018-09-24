// Copyright (c) 2018 Uber Technologies, Inc.
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

package yarpc

import "bytes"

// Buffer is used for requests and responses.
type Buffer struct {
	// TODO: incorporate buffer pooling (internal/bufferpool.Buffer)

	bytes.Buffer
}

// NewBufferBytes creates a new Buffer, using the bytes as its initial contents.
// The new Buffer takes ownership of bytes, and the caller should not use buf
// after this call.
func NewBufferBytes(b []byte) *Buffer {
	return &Buffer{*bytes.NewBuffer(b)}
}

// NewBufferString creates a new Buffer, using the string as its initial
// contents.
func NewBufferString(s string) *Buffer {
	return &Buffer{*bytes.NewBufferString(s)}
}
