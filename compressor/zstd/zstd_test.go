// Copyright (c) 2026 Uber Technologies, Inc.
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

package yarpczstd_test

import (
	"bytes"
	"io"
	"testing"

	"github.com/klauspost/compress/zstd"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	yarpczstd "go.uber.org/yarpc/compressor/zstd"
)

var quote = "Now is the time for all good men to come to the aid of their country"
var input = []byte(quote + quote + quote)

// TestCompressionPooling reuses a single Compressor across many sequential
// round-trips to exercise encoder/decoder pool recycling. Also sanity-checks
// that compressed output is smaller than the input.
func TestCompressionPooling(t *testing.T) {
	compressor := yarpczstd.New()
	for i := 0; i < 128; i++ {
		buf := bytes.NewBuffer(nil)
		writer, err := compressor.Compress(buf)
		require.NoError(t, err)

		_, err = writer.Write(input)
		require.NoError(t, err)
		require.NoError(t, writer.Close())

		assert.True(t, buf.Len() < len(input), "compressed data should be smaller than input")

		str, err := compressor.Decompress(buf)
		require.NoError(t, err)

		output, err := io.ReadAll(str)
		require.NoError(t, err)
		require.NoError(t, str.Close())

		assert.Equal(t, input, output)
	}
}

func TestEveryCompressionLevel(t *testing.T) {
	levels := []zstd.EncoderLevel{
		zstd.SpeedFastest,
		zstd.SpeedDefault,
		zstd.SpeedBetterCompression,
		zstd.SpeedBestCompression,
	}

	for _, level := range levels {
		t.Run(level.String(), func(t *testing.T) {
			compressor := yarpczstd.New(yarpczstd.Level(level))

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

func TestZstdName(t *testing.T) {
	assert.Equal(t, "zstd", yarpczstd.New().Name())
}

func TestRawEncoderDecoderOptions(t *testing.T) {
	compressor := yarpczstd.New(
		yarpczstd.Level(zstd.SpeedBestCompression),
		yarpczstd.EncoderOptions(zstd.WithWindowSize(1<<20)),
		yarpczstd.DecoderOptions(zstd.WithDecoderLowmem(true)),
	)

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

func TestDecompressPoolingAfterError(t *testing.T) {
	compressor := yarpczstd.New()

	// Warm up the pool with valid data.
	buf := bytes.NewBuffer(nil)
	writer, err := compressor.Compress(buf)
	require.NoError(t, err)
	_, err = writer.Write(input)
	require.NoError(t, err)
	require.NoError(t, writer.Close())
	validData := buf.Bytes()

	r, err := compressor.Decompress(bytes.NewReader(validData))
	require.NoError(t, err)
	io.Copy(io.Discard, r)
	r.Close()

	// Feed invalid data — the pooled decoder should handle the error.
	invalidData := []byte("this is not valid zstd data")
	r, err = compressor.Decompress(bytes.NewReader(invalidData))
	if err == nil {
		// zstd may defer the error to Read rather than Decompress.
		_, err = io.ReadAll(r)
		r.Close()
	}
	require.Error(t, err)

	// After the error, the pool should still produce working decoders.
	r, err = compressor.Decompress(bytes.NewReader(validData))
	require.NoError(t, err)
	output, err := io.ReadAll(r)
	require.NoError(t, err)
	require.NoError(t, r.Close())
	assert.Equal(t, input, output)
}

