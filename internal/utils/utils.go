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

package utils

import (
	"io"

	"go.uber.org/yarpc/internal/bufferpool"
)

// ReadBytes returns bytes from the reader
// It avoids double copy if body is an instance of buffer and returns the bytes directly
func ReadBytes(body io.Reader, buf *bufferpool.Buffer) ([]byte, error) {
	if buf, ok := body.(*bufferpool.Buffer); ok {
		return buf.Bytes(), nil
	}

	if _, err := buf.ReadFrom(body); err != nil {
		return nil, err
	}
	if closer, ok := body.(io.Closer); ok {
		if err := closer.Close(); err != nil {
			return nil, err
		}
	}
	return buf.Bytes(), nil
}
