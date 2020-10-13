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

package utils_test

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/yarpc/internal/bufferpool"
	"go.uber.org/yarpc/internal/utils"
)

func TestReadBytes_WithoutCopy(t *testing.T) {
	body := bufferpool.Get()
	defer bufferpool.Put(body)

	_, err := body.Write([]byte("test"))

	buf := bufferpool.Get()
	defer bufferpool.Put(buf)

	bytes, err := utils.ReadBytes(body, buf)
	assert.NoError(t, err, "unexpected error in read bytes")
	assert.Equal(t, []byte("test"), bytes, "bytes didn't match")
	assert.Equal(t, []byte(nil), buf.Bytes(), "buffer bytes must be empty")
}

func TestReadBytes_WithCopy(t *testing.T) {
	buf := bufferpool.Get()
	defer bufferpool.Put(buf)
	bytes, err := utils.ReadBytes(bytes.NewReader([]byte("test")), buf)
	assert.NoError(t, err, "unexpected error in read bytes")
	assert.Equal(t, []byte("test"), bytes, "bytes didn't match")
	assert.Equal(t, []byte("test"), buf.Bytes(), "buffer bytes must be empty")
}

func TestReadBytes_HandleReadErr(t *testing.T) {
	buf := bufferpool.Get()
	defer bufferpool.Put(buf)

	reader := mockReader{readErr: errors.New("test error")}
	_, err := utils.ReadBytes(reader, buf)

	assert.Error(t, err, "expected error in read bytes")
}

func TestReadBytes_HandleCloseErr(t *testing.T) {
	buf := bufferpool.Get()
	defer bufferpool.Put(buf)

	reader := mockReader{closeErr: errors.New("close error")}
	_, err := utils.ReadBytes(reader, buf)

	assert.Error(t, err, "expected error in read bytes")
}

type mockReader struct {
	readErr  error
	closeErr error
}

func (r mockReader) Read(b []byte) (int, error) {
	if r.readErr == nil {
		return 0, io.EOF
	}
	return 0, r.readErr
}

func (r mockReader) Close() error {
	return r.closeErr
}
