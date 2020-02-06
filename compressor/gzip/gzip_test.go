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

package gzip_test

import (
	"bytes"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc/compressor/gzip"
)

func TestGzip(t *testing.T) {
	input := []byte("Now is the time for all good men to come to the aid of their country")

	buf := bytes.NewBuffer(nil)
	writer, err := gzip.Compressor{}.Compress(buf)
	require.NoError(t, err)

	_, err = writer.Write(input)
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	str, err := gzip.Compressor{}.Decompress(buf)
	require.NoError(t, err)

	output, err := ioutil.ReadAll(str)
	require.NoError(t, err)

	assert.Equal(t, input, output)
}

func TestGzipName(t *testing.T) {
	assert.Equal(t, "gzip", gzip.Compressor{}.Name())
}
