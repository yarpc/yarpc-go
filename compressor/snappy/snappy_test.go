// Copyright (c) 2024 Uber Technologies, Inc.
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

package yarpcsnappy_test

import (
	"bytes"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc/compressor/snappy"
)

// This should be compressible:
var quote = "Now is the time for all good men to come to the aid of their country"
var input = []byte(quote + quote + quote)

func TestSnappy(t *testing.T) {
	buf := bytes.NewBuffer(nil)
	writer, err := yarpcsnappy.New().Compress(buf)
	require.NoError(t, err)

	_, err = writer.Write(input)
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	str, err := yarpcsnappy.New().Decompress(buf)
	require.NoError(t, err)

	output, err := ioutil.ReadAll(str)
	require.NoError(t, err)
	require.NoError(t, str.Close())

	assert.Equal(t, input, output)
}

func TestCompressionPooling(t *testing.T) {
	compressor := yarpcsnappy.New()
	for i := 0; i < 128; i++ {
		buf := bytes.NewBuffer(nil)
		writer, err := compressor.Compress(buf)
		require.NoError(t, err)

		_, err = writer.Write(input)
		require.NoError(t, err)
		require.NoError(t, writer.Close())

		str, err := compressor.Decompress(buf)
		require.NoError(t, err)

		output, err := ioutil.ReadAll(str)
		require.NoError(t, err)
		require.NoError(t, str.Close())

		assert.Equal(t, input, output)
	}
}

func TestSnappyName(t *testing.T) {
	assert.Equal(t, "snappy", yarpcsnappy.New().Name())
}

func TestSnappyCompression(t *testing.T) {
	compressor := yarpcsnappy.New()

	buf := bytes.NewBuffer(nil)
	writer, err := compressor.Compress(buf)
	require.NoError(t, err)

	_, err = writer.Write(input)
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	// Sanity check
	assert.True(t, buf.Len() < len(input), "one would think the compressed data would be smaller")
	t.Logf("decompress: %d\n", len(input))
	t.Logf("compressed: %d\n", buf.Len())
}
