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

package yarpcgzip_test

import (
	"bytes"
	"compress/gzip"
	"io"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	yarpcgzip "go.uber.org/yarpc/compressor/gzip"
)

// This should be compressible:
var quote = "Now is the time for all good men to come to the aid of their country"
var input = []byte(quote + quote + quote)

func TestGzip(t *testing.T) {
	buf := bytes.NewBuffer(nil)
	writer, err := yarpcgzip.New().Compress(buf)
	require.NoError(t, err)

	_, err = writer.Write(input)
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	str, err := yarpcgzip.New().Decompress(buf)
	require.NoError(t, err)

	output, err := io.ReadAll(str)
	require.NoError(t, err)
	require.NoError(t, str.Close())

	assert.Equal(t, input, output)
}

func TestCompressionPooling(t *testing.T) {
	compressor := yarpcgzip.New()
	for i := 0; i < 128; i++ {
		buf := bytes.NewBuffer(nil)
		writer, err := compressor.Compress(buf)
		require.NoError(t, err)

		_, err = writer.Write(input)
		require.NoError(t, err)
		require.NoError(t, writer.Close())

		str, err := compressor.Decompress(buf)
		require.NoError(t, err)

		output, err := io.ReadAll(str)
		require.NoError(t, err)
		require.NoError(t, str.Close())

		assert.Equal(t, input, output)
	}
}

func TestEveryCompressionLevel(t *testing.T) {
	levels := []int{
		gzip.NoCompression,
		gzip.BestSpeed,
		gzip.BestCompression,
		gzip.DefaultCompression,
		gzip.HuffmanOnly,
	}

	for _, level := range levels {
		t.Run(strconv.Itoa(level), func(t *testing.T) {
			compressor := yarpcgzip.New(yarpcgzip.Level(level))

			buf := bytes.NewBuffer(nil)
			writer, err := compressor.Compress(buf)
			require.NoError(t, err)

			_, err = writer.Write(input)
			require.NoError(t, err)
			require.NoError(t, writer.Close())

			str, err := compressor.Decompress(buf)
			require.NoError(t, err)

			output, err := io.ReadAll(str)
			require.NoError(t, err)
			require.NoError(t, str.Close())

			assert.Equal(t, input, output)
		})
	}
}

func TestGzipName(t *testing.T) {
	assert.Equal(t, "gzip", yarpcgzip.New().Name())
}

func TestDecompressedSize(t *testing.T) {
	compressor := yarpcgzip.New(yarpcgzip.Level(gzip.BestCompression))

	buf := bytes.NewBuffer(nil)
	writer, err := compressor.Compress(buf)
	require.NoError(t, err)

	_, err = writer.Write(input)
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	assert.Equal(t, len(input), compressor.DecompressedSize(buf.Bytes()))

	// Sanity check
	assert.True(t, buf.Len() < len(input), "one would think the compressed data would be smaller")
	t.Logf("decompress: %d\n", len(input))
	t.Logf("compressed: %d\n", buf.Len())
}

func TestDecompressedSizeError(t *testing.T) {
	compressor := yarpcgzip.New(yarpcgzip.Level(gzip.BestCompression))
	assert.Equal(t, -1, compressor.DecompressedSize([]byte{}))
}
